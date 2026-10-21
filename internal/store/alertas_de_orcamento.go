package store

import (
	"database/sql"
	"time"

	"fincontrol/internal/models"
)

// UsuariosComAlertaDeOrcamento lista os usuarios ativos que pediram alerta de
// orcamento por pelo menos um canal.
func UsuariosComAlertaDeOrcamento(db *sql.DB) ([]models.NotificacaoPendente, error) {
	linhas, err := db.Query(`
		SELECT u.id, u.nome, u.email, u.family_id,
		       n.telegram_chat_id, n.alerta_orcamento_email, n.alerta_orcamento_telegram
		FROM user_notificacoes n
		JOIN users u ON u.id = n.user_id
		WHERE u.ativo AND (n.alerta_orcamento_email OR n.alerta_orcamento_telegram)
		ORDER BY u.family_id, u.id`)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	var usuarios []models.NotificacaoPendente
	for linhas.Next() {
		var usuario models.NotificacaoPendente
		if err := linhas.Scan(&usuario.User.ID, &usuario.User.Nome, &usuario.User.Email, &usuario.User.FamilyID,
			&usuario.Notificacoes.TelegramChatID, &usuario.Notificacoes.AlertaOrcamentoEmail,
			&usuario.Notificacoes.AlertaOrcamentoTelegram); err != nil {
			return nil, err
		}
		usuarios = append(usuarios, usuario)
	}
	return usuarios, linhas.Err()
}

// AlertasDeOrcamentoEnviados devolve, por categoria, as faixas ja avisadas ao
// usuario no mes.
func AlertasDeOrcamentoEnviados(db *sql.DB, userID int, mes time.Time) (map[int]map[string]bool, error) {
	linhas, err := db.Query(`SELECT categoria_id, faixa FROM alertas_de_orcamento_enviados
		WHERE user_id=$1 AND mes=$2`, userID, mes)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	enviados := map[int]map[string]bool{}
	for linhas.Next() {
		var categoriaID int
		var faixa string
		if err := linhas.Scan(&categoriaID, &faixa); err != nil {
			return nil, err
		}
		if enviados[categoriaID] == nil {
			enviados[categoriaID] = map[string]bool{}
		}
		enviados[categoriaID][faixa] = true
	}
	return enviados, linhas.Err()
}

func RegistrarAlertaDeOrcamentoEnviado(db *sql.DB, userID, categoriaID int, mes time.Time, faixa string) error {
	_, err := db.Exec(`INSERT INTO alertas_de_orcamento_enviados (user_id, categoria_id, mes, faixa)
		VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, userID, categoriaID, mes, faixa)
	return err
}
