package auth

import (
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
)

// SendResetEmail envia email com link de recuperação de senha.
// Se SMTP não estiver configurado, loga o link no console.
func SendResetEmail(to, token, baseURL string) error {
	link := baseURL + "/reset-senha?token=" + token

	if !SMTPConfigurado() {
		log.Printf("[password-reset] SMTP nao configurado. Link de recuperacao: %s", link)
		return nil
	}

	subject := "Recuperacao de Senha - FinBertoldi"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:sans-serif;background:#1a1a2e;color:#e0e0e0;padding:2rem">
<div style="max-width:500px;margin:0 auto;background:#16213e;padding:2rem;border-radius:12px">
<h2 style="color:#27ae60">FinBertoldi</h2>
<p>Voce solicitou a recuperacao de senha.</p>
<p>Clique no botao abaixo para criar uma nova senha:</p>
<a href="%s" style="display:inline-block;padding:12px 24px;background:#27ae60;color:#fff;text-decoration:none;border-radius:8px;font-weight:bold;margin:1rem 0">Redefinir Senha</a>
<p style="font-size:.85rem;color:#999">Este link expira em 1 hora.</p>
<p style="font-size:.8rem;color:#666">Se voce nao solicitou essa recuperacao, ignore este email.</p>
</div>
</body></html>`, link)

	return SendEmail(to, subject, body)
}

// smtpConfig é a configuração de envio já com os padrões embutidos.
type smtpConfig struct {
	host, port, user, pass string
	from                   string // cabeçalho From (pode ter nome: "FinBertoldi <x@gmail.com>")
}

// lerSMTP lê as variáveis SMTP_*. Para contas Gmail basta SMTP_USER e SMTP_PASS
// (senha de app): servidor, porta e remetente são preenchidos automaticamente.
func lerSMTP() smtpConfig {
	c := smtpConfig{
		host: strings.TrimSpace(os.Getenv("SMTP_HOST")),
		port: strings.TrimSpace(os.Getenv("SMTP_PORT")),
		user: strings.TrimSpace(os.Getenv("SMTP_USER")),
		pass: strings.ReplaceAll(os.Getenv("SMTP_PASS"), " ", ""), // senha de app do Google vem com espaços
		from: strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
	dominio := strings.ToLower(c.user[strings.LastIndex(c.user, "@")+1:])
	if c.host == "" && (dominio == "gmail.com" || dominio == "googlemail.com") {
		c.host = "smtp.gmail.com"
	}
	if c.port == "" {
		c.port = "587"
	}
	if c.from == "" && c.user != "" {
		c.from = "FinBertoldi <" + c.user + ">"
	}
	return c
}

// SMTPConfigurado indica se há servidor e remetente para enviar e-mails.
func SMTPConfigurado() bool {
	c := lerSMTP()
	return c.host != "" && c.from != ""
}

// SendEmail envia um e-mail HTML. Porta 465 usa TLS direto; as demais, STARTTLS.
func SendEmail(to, subject, htmlBody string) error {
	c := lerSMTP()
	if c.host == "" || c.from == "" {
		return fmt.Errorf("SMTP não configurado")
	}
	remetente, err := mail.ParseAddress(c.from)
	if err != nil {
		return fmt.Errorf("SMTP_FROM inválido: %w", err)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		remetente.String(), to, mime.QEncoding.Encode("utf-8", subject), htmlBody)

	var auth smtp.Auth
	if c.user != "" {
		auth = smtp.PlainAuth("", c.user, c.pass, c.host)
	}
	addr := net.JoinHostPort(c.host, c.port)
	if c.port != "465" {
		return smtp.SendMail(addr, auth, remetente.Address, []string{to}, []byte(msg))
	}

	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: c.host})
	if err != nil {
		return err
	}
	cl, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return err
	}
	defer cl.Close()
	if auth != nil {
		if err := cl.Auth(auth); err != nil {
			return err
		}
	}
	if err := cl.Mail(remetente.Address); err != nil {
		return err
	}
	if err := cl.Rcpt(to); err != nil {
		return err
	}
	w, err := cl.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return cl.Quit()
}
