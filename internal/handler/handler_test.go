package handler

import (
	"testing"
	"time"
)

func TestFormatBRL(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "R$ 0,00"},
		{1, "R$ 1,00"},
		{1.5, "R$ 1,50"},
		{1234.56, "R$ 1.234,56"},
		{1000000, "R$ 1.000.000,00"},
		{-50.25, "-R$ 50,25"},
		{0.01, "R$ 0,01"},
		{12.345, "R$ 12,35"},
	}
	for _, c := range cases {
		got := formatBRL(c.in)
		if got != c.want {
			t.Errorf("formatBRL(%v) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestParseFloat(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"100", 100},
		{"100,50", 100.5},
		{"1.000,50", 1000.5},
		{"1.234.567,89", 1234567.89},
		{"  500,00  ", 500},
		{"", 0},
		{"abc", 0},
		{"0", 0},
	}
	for _, c := range cases {
		got := parseFloat(c.in)
		if got != c.want {
			t.Errorf("parseFloat(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

func TestParseInt(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"42", 42},
		{"  100  ", 100},
		{"0", 0},
		{"", 0},
		{"abc", 0},
		{"-5", -5},
	}
	for _, c := range cases {
		got := parseInt(c.in)
		if got != c.want {
			t.Errorf("parseInt(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	d := parseDate("2026-05-19")
	if d.Year() != 2026 || d.Month() != 5 || d.Day() != 19 {
		t.Errorf("parseDate('2026-05-19') = %v; want 2026-05-19", d)
	}

	d2 := parseDate("invalida")
	if d2.IsZero() {
		t.Error("parseDate de string inválida não deve retornar zero time")
	}
}

func TestMesDisplay(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2026-01", "Janeiro/2026"},
		{"2026-05", "Maio/2026"},
		{"2025-12", "Dezembro/2025"},
		{"invalido", "invalido"},
	}
	for _, c := range cases {
		got := mesDisplay(c.in)
		if got != c.want {
			t.Errorf("mesDisplay(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

// Testa lógica de cálculo de parcela vigente.
func TestParcelaCalculoMesVigente(t *testing.T) {
	parcelaNoMes := func(inicio, mes time.Time) int {
		return (mes.Year()-inicio.Year())*12 + int(mes.Month()-inicio.Month()) + 1
	}

	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		mes   time.Time
		want  int
		descr string
	}{
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 1, "mês de início"},
		{time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), 2, "mês seguinte"},
		{time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), 12, "12 meses depois"},
		{time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), 13, "1 ano depois"},
		{time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), 0, "antes do início (não vigente)"},
	}
	for _, c := range cases {
		got := parcelaNoMes(inicio, c.mes)
		if got != c.want {
			t.Errorf("parcelaNoMes(%s): got %d; want %d [%s]", c.mes.Format("2006-01"), got, c.want, c.descr)
		}
	}
}

// Testa juros compostos (lógica do planejamento FIRE).
func TestJurosCompostosValorFuturo(t *testing.T) {
	fv := func(principal, pmt, annualRate float64, years int) float64 {
		r := annualRate / 100 / 12
		n := float64(years * 12)
		pow := 1.0
		for i := 0; i < int(n); i++ {
			pow *= (1 + r)
		}
		if r == 0 {
			return principal + pmt*n
		}
		return principal*pow + pmt*((pow-1)/r)
	}

	got := fv(0, 1000, 10, 10)
	if got < 200000 || got > 220000 {
		t.Errorf("FV(0, 1000/mês, 10%%, 10a) = %.2f; esperado ~205k", got)
	}

	gotZero := fv(0, 1000, 0, 10)
	if gotZero != 120000 {
		t.Errorf("FV(0, 1000/mês, 0%%, 10a) = %.2f; want 120000", gotZero)
	}

	gotJurosApenas := fv(100000, 0, 10, 10)
	if gotJurosApenas < 260000 || gotJurosApenas > 280000 {
		t.Errorf("FV(100k, 0, 10%%, 10a) = %.2f; esperado ~270k", gotJurosApenas)
	}
}

// Testa cálculo da regra FIRE.
func TestFireCalculo(t *testing.T) {
	despesaMensal := 5000.0
	despesaAnual := despesaMensal * 12

	prox := func(a, b float64) bool {
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		return diff < 0.01
	}

	if fire25 := despesaAnual * 25; !prox(fire25, 1500000) {
		t.Errorf("FIRE 25x = %.2f; want 1500000", fire25)
	}
	if fire285 := despesaAnual * 28.5; !prox(fire285, 1710000) {
		t.Errorf("FIRE 28.5x = %.2f; want 1710000", fire285)
	}
	if fire333 := despesaAnual * 33.3; !prox(fire333, 1998000) {
		t.Errorf("FIRE 33.3x = %.2f; want 1998000", fire333)
	}
}

// Testa cálculo de taxa de poupança.
func TestTaxaPoupanca(t *testing.T) {
	calcular := func(receita, despesa float64) float64 {
		if receita == 0 {
			return 0
		}
		return ((receita - despesa) / receita) * 100
	}

	cases := []struct {
		rec, des, want float64
	}{
		{10000, 8000, 20},
		{10000, 5000, 50},
		{10000, 10000, 0},
		{10000, 11000, -10},
		{0, 5000, 0},
	}
	for _, c := range cases {
		got := calcular(c.rec, c.des)
		if got != c.want {
			t.Errorf("taxa(%.0f, %.0f) = %.2f; want %.2f", c.rec, c.des, got, c.want)
		}
	}
}
