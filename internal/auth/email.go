package auth

import (
	"fmt"
	"log"
	"mime"
	"net/smtp"
	"os"
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

// SMTPConfigurado indica se as variáveis SMTP mínimas estão definidas.
func SMTPConfigurado() bool {
	return os.Getenv("SMTP_HOST") != "" && os.Getenv("SMTP_FROM") != ""
}

// SendEmail envia um e-mail HTML usando as variáveis SMTP_*.
func SendEmail(to, subject, htmlBody string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	from := os.Getenv("SMTP_FROM")
	if !SMTPConfigurado() {
		return fmt.Errorf("SMTP não configurado")
	}
	if port == "" {
		port = "587"
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, mime.QEncoding.Encode("utf-8", subject), htmlBody)

	auth := smtp.PlainAuth("", os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASS"), host)
	return smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(msg))
}
