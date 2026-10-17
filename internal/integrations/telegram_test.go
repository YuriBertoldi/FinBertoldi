package integrations

import "testing"

func TestValidarTokenTelegram(t *testing.T) {
	validos := []string{"123456789:AAEqVGlDSGflf40M0NHOrb-YtGUZbaBYwoX"}
	invalidos := []string{"", "AAEqVGlDSGflf40M0NHOrb-YtGUZbaBYwoX", "123:abc", "123456789 AAEqVGlDSGflf40M0NHOrb-YtGUZbaBYwoX"}
	for _, v := range validos {
		if err := ValidarTokenTelegram(v); err != nil {
			t.Errorf("%q deveria ser válido: %v", v, err)
		}
	}
	for _, v := range invalidos {
		if ValidarTokenTelegram(v) == nil {
			t.Errorf("%q deveria ser inválido", v)
		}
	}
}

func TestExplicarErroTelegram(t *testing.T) {
	if got := explicarErroTelegram(400, "Bad Request: chat not found"); got == "" || got[:4] != "chat" {
		t.Errorf("chat not found: %q", got)
	}
	if got := explicarErroTelegram(404, "Not Found"); got[:5] != "token" {
		t.Errorf("404: %q", got)
	}
}
