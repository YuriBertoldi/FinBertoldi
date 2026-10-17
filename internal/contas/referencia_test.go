package contas

import (
	"testing"
	"time"

	"fincontrol/internal/models"
)

func TestMesReferencia(t *testing.T) {
	cases := []struct {
		envio    time.Time
		seguinte bool
		want     string
	}{
		{time.Date(2026, 10, 23, 9, 0, 0, 0, time.Local), false, "2026-10"},
		{time.Date(2026, 10, 23, 9, 0, 0, 0, time.Local), true, "2026-11"},
		{time.Date(2026, 12, 23, 9, 0, 0, 0, time.Local), true, "2027-01"},
		{time.Date(2027, 1, 31, 9, 0, 0, 0, time.Local), true, "2027-02"},
	}
	for _, c := range cases {
		got := models.Notificacoes{MesSeguinte: c.seguinte}.MesReferencia(c.envio).Format("2006-01")
		if got != c.want {
			t.Errorf("envio %s seguinte=%v: got %s want %s", c.envio.Format("02/01/2006"), c.seguinte, got, c.want)
		}
	}
}
