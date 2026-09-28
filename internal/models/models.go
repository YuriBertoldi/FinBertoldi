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
	Pago          bool
	Mes           string
	// Financiamento
	Financiamento    bool
	ValorOriginal    float64
	TaxaJuros        float64  // taxa mensal % (ex: 1.99)
	TotalEconomizado float64
	TotalJuros       float64  // calculado: (ValorParcela * TotalParcelas) - ValorOriginal
	Antecipada       bool     // contexto parcelamentos_mes
	ValorPago        *float64 // valor efetivamente pago (nil = valor_parcela cheio)
}

type AntecipacaoPreview struct {
	ParcelaNum       int
	MesesAntecipados int
	ValorOriginal    float64
	Desconto         float64
	ValorComDesconto float64
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
	Tipo     string // "deposito" ou "retirada"
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

// DashboardResumo contém apenas os totais dos cards — usado também no OOB swap do toggle.
type DashboardResumo struct {
	TotalReceitas  float64
	TotalDespesas  float64
	Sobra          float64
	Caixa          float64
	TotalInvestido float64
	TotalPago      float64
	TotalPendente  float64
}

type DashboardData struct {
	BasePage
	DashboardResumo
	Mes             string
	MesDisplay      string
	MesPrev         string
	MesNext         string
	ReservaEM       float64
	DespesasBasicas []DespesaMes
	DespesasCartao  []DespesaMes
	DespesasVR      []DespesaMes
	Parcelamentos   []Parcelamento
	Historico       []MesResumo
	Integracoes     DashboardIntegracoes
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
	TabAtivo        string
	Despesas        []DespesaFixa
	Parcelamentos   []Parcelamento
	Categorias      []Categoria
	Cartoes         []Cartao
	MesStr           string
	MesDisplay       string
	MesPrev          string
	MesNext          string
	DespesasBasicas  []DespesaMes
	DespesasCartao   []DespesaMes
	DespesasVR       []DespesaMes
	ParcelamentosMes []Parcelamento
}

type EmprestimosPage struct {
	BasePage
	Emprestimos   []Emprestimo
	TotalDevo     float64
	TotalAReceber float64
}

type IntegracaoConfig struct {
	ID           int
	Integracao   string // "pluggy", "belvo", etc
	ClientID     string
	ClientSecret string
	ServiceURL   string
	Ativo        bool
}

type CadastrosPage struct {
	BasePage
	TabAtivo      string
	Categorias    []Categoria
	Cartoes       []Cartao
	PluggyConfig  *IntegracaoConfig
	PluggyContas  []PluggyItem
	PluggyStatus  string // "online", "offline", "nao_configurado"
	PluggyErro    string
	PluggySucesso string
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

// --- Transações Bancárias ---

type TransacaoBanco struct {
	ID           int
	Data         time.Time
	Descricao    string
	Valor        float64
	Tipo         string // "debito" ou "credito"
	Categoria    string
	Origem       string // "csv", "ofx", "pluggy"
	Banco        string
	FitID        string
	Status       string // "pendente", "categorizada", "convertida", "ignorada"
	DespesaID    *int
	ReceitaID    *int
	PluggyItemID string
	CriadoEm     time.Time
}

type PluggyItem struct {
	ID            int
	FamilyID      int
	ItemID        string
	ConnectorName string
	Status        string
	LastSync      *time.Time
	CriadoEm      time.Time
}

type TransacoesPage struct {
	BasePage
	Transacoes    []TransacaoBanco
	Categorias    []Categoria
	Contas        []PluggyItem
	TotalEntradas float64
	TotalSaidas   float64
	Filtros       TransacoesFiltros
	Resultado     *ImportBancoResultado
	Erro          string
}

type TransacoesFiltros struct {
	Mes    string
	Origem string
	Status string
	Banco  string
}

type ImportBancoResultado struct {
	Total     int
	Novos     int
	Duplicados int
}

type ImportResultado struct {
	Despesas      int
	Parcelamentos int
	Receitas      int
	Investimentos int
	Emprestimos   int
}

func (r ImportResultado) Total() int {
	return r.Despesas + r.Parcelamentos + r.Receitas + r.Investimentos + r.Emprestimos
}

// --- Integrações ---

type DadoEconomico struct {
	ID        int
	Tipo      string  // "selic", "cdi", "ipca", "usd", "eur", "btc"
	Valor     float64
	Data      time.Time
	Fonte     string // "bcb", "awesomeapi"
	CriadoEm  time.Time
}

type Feriado struct {
	Data time.Time
	Nome string
	Tipo string // "national"
}

type IntegracaoStatus struct {
	Nome       string // "bcb", "cotacoes", "telegram", "brasilapi", "sheets"
	Label      string
	Descricao  string
	Ativa      bool
	Campos     []IntegracaoCampo
	StatusMsg  string
}

type IntegracaoCampo struct {
	Nome        string
	Label       string
	Tipo        string // "text", "password", "textarea"
	Valor       string
	Placeholder string
}

type IntegracoesPage struct {
	BasePage
	Integracoes []IntegracaoStatus
	Erro        string
	Sucesso     string
}

type DashboardIntegracoes struct {
	Selic    *float64
	CDI      *float64
	IPCA     *float64
	USD      *float64
	EUR      *float64
	BTC      *float64
	Feriados []Feriado
}

type ImportExportPage struct {
	BasePage
	Resultado *ImportResultado
	Erro      string
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
