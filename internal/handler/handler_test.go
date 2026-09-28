package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fincontrol/internal/models"
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
		{"223.05", 223.05},   // float padrão do excelize (dot decimal)
		{"1234.56", 1234.56}, // float padrão sem milhar
		{"807", 807},
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

func TestMesFromRequest(t *testing.T) {
	// mês válido via query param
	r := httptest.NewRequest("GET", "/?mes=2026-03", nil)
	mes, mesStr := mesFromRequest(r)
	if mesStr != "2026-03" {
		t.Errorf("mesStr = %q; want '2026-03'", mesStr)
	}
	if mes.Year() != 2026 || mes.Month() != 3 || mes.Day() != 1 {
		t.Errorf("mes = %v; want 2026-03-01", mes)
	}

	// sem param → mês atual
	r2 := httptest.NewRequest("GET", "/", nil)
	_, mesStr2 := mesFromRequest(r2)
	want := time.Now().Format("2006-01")
	if mesStr2 != want {
		t.Errorf("sem param: mesStr = %q; want %q", mesStr2, want)
	}

	// param inválido → fallback para mês atual
	r3 := httptest.NewRequest("GET", "/?mes=invalido", nil)
	_, mesStr3 := mesFromRequest(r3)
	if mesStr3 != want {
		t.Errorf("param inválido: mesStr = %q; want %q", mesStr3, want)
	}
}

func TestBoolStr(t *testing.T) {
	if got := boolStr(true); got != "S" {
		t.Errorf("boolStr(true) = %q; want 'S'", got)
	}
	if got := boolStr(false); got != "N" {
		t.Errorf("boolStr(false) = %q; want 'N'", got)
	}
}

func TestParseBool(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"S", true},
		{"s", true},
		{"SIM", true},
		{"sim", true},
		{"TRUE", true},
		{"true", true},
		{"1", true},
		{"N", false},
		{"NAO", false},
		{"FALSE", false},
		{"0", false},
		{"", false},
	}
	for _, c := range cases {
		got := parseBool(c.in)
		if got != c.want {
			t.Errorf("parseBool(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

func TestStrRow(t *testing.T) {
	row := []string{"  alpha  ", "beta", ""}
	if got := str(row, 0); got != "alpha" {
		t.Errorf("str(row,0) = %q; want 'alpha' (sem espaços)", got)
	}
	if got := str(row, 1); got != "beta" {
		t.Errorf("str(row,1) = %q; want 'beta'", got)
	}
	if got := str(row, 2); got != "" {
		t.Errorf("str(row,2) = %q; want ''", got)
	}
	// índice fora do range
	if got := str(row, 10); got != "" {
		t.Errorf("str(row,10) = %q; want ''", got)
	}
}

// Testa cálculo dos cards do dashboard (calcResumo).
func TestCalcResumo(t *testing.T) {
	basicas := []models.DespesaMes{
		{Valor: 500, Pago: true},
		{Valor: 300, Pago: false},
	}
	cartao := []models.DespesaMes{
		{Valor: 200, Pago: true},
	}
	vr := []models.DespesaMes{}
	parcs := []models.Parcelamento{
		{ValorParcela: 150, Pago: false},
	}

	resumo := calcResumo(basicas, cartao, vr, parcs, 5000, 20000, 1000)

	// TotalDespesas = 500 + 300 + 200 + 150 = 1150
	if resumo.TotalDespesas != 1150 {
		t.Errorf("TotalDespesas = %.2f; want 1150", resumo.TotalDespesas)
	}
	// TotalPago = 500 + 200 = 700
	if resumo.TotalPago != 700 {
		t.Errorf("TotalPago = %.2f; want 700", resumo.TotalPago)
	}
	// TotalPendente = 300 + 150 = 450
	if resumo.TotalPendente != 450 {
		t.Errorf("TotalPendente = %.2f; want 450", resumo.TotalPendente)
	}
	// Sobra = 5000 - 1150 = 3850
	if resumo.Sobra != 3850 {
		t.Errorf("Sobra = %.2f; want 3850", resumo.Sobra)
	}
	// Caixa = Receitas - Pago - InvestidoNoMes = 5000 - 700 - 1000 = 3300
	if resumo.Caixa != 3300 {
		t.Errorf("Caixa = %.2f; want 3300 (5000 - 700 - 1000)", resumo.Caixa)
	}
	if resumo.TotalInvestido != 20000 {
		t.Errorf("TotalInvestido = %.2f; want 20000", resumo.TotalInvestido)
	}
}

// --- Testes dos parsers de importação bancária (bankimport.go) ---

func TestParseDateFlex(t *testing.T) {
	cases := []struct {
		in      string
		wantY   int
		wantM   time.Month
		wantD   int
		wantErr bool
	}{
		{"15/03/2026", 2026, 3, 15, false},
		{"2026-03-15", 2026, 3, 15, false},
		{"15-03-2026", 2026, 3, 15, false},
		{"1/5/2026", 2026, 5, 1, false},
		{"invalido", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}
	for _, c := range cases {
		got, err := parseDateFlex(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseDateFlex(%q) deveria dar erro", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDateFlex(%q) erro inesperado: %v", c.in, err)
			continue
		}
		if got.Year() != c.wantY || got.Month() != c.wantM || got.Day() != c.wantD {
			t.Errorf("parseDateFlex(%q) = %v; want %d-%02d-%02d", c.in, got, c.wantY, c.wantM, c.wantD)
		}
	}
}

func TestParseValorBR(t *testing.T) {
	cases := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{"100", 100, false},
		{"100,50", 100.5, false},
		{"1.234,56", 1234.56, false},
		{"1.234.567,89", 1234567.89, false},
		{"1,234.56", 1234.56, false},
		{"-50,25", -50.25, false},
		{"R$ 1.234,56", 1234.56, false},
		{"R$100", 100, false},
		{"  500,00  ", 500, false},
		{"", 0, true},
		{"abc", 0, true},
	}
	for _, c := range cases {
		got, err := parseValorBR(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseValorBR(%q) deveria dar erro", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseValorBR(%q) erro inesperado: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseValorBR(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

func TestNormalizeHeader(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"  Data  ", "data"},
		{"DESCRICAO", "descricao"},
		{"\xef\xbb\xbfdata", "data"},
		{"Valor", "valor"},
	}
	for _, c := range cases {
		got := normalizeHeader(c.in)
		if got != c.want {
			t.Errorf("normalizeHeader(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestParseCSVFile(t *testing.T) {
	csv := "data,descricao,valor\n15/03/2026,Supermercado,-150.50\n16/03/2026,Salario,5000\n"
	txns, err := parseCSVFile(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseCSVFile erro: %v", err)
	}
	if len(txns) != 2 {
		t.Fatalf("esperado 2 transacoes, got %d", len(txns))
	}

	if txns[0].Descricao != "Supermercado" {
		t.Errorf("txn[0].Descricao = %q; want 'Supermercado'", txns[0].Descricao)
	}
	if txns[0].Valor != -150.50 {
		t.Errorf("txn[0].Valor = %v; want -150.50", txns[0].Valor)
	}
	if txns[0].Tipo != "debito" {
		t.Errorf("txn[0].Tipo = %q; want 'debito'", txns[0].Tipo)
	}
	if txns[0].Origem != "csv" {
		t.Errorf("txn[0].Origem = %q; want 'csv'", txns[0].Origem)
	}

	if txns[1].Tipo != "credito" {
		t.Errorf("txn[1].Tipo = %q; want 'credito' (valor positivo)", txns[1].Tipo)
	}
}

func TestParseCSVFile_SemicolonSeparated(t *testing.T) {
	csv := "data;descricao;valor\n15/03/2026;Padaria;-25,50\n"
	txns, err := parseCSVFile(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("parseCSVFile semicolon erro: %v", err)
	}
	if len(txns) != 1 {
		t.Fatalf("esperado 1 transacao, got %d", len(txns))
	}
	if txns[0].Descricao != "Padaria" {
		t.Errorf("Descricao = %q; want 'Padaria'", txns[0].Descricao)
	}
}

func TestParseCSVFile_MissingColumns(t *testing.T) {
	csv := "nome,preco\nProduto,100\n"
	_, err := parseCSVFile(strings.NewReader(csv))
	if err == nil {
		t.Error("parseCSVFile sem colunas obrigatorias deveria dar erro")
	}
}

func TestParseCSVFile_WithCategoria(t *testing.T) {
	csv := "data,descricao,valor,categoria\n2026-01-15,Netflix,-55.90,Streaming\n"
	txns, err := parseCSVFile(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if txns[0].Categoria != "Streaming" {
		t.Errorf("Categoria = %q; want 'Streaming'", txns[0].Categoria)
	}
}

func TestParseCSVFile_Empty(t *testing.T) {
	csv := "data,descricao,valor\n"
	_, err := parseCSVFile(strings.NewReader(csv))
	if err == nil {
		t.Error("parseCSVFile vazio deveria dar erro")
	}
}

func TestParseCSVFile_FitIDDedup(t *testing.T) {
	csv := "data,descricao,valor\n2026-01-15,Compra A,-100\n2026-01-15,Compra A,-100\n"
	txns, err := parseCSVFile(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if txns[0].FitID == "" {
		t.Error("FitID deveria ser gerado para deduplicacao")
	}
	if txns[0].FitID != txns[1].FitID {
		t.Error("transacoes identicas deveriam gerar o mesmo FitID")
	}
}
