package main

import "time"

// --- Entidades ---

type DespesaFixa struct {
	ID        int
	Nome      string
	Valor     float64
	Categoria string
	Ativa     bool
	CriadoEm time.Time
}

type DespesaMes struct {
	ID        int
	Nome      string
	Valor     float64
	Categoria string
	Mes       string
	Pago      bool
}

type Parcelamento struct {
	ID            int
	Descricao     string
	Cartao        string
	ValorParcela  float64
	ParcelaAtual  int
	TotalParcelas int
	DataInicio    time.Time
	Ativo         bool
	Restantes     int
	ValorRestante float64
}

type Receita struct {
	ID         int
	Descricao  string
	Valor      float64
	Data       time.Time
	Tipo       string
	Recorrente bool
	CriadoEm  time.Time
}

type Investimento struct {
	ID          int
	Instituicao string
	Tipo        string
	Valor       float64
	Data        time.Time
	Notas       string
	CriadoEm   time.Time
}

type ReservaEM struct {
	ID       int
	Valor    float64
	Data     time.Time
	Notas    string
	CriadoEm time.Time
}

type Emprestimo struct {
	ID       int
	Pessoa   string
	Valor    float64
	Direcao  string
	Data     time.Time
	Pago     bool
	Notas    string
	CriadoEm time.Time
}

type User struct {
	ID        int
	Nome      string
	Email     string
	Admin     bool
	Ativo     bool
	CriadoEm time.Time
}

// --- Page Data ---

// BasePage é embutida em todos os page data. Contém campos comuns ao layout.
type BasePage struct {
	CurrentUser *User
	Active      string
	Title       string
}

type DashboardData struct {
	BasePage
	Mes             string
	MesDisplay      string
	MesPrev         string
	MesNext         string
	TotalReceitas   float64
	TotalDespesas   float64
	Sobra           float64
	TotalInvestido  float64
	ReservaEM       float64
	DespesasBasicas []DespesaMes
	DespesasCartao  []DespesaMes
	DespesasVR      []DespesaMes
	Parcelamentos   []Parcelamento
	// Histórico para gráficos (últimos N meses)
	Historico       []MesResumo
}

// MesResumo representa o total de receitas/despesas de um mês para gráficos
type MesResumo struct {
	Mes          string  // "2026-05"
	MesLabel     string  // "Mai/26"
	Receitas     float64
	Despesas     float64
	Sobra        float64
	Investido    float64
}

type DespesasPage struct {
	BasePage
	TabAtivo      string
	Despesas      []DespesaFixa
	Parcelamentos []Parcelamento
	Emprestimos   []Emprestimo
	TotalDevo     float64
	TotalAReceber float64
}

type ReceitasPage struct {
	BasePage
	Receitas []Receita
}

type InvestimentosPage struct {
	BasePage
	Investimentos    []Investimento
	Totais           map[string]float64
	TotalGeral       float64
	ReservaEM        float64
	HistoricoReserva []ReservaEM
}

type UsuariosPage struct {
	BasePage
	Usuarios []User
	Erro     string
	Sucesso  string
}

type LoginPage struct {
	Title string
	Erro  string
}

type PlanejamentoPage struct {
	BasePage
	ReceitaMensal     float64 // receitas recorrentes (mensais)
	DespesaMensal     float64 // despesas fixas ativas + parcelamentos vigentes
	SobraMensal       float64 // receita - despesa
	TaxaPoupanca      float64 // sobra/receita * 100
	PatrimonioAtual   float64 // total investido + reserva EM
	TotalInvestido    float64
	ReservaAtual      float64
	ReservaIdeal6m    float64 // 6x despesa mensal
	ReservaIdeal12m   float64
	FireConservador   float64 // 33.3x despesa anual (3% withdrawal)
	FireModerado      float64 // 28.5x (3.5%)
	FireAgressivo     float64 // 25x (4%)
	DespesaAnual      float64
	HistoricoSobra    []MesResumo
}
