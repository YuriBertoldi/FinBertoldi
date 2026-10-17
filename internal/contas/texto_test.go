package contas

import (
	"strings"
	"testing"
	"unicode/utf8"

	"fincontrol/internal/models"
)

func TestBuildTexto(t *testing.T) {
	despesas := []models.DespesaMes{
		{Nome: "Energia", Valor: 500, Pago: true},
		{Nome: "Plano saude Eloah/Rhaizza", Valor: 1000},
	}
	parcs := []models.Parcelamento{
		{Descricao: "Carro", ValorParcela: 2514, ParcelaAtual: 3, TotalParcelas: 36},
		{Descricao: "Dufry 10/10", ValorParcela: 74.47, ParcelaAtual: 1, TotalParcelas: 1},
	}
	txt := BuildTexto("Outubro/2026", despesas, parcs, 5000)

	for _, want := range []string{
		"*💰 Contas – Outubro/2026*",
		"*🏠 FIXAS*", "*📦 PARCELAMENTOS*", "*💳 COMPRAS À VISTA*",
		"Carro 3/36", "Dufry 10/10",
		waLinha("TOTAL", 4088.47), waLinha("Já pago", 500), waLinha("A pagar", 3588.47),
		waLinha("Sobra", 911.53),
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("faltando %q em:\n%s", want, txt)
		}
	}
	// maior valor primeiro dentro da seção
	if strings.Index(txt, "Plano saude") > strings.Index(txt, "Energia") {
		t.Error("fixas deveriam estar ordenadas por valor decrescente")
	}
}

func TestWaLinhaLargura(t *testing.T) {
	for _, l := range []string{
		waLinha("Plano saude Eloah/Rhaizza", 1000),
		waLinha("Água", 5.4),
		waLinha("TOTAL", 12946.95),
		waLinha("Sobra", -49.95),
	} {
		if n := utf8.RuneCountInString(strings.TrimSuffix(l, "\n")); n != waLargura {
			t.Errorf("linha %q com %d colunas, esperado %d", l, n, waLargura)
		}
	}
}

func TestWaValor(t *testing.T) {
	cases := map[float64]string{0: "0,00", 5.4: "5,40", 1234.56: "1.234,56", 12946.95: "12.946,95", -49.95: "-49,95"}
	for in, want := range cases {
		if got := waValor(in); got != want {
			t.Errorf("waValor(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTelegramHTML(t *testing.T) {
	got := TelegramHTML("*💰 Contas*\n\n*🏠 FIXAS*\n```\nA & B   1,00\n```\n")
	want := "<b>💰 Contas</b>\n\n<b>🏠 FIXAS</b><pre>A &amp; B   1,00</pre>"
	if got != want {
		t.Errorf("TelegramHTML =\n%q\nwant\n%q", got, want)
	}
}

func TestEmailHTMLEscapa(t *testing.T) {
	h := EmailHTML("*x*\n```\n<script>\n```")
	if strings.Contains(h, "<script>") || !strings.Contains(h, "<strong>x</strong>") {
		t.Errorf("EmailHTML não escapou/converteu corretamente:\n%s", h)
	}
}
