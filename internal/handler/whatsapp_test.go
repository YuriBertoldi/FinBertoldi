package handler

import (
	"strings"
	"testing"

	"fincontrol/internal/contas"
	"fincontrol/internal/models"
)

func TestResultadoEnvioHTML(t *testing.T) {
	h := resultadoEnvioHTML([]contas.Resultado{
		{Canal: "E-mail", Destino: "a@b.com"},
		{Canal: "Telegram", Erro: "chat <script> não encontrado"},
	})
	if !strings.Contains(h, "✅ <strong>E-mail</strong> — a@b.com") {
		t.Errorf("sucesso deve mostrar o destino: %s", h)
	}
	if !strings.Contains(h, "⚠️ <strong>Telegram</strong>") || strings.Contains(h, "<script>") {
		t.Errorf("erro deve aparecer com aviso e escapado: %s", h)
	}
}

func layoutPadrao() []models.DashboardWidget {
	return []models.DashboardWidget{
		{Key: "resumo", Visivel: true, Coluna: "topo"},
		{Key: "grafico_sobra", Visivel: true, Coluna: "esquerda"},
		{Key: "reserva_em", Visivel: true, Coluna: "direita"},
		{Key: "feriados", Visivel: true, Coluna: "direita"},
	}
}

func chaves(l []models.DashboardWidget) string {
	var s []string
	for _, w := range l {
		s = append(s, w.Key+":"+w.Coluna)
	}
	return strings.Join(s, ",")
}

func TestMontarLayoutWidgets(t *testing.T) {
	colunas := map[string]string{
		"coluna_feriados":      "esquerda",
		"coluna_reserva_em":    "esquerda",
		"coluna_resumo":        "direita", // resumo é fixo: deve ser ignorado
		"coluna_grafico_sobra": "lateral", // valor inválido: mantém a atual
	}
	got := montarLayoutWidgets(layoutPadrao(),
		[]string{"feriados", "reserva_em", "resumo", "grafico_sobra", "inexistente"},
		[]string{"resumo", "feriados", "reserva_em"},
		func(k string) string { return colunas[k] })

	if c := chaves(got); c != "resumo:topo,feriados:esquerda,reserva_em:esquerda,grafico_sobra:esquerda" {
		t.Errorf("layout = %s", c)
	}
	for _, w := range got {
		if w.Key == "grafico_sobra" && w.Visivel {
			t.Error("grafico_sobra não foi marcado e deveria ficar oculto")
		}
	}
}

func TestMontarLayoutWidgetsMantemOsQueNaoVieram(t *testing.T) {
	got := montarLayoutWidgets(layoutPadrao(), []string{"feriados"}, []string{"feriados"},
		func(string) string { return "" })
	if c := chaves(got); c != "resumo:topo,feriados:direita,grafico_sobra:esquerda,reserva_em:direita" {
		t.Errorf("layout = %s", c)
	}
	for _, w := range got {
		if w.Key == "reserva_em" && !w.Visivel {
			t.Error("widget que não veio no formulário deve manter a visibilidade atual")
		}
	}
}

func TestCanaisEnvio(t *testing.T) {
	cases := map[string][2]bool{"email": {true, false}, "telegram": {false, true}, "": {true, true}, "x": {true, true}}
	for in, want := range cases {
		if e, tg := canaisEnvio(in); e != want[0] || tg != want[1] {
			t.Errorf("canaisEnvio(%q) = %v,%v; want %v", in, e, tg, want)
		}
	}
}

func TestValorParaSalvar(t *testing.T) {
	cases := []struct {
		campo string
		vals  []string
		want  string
		ok    bool
	}{
		{"ativa", []string{"on"}, "", false},
		{"chat_id", []string{" 123 "}, "123", true},
		{"chat_id", []string{""}, "", true},    // campo comum pode ser apagado
		{"bot_token", []string{""}, "", false}, // segredo em branco mantém o salvo
		{"service_account_json", []string{"  "}, "", false},
		{"bot_token", []string{"1:abc"}, "1:abc", true}, // segredo novo substitui
		{"spreadsheet_id", nil, "", false},
	}
	for _, c := range cases {
		if v, ok := valorParaSalvar(c.campo, c.vals); v != c.want || ok != c.ok {
			t.Errorf("valorParaSalvar(%q, %q) = %q,%v; want %q,%v", c.campo, c.vals, v, ok, c.want, c.ok)
		}
	}
}
