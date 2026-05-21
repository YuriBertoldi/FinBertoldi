package models

import "time"

// --- Entidades ---

type DespesaFixa struct {
	ID        int
	Nome      string
	Valor     float64
	Categoria string
	Grupo     string
	Ativa     bool
	CriadoEm time.Time
}

type DespesaMes struct {
	ID        int
	Nome      string
	Valor     float64
	Categoria string
	Grupo     string
	Mes       string
	Pago      bool
}

type Categoria struct {
	ID       int
	Nome     string
	Grupo    string
	Cor      string
	Ativo    bool
	CriadoEm time.Time
}

type Cartao struct {
	ID       int
	Nome     string
	Bandeira string
	Limite   float64
	Cor      string
	Ativo    bool
	CriadoEm time.Time
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
	ID            int
	Pessoa        string
	Valor         float64
	ValorPago     float64
	ValorRestante float64
	Direcao       string
	Data          time.Time
	Pago          bool
	Notas         string
	CriadoEm      time.Time
}

type ScreenInfo struct {
	Key   string
	Label string
}

type User struct {
	ID             int
	Nome           string
	Email          string
	Admin          bool
	FamilyAdmin    bool
	Ativo          bool
	FamilyID       int
	FamilyNome     string
	CriadoEm       time.Time
	BlockedScreens []string
}

type Family struct {
	ID         int
	Nome       string
	NumMembros int
	CriadoEm   time.Time
}

// --- Page Data ---

type BasePage struct {
	CurrentUser    *User
	Active         string
	Title          string
	BlockedScreens []string
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
	Historico       []MesResumo
}

type MesResumo struct {
	Mes       string
	MesLabel  string
	Receitas  float64
	Despesas  float64
	Sobra     float64
	Investido float64
}

type DespesasPage struct {
	BasePage
	TabAtivo      string
	Despesas      []DespesaFixa
	Parcelamentos []Parcelamento
	Categorias    []Categoria
	Cartoes       []Cartao
}

type EmprestimosPage struct {
	BasePage
	Emprestimos   []Emprestimo
	TotalDevo     float64
	TotalAReceber float64
}

type CadastrosPage struct {
	BasePage
	TabAtivo   string
	Categorias []Categoria
	Cartoes    []Cartao
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
	Familias []Family
	Erro     string
	Sucesso  string
}

type FamiliasPage struct {
	BasePage
	Familias []Family
}

type MeuTimePage struct {
	BasePage
	Membros []User
	Telas   []ScreenInfo
	Erro    string
	Sucesso string
}

type LoginPage struct {
	Title string
	Erro  string
}

type PlanejamentoPage struct {
	BasePage
	ReceitaMensal   float64
	DespesaMensal   float64
	SobraMensal     float64
	TaxaPoupanca    float64
	PatrimonioAtual float64
	TotalInvestido  float64
	ReservaAtual    float64
	ReservaIdeal6m  float64
	ReservaIdeal12m float64
	FireConservador float64
	FireModerado    float64
	FireAgressivo   float64
	DespesaAnual    float64
	HistoricoSobra  []MesResumo
}
