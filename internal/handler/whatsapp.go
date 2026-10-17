package handler

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"fincontrol/internal/auth"
	"fincontrol/internal/contas"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

// HandlePagamentosWhatsApp devolve as contas do mês como texto pronto para colar no WhatsApp.
func HandlePagamentosWhatsApp(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mes, _ := mesFromRequest(r)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, contas.Texto(db, auth.CurrentUser(r).FamilyID, mes))
	}
}

// HandlePagamentosEnviar envia as contas do mês para o e-mail e o Telegram do usuário logado.
// Responde um fragmento HTML (HTMX) com o resultado de cada canal.
func HandlePagamentosEnviar(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mes, _ := mesFromRequest(r)
		u := auth.CurrentUser(r)
		res := contas.Enviar(db, u, store.GetNotificacoes(db, u.ID), mes, true, true)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, resultadoEnvioHTML(res))
	}
}

func resultadoEnvioHTML(res []contas.Resultado) string {
	out := `<ul class="wa-resultado">`
	for _, r := range res {
		ico := "✅"
		if r.Erro != "" {
			ico = "⚠️"
		}
		msg := r.Destino
		if r.Erro != "" {
			msg = r.Erro
		}
		out += fmt.Sprintf(`<li>%s <strong>%s</strong> — %s</li>`, ico, r.Canal, html.EscapeString(msg))
	}
	return out + `</ul>`
}

// --- Notificações do usuário ---

func HandleNotificacoes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		n := store.GetNotificacoes(db, u.ID)
		mesEnvio := n.MesReferencia(time.Now())
		render(w, "notificacoes", models.NotificacoesPage{
			BasePage:        bp(db, r, "notificacoes", "Notificações"),
			Notificacoes:    n,
			MesEnvio:        mesEnvio.Format("2006-01"),
			MesEnvioDisplay: contas.MesDisplay(mesEnvio),
			TelegramBot:     store.GetIntegracaoKV(db, fid, "telegram", "bot_token") != "",
			TelegramChatFam: store.GetIntegracaoKV(db, fid, "telegram", "chat_id"),
			SMTPConfigurado: auth.SMTPConfigurado(),
			Msg:             r.URL.Query().Get("ok"),
			Erro:            r.URL.Query().Get("erro"),
		})
	}
}

func HandleSaveNotificacoes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		u := auth.CurrentUser(r)
		atual := store.GetNotificacoes(db, u.ID)
		dia, _ := strconv.Atoi(r.FormValue("dia_envio"))
		if dia < 1 || dia > 31 {
			dia = 1
		}
		atual.EmailMensal = r.FormValue("email_mensal") == "on"
		atual.TelegramMensal = r.FormValue("telegram_mensal") == "on"
		atual.TelegramChatID = r.FormValue("telegram_chat_id")
		atual.DiaEnvio = dia
		atual.MesSeguinte = r.FormValue("mes_referencia") == "seguinte"
		if err := store.SaveNotificacoes(db, u.ID, atual); err != nil {
			http.Redirect(w, r, "/notificacoes?erro=Erro ao salvar: "+err.Error(), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/notificacoes?ok=Preferências salvas", http.StatusSeeOther)
	}
}
