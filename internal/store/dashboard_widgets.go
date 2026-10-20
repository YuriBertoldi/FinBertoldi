package store

import (
	"database/sql"
	"fincontrol/internal/models"
)

var defaultWidgets = []string{
	"resumo", "grafico_receitas_despesas", "grafico_sobra", "grafico_investimentos",
	"detalhes_mes", "parcelamentos", "reserva_em", "indicadores", "feriados",
}

// widgetCatalogo define rótulo e coluna padrão de cada widget, na ordem padrão.
var widgetCatalogo = []models.DashboardWidget{
	{Key: "resumo", Label: "💵 Resumo (cards)", Visivel: true, Coluna: "topo"},
	{Key: "grafico_receitas_despesas", Label: "📊 Receitas vs Despesas", Visivel: true, Coluna: "esquerda"},
	{Key: "grafico_sobra", Label: "📈 Sobra Mensal", Visivel: true, Coluna: "esquerda"},
	{Key: "grafico_investimentos", Label: "💰 Investimentos", Visivel: true, Coluna: "esquerda"},
	{Key: "detalhes_mes", Label: "📋 Detalhes do mês", Visivel: true, Coluna: "esquerda"},
	{Key: "parcelamentos", Label: "💳 Parcelamentos", Visivel: true, Coluna: "direita"},
	{Key: "reserva_em", Label: "🛡️ Reserva de Emergência", Visivel: true, Coluna: "direita"},
	{Key: "indicadores", Label: "📡 Indicadores", Visivel: true, Coluna: "direita"},
	{Key: "feriados", Label: "📅 Feriados", Visivel: true, Coluna: "direita"},
}

// GetDashboardLayout devolve todos os widgets na ordem salva pelo usuário
// (widgets novos, ainda não salvos, entram no fim com o padrão).
func GetDashboardLayout(db *sql.DB, userID int) []models.DashboardWidget {
	catalogo := map[string]models.DashboardWidget{}
	for _, w := range widgetCatalogo {
		catalogo[w.Key] = w
	}
	var layout []models.DashboardWidget
	usados := map[string]bool{}
	rows, err := db.Query(`SELECT widget_key, visivel, coluna FROM dashboard_widgets
		WHERE user_id=$1 ORDER BY posicao, widget_key`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key, coluna string
			var vis bool
			if rows.Scan(&key, &vis, &coluna) != nil {
				continue
			}
			w, ok := catalogo[key]
			if !ok || usados[key] {
				continue
			}
			w.Visivel = vis
			if w.Coluna != "topo" && (coluna == "esquerda" || coluna == "direita") {
				w.Coluna = coluna
			}
			layout = append(layout, w)
			usados[key] = true
		}
	}
	for _, w := range widgetCatalogo {
		if !usados[w.Key] {
			layout = append(layout, w)
		}
	}
	return OrdenarTopo(layout)
}

// OrdenarTopo mantém os widgets fixos no topo ("resumo") sempre em primeiro.
func OrdenarTopo(layout []models.DashboardWidget) []models.DashboardWidget {
	var topo, resto []models.DashboardWidget
	for _, w := range layout {
		if w.Coluna == "topo" {
			topo = append(topo, w)
		} else {
			resto = append(resto, w)
		}
	}
	return append(topo, resto...)
}

// SaveDashboardLayout grava ordem, coluna e visibilidade dos widgets do usuário.
func SaveDashboardLayout(db *sql.DB, userID int, layout []models.DashboardWidget) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM dashboard_widgets WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for pos, w := range layout {
		if _, err := tx.Exec(`INSERT INTO dashboard_widgets (user_id, widget_key, visivel, posicao, coluna)
			VALUES ($1,$2,$3,$4,$5)`, userID, w.Key, w.Visivel, pos, w.Coluna); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func GetDashboardWidgets(db *sql.DB, userID int) map[string]bool {
	result := make(map[string]bool)
	// Default: all visible
	for _, w := range defaultWidgets {
		result[w] = true
	}

	rows, err := db.Query(`SELECT widget_key, visivel FROM dashboard_widgets WHERE user_id=$1`, userID)
	if err != nil {
		return result
	}
	defer rows.Close()

	hasRows := false
	for rows.Next() {
		hasRows = true
		var key string
		var vis bool
		rows.Scan(&key, &vis)
		result[key] = vis
	}

	if !hasRows {
		return result // No config saved yet, show all
	}
	return result
}

func SaveDashboardWidgets(db *sql.DB, userID int, widgets map[string]bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec(`DELETE FROM dashboard_widgets WHERE user_id=$1`, userID)
	pos := 0
	for _, w := range widgetCatalogo {
		tx.Exec(`INSERT INTO dashboard_widgets (user_id, widget_key, visivel, posicao, coluna) VALUES ($1,$2,$3,$4,$5)`,
			userID, w.Key, widgets[w.Key], pos, w.Coluna)
		pos++
	}
	return tx.Commit()
}

func GetWidgetKeys() []string {
	return defaultWidgets
}
