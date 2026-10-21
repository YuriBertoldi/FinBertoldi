package contas

import (
	"database/sql"
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"
	"time"

	"fincontrol/internal/auth"
	"fincontrol/internal/integrations"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

// Resultado do envio em um canal.
type Resultado struct {
	Canal   string // "E-mail" ou "Telegram"
	Destino string // para onde foi enviado (quando deu certo)
	Erro    string // vazio = enviado
}

// Enviar manda as contas do mês do usuário pelos canais pedidos.
func Enviar(db *sql.DB, u *models.User, n models.Notificacoes, mes time.Time, email, telegram bool) []Resultado {
	assunto := "Contas de " + MesDisplay(mes) + " - FinBertoldi"
	return enviarMensagem(db, u, n, assunto, Texto(db, u.FamilyID, mes), email, telegram)
}

// enviarMensagem manda um texto no formato do WhatsApp (``` e *) pelos canais
// pedidos: por e-mail para o endereco do usuario e pelo bot do Telegram da
// familia, no chat do usuario (ou no da familia, se ele nao informou o dele).
func enviarMensagem(db *sql.DB, u *models.User, n models.Notificacoes, assunto, txt string, email, telegram bool) []Resultado {
	var res []Resultado

	if email {
		r := Resultado{Canal: "E-mail", Destino: u.Email}
		switch {
		case !auth.SMTPConfigurado():
			r.Erro = "envio de e-mail não configurado no servidor (SMTP)"
		default:
			if err := auth.SendEmail(u.Email, assunto, EmailHTML(txt)); err != nil {
				r.Erro = "falha ao enviar: " + err.Error()
			}
		}
		res = append(res, r)
	}

	if telegram {
		r := Resultado{Canal: "Telegram"}
		token := store.GetIntegracaoKV(db, u.FamilyID, "telegram", "bot_token")
		chat := n.TelegramChatID
		if chat == "" {
			chat = store.GetIntegracaoKV(db, u.FamilyID, "telegram", "chat_id")
		}
		switch {
		case token == "":
			r.Erro = "bot do Telegram não configurado (Integrações)"
		case chat == "":
			r.Erro = "informe seu Chat ID em Notificações"
		default:
			r.Destino = "chat " + chat
			if err := integrations.SendTelegramMessage(token, chat, TelegramHTML(txt)); err != nil {
				r.Erro = "falha ao enviar: " + err.Error()
			}
		}
		res = append(res, r)
	}
	return res
}

var reNegrito = regexp.MustCompile(`\*([^*\n]+)\*`)

// converte o texto do WhatsApp (``` e *) em HTML; pre recebe os atributos dados.
func waParaHTML(txt, preAttrs, negrito string) string {
	var b strings.Builder
	for i, parte := range strings.Split(txt, "```") {
		if i%2 == 1 {
			fmt.Fprintf(&b, "<pre%s>%s</pre>", preAttrs, html.EscapeString(strings.Trim(parte, "\n")))
			continue
		}
		p := reNegrito.ReplaceAllString(html.EscapeString(strings.Trim(parte, "\n")), "<"+negrito+">$1</"+negrito+">")
		b.WriteString(p)
	}
	return b.String()
}

// TelegramHTML converte para o subconjunto de HTML aceito pelo Telegram (<b>, <pre>).
func TelegramHTML(txt string) string {
	s := waParaHTML(txt, "", "b")
	return strings.ReplaceAll(s, "</pre><b>", "</pre>\n\n<b>")
}

// EmailHTML monta o corpo do e-mail no mesmo visual escuro do e-mail de recuperação de senha.
func EmailHTML(txt string) string {
	corpo := waParaHTML(txt,
		` style="font-family:Consolas,Menlo,monospace;font-size:13px;background:#0f1419;color:#e6e9ed;padding:12px;border-radius:8px;margin:6px 0 16px"`,
		"strong")
	corpo = strings.ReplaceAll(corpo, "\n", "<br>")
	return `<!DOCTYPE html>
<html><body style="font-family:sans-serif;background:#1a1a2e;color:#e0e0e0;padding:2rem">
<div style="max-width:520px;margin:0 auto;background:#16213e;padding:2rem;border-radius:12px;line-height:1.5">
<h2 style="color:#27ae60;margin-top:0">FinBertoldi</h2>
` + corpo + `
<p style="font-size:.8rem;color:#888;margin-top:1.5rem">Para mudar o envio automático, acesse Notificações no FinBertoldi.</p>
</div>
</body></html>`
}

// StartScheduler verifica de hora em hora quem deve receber o envio mensal hoje
// e quais orcamentos passaram de 80% ou de 100%.
func StartScheduler(db *sql.DB) {
	go func() {
		time.Sleep(15 * time.Second)
		for {
			agora := time.Now()
			EnviarMensais(db, agora)
			EnviarAlertasDeOrcamento(db, agora)
			time.Sleep(time.Hour)
		}
	}()
}

// EnviarMensais envia as contas do mês para quem tem envio automático no dia de hoje
// (a partir das 8h) e ainda não recebeu neste mês.
func EnviarMensais(db *sql.DB, agora time.Time) {
	if agora.Hour() < 8 {
		return
	}
	mes := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.UTC)
	pend, err := store.GetNotificacoesPendentes(db, agora.Day(), mes)
	if err != nil {
		log.Printf("[notificacoes] erro ao buscar pendentes: %v", err)
		return
	}
	for _, p := range pend {
		res := Enviar(db, &p.User, p.Notificacoes, p.Notificacoes.MesReferencia(agora), p.Notificacoes.EmailMensal, p.Notificacoes.TelegramMensal)
		for _, r := range res {
			if r.Erro != "" {
				log.Printf("[notificacoes] user=%d %s: %s", p.User.ID, r.Canal, r.Erro)
			}
		}
		// Marca mesmo com erro para não reenviar a cada hora; o usuário pode enviar manualmente.
		store.MarcarEnvioMensal(db, p.User.ID, mes)
	}
}
