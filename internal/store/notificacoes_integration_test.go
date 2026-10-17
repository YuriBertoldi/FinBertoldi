//go:build integration

package store

import (
	"database/sql"
	"testing"
	"time"

	"fincontrol/internal/models"
)

func criarUsuario(t *testing.T, db *sql.DB, email string, fid int) int {
	t.Helper()
	id, err := CreateUserReturningID(db, "Teste", email, "hash", false, fid)
	if err != nil {
		t.Fatalf("criar usuário %s: %v", email, err)
	}
	return id
}

func idsPendentes(t *testing.T, db *sql.DB, hoje int, mes time.Time) map[int]bool {
	t.Helper()
	l, err := GetNotificacoesPendentes(db, hoje, mes)
	if err != nil {
		t.Fatalf("GetNotificacoesPendentes: %v", err)
	}
	ids := map[int]bool{}
	for _, p := range l {
		ids[p.User.ID] = true
	}
	return ids
}

func TestNotificacoes_SalvarELer(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)
	uid := criarUsuario(t, db, "a@teste.com", fA)

	if n := GetNotificacoes(db, uid); n.DiaEnvio != 1 || n.EmailMensal || n.MesSeguinte {
		t.Errorf("padrão sem cadastro = %+v; want dia 1, tudo desligado", n)
	}

	in := models.Notificacoes{EmailMensal: true, TelegramMensal: true, TelegramChatID: " 123 ", DiaEnvio: 23, MesSeguinte: true}
	if err := SaveNotificacoes(db, uid, in); err != nil {
		t.Fatalf("salvar: %v", err)
	}
	got := GetNotificacoes(db, uid)
	if !got.EmailMensal || !got.TelegramMensal || got.TelegramChatID != "123" || got.DiaEnvio != 23 || !got.MesSeguinte {
		t.Errorf("lido = %+v", got)
	}

	in.DiaEnvio = 0
	if err := SaveNotificacoes(db, uid, in); err == nil {
		t.Error("dia 0 deveria ser rejeitado pelo banco")
	}
}

func TestNotificacoes_Pendentes(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)
	dia23 := criarUsuario(t, db, "dia23@teste.com", fA)
	dia31 := criarUsuario(t, db, "dia31@teste.com", fA)
	desligado := criarUsuario(t, db, "off@teste.com", fA)
	inativo := criarUsuario(t, db, "inativo@teste.com", fA)

	SaveNotificacoes(db, dia23, models.Notificacoes{EmailMensal: true, DiaEnvio: 23})
	SaveNotificacoes(db, dia31, models.Notificacoes{TelegramMensal: true, DiaEnvio: 31})
	SaveNotificacoes(db, desligado, models.Notificacoes{DiaEnvio: 23})
	SaveNotificacoes(db, inativo, models.Notificacoes{EmailMensal: true, DiaEnvio: 23})
	ToggleUserAtivo(db, inativo)

	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	novembro := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	if got := idsPendentes(t, db, 23, outubro); !got[dia23] || got[dia31] || got[desligado] || got[inativo] || len(got) != 1 {
		t.Errorf("dia 23/out: pendentes = %v; want só %d", got, dia23)
	}
	if got := idsPendentes(t, db, 22, outubro); len(got) != 0 {
		t.Errorf("dia 22/out: ninguém deveria receber, got %v", got)
	}
	// novembro tem 30 dias: quem escolheu 31 recebe no dia 30
	if got := idsPendentes(t, db, 30, novembro); !got[dia31] || len(got) != 1 {
		t.Errorf("30/nov (último dia): pendentes = %v; want só %d", got, dia31)
	}
	// em outubro (31 dias) o dia 30 não é o último: quem escolheu 31 ainda não recebe
	if got := idsPendentes(t, db, 30, outubro); got[dia31] {
		t.Error("30/out não é o último dia: dia 31 não deveria receber")
	}

	// depois de enviado no mês, não repete; no mês seguinte volta a valer
	MarcarEnvioMensal(db, dia23, outubro)
	if got := idsPendentes(t, db, 23, outubro); got[dia23] {
		t.Error("já enviado em outubro: não deveria repetir")
	}
	if got := idsPendentes(t, db, 23, novembro); !got[dia23] {
		t.Error("novembro: deveria voltar a receber")
	}
	if n := GetNotificacoes(db, dia23); n.UltimoEnvioFmt() != "10/2026" {
		t.Errorf("último envio = %q; want 10/2026", n.UltimoEnvioFmt())
	}
}

func TestDashboardLayout(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)
	uid := criarUsuario(t, db, "dash@teste.com", fA)

	padrao := GetDashboardLayout(db, uid)
	if len(padrao) != len(widgetCatalogo) || padrao[0].Key != "resumo" {
		t.Fatalf("layout padrão inesperado: %+v", padrao)
	}

	// usuário sobe feriados para a esquerda e esconde a sobra
	novo := []models.DashboardWidget{
		{Key: "resumo", Visivel: true, Coluna: "topo"},
		{Key: "feriados", Visivel: true, Coluna: "esquerda"},
		{Key: "grafico_sobra", Visivel: false, Coluna: "esquerda"},
	}
	if err := SaveDashboardLayout(db, uid, novo); err != nil {
		t.Fatalf("salvar layout: %v", err)
	}
	got := GetDashboardLayout(db, uid)
	if got[1].Key != "feriados" || got[1].Coluna != "esquerda" {
		t.Errorf("feriados deveria vir logo após o resumo, na esquerda: %+v", got[1])
	}
	if got[2].Key != "grafico_sobra" || got[2].Visivel {
		t.Errorf("grafico_sobra deveria estar oculto: %+v", got[2])
	}
	// widgets que não estavam salvos entram no fim com o padrão
	if len(got) != len(widgetCatalogo) {
		t.Errorf("len = %d; want %d (widgets não salvos devem aparecer)", len(got), len(widgetCatalogo))
	}

	// coluna inválida gravada no banco é ignorada
	db.Exec(`UPDATE dashboard_widgets SET coluna='lateral' WHERE user_id=$1 AND widget_key='feriados'`, uid)
	for _, w := range GetDashboardLayout(db, uid) {
		if w.Key == "feriados" && w.Coluna != "direita" {
			t.Errorf("coluna inválida deveria voltar ao padrão (direita), got %q", w.Coluna)
		}
	}

	// compatibilidade: visibilidade antiga continua funcionando
	if vis := GetDashboardWidgets(db, uid); vis["grafico_sobra"] || !vis["feriados"] {
		t.Errorf("GetDashboardWidgets = %v", vis)
	}
}
