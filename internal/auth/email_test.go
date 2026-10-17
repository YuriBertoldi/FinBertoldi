package auth

import "testing"

func TestLerSMTPPadroesGmail(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("SMTP_USER", "finbertoldi@gmail.com")
	t.Setenv("SMTP_PASS", "abcd efgh ijkl mnop")

	c := lerSMTP()
	if c.host != "smtp.gmail.com" || c.port != "587" {
		t.Errorf("host/porta = %s:%s", c.host, c.port)
	}
	if c.from != "FinBertoldi <finbertoldi@gmail.com>" {
		t.Errorf("from = %q", c.from)
	}
	if c.pass != "abcdefghijklmnop" {
		t.Errorf("senha de app deveria vir sem espaços: %q", c.pass)
	}
	if !SMTPConfigurado() {
		t.Error("deveria estar configurado só com usuário e senha do Gmail")
	}
}

func TestLerSMTPSemConfig(t *testing.T) {
	for _, k := range []string{"SMTP_HOST", "SMTP_PORT", "SMTP_FROM", "SMTP_USER", "SMTP_PASS"} {
		t.Setenv(k, "")
	}
	if SMTPConfigurado() {
		t.Error("sem variáveis não deveria estar configurado")
	}
}

func TestLerSMTPExplicito(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp-relay.brevo.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USER", "login@smtp-brevo.com")
	t.Setenv("SMTP_PASS", "x")
	t.Setenv("SMTP_FROM", "Contas <contas@exemplo.com>")
	c := lerSMTP()
	if c.host != "smtp-relay.brevo.com" || c.port != "465" || c.from != "Contas <contas@exemplo.com>" {
		t.Errorf("config explícita não respeitada: %+v", c)
	}
}
