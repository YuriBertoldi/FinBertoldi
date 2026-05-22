package handler

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"time"

	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

var tmplReports = map[string]*template.Template{}

func buildReportFuncMap() template.FuncMap {
	return template.FuncMap{
		"brl": func(v float64) string { return formatBRL(v) },
	}
}

func InitReportTemplates() {
	reports := []string{"despesas", "receitas", "investimentos", "emprestimos", "dashboard"}
	for _, name := range reports {
		t := template.New("").Funcs(buildReportFuncMap())
		template.Must(t.ParseFiles(
			"templates/relatorio-base.html",
			"templates/relatorio-"+name+".html",
		))
		tmplReports[name] = t
	}
}

func renderReport(w http.ResponseWriter, name string, data any) {
	t, ok := tmplReports[name]
	if !ok {
		http.Error(w, "relatório não encontrado: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "relatorio-base", data); err != nil {
		log.Printf("report template %s error: %v", name, err)
		http.Error(w, "Erro ao gerar relatório", http.StatusInternalServerError)
	}
}

func geradoEm() string {
	return time.Now().Format("02/01/2006 15:04")
}

// --- Report data types ---

type rBase struct {
	Title      string
	FamilyNome string
	GeradoEm   string
	MesDisplay string
}

type rDespesas struct {
	rBase
	Despesas      []models.DespesaFixa
	Parcelamentos []models.Parcelamento
	TotalFixas    string
	TotalParc     string
	Total         string
}

type rReceitas struct {
	rBase
	Receitas        []models.Receita
	Total           string
	TotalRecorrente string
}

type rInvestimentos struct {
	rBase
	Investimentos    []models.Investimento
	Totais           map[string]float64
	TotalGeral       string
	ReservaEM        string
	Patrimonio       string
	HistoricoReserva []models.ReservaEM
}

type rEmprestimos struct {
	rBase
	Emprestimos   []models.Emprestimo
	TotalDevo     string
	TotalAReceber string
}

type rDashboard struct {
	rBase
	TotalReceitas  string
	TotalDespesas  string
	Sobra          string
	Sobra_f        float64
	TotalInvestido string
	ReservaEM      string
	TaxaPoupanca   string
	DespesasBasicas []models.DespesaMes
	DespesasCartao  []models.DespesaMes
	DespesasVR      []models.DespesaMes
	Parcelamentos   []models.Parcelamento
	Historico       []models.MesResumo
}

// --- Handlers ---

func HandleRelatorioDespesas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		despesas, _ := store.GetDespesasFixas(db, fid)
		parcelamentos, _ := store.GetParcelamentos(db, fid, false)

		var totalFixas, totalParc float64
		for _, d := range despesas {
			if d.Ativa {
				totalFixas += d.Valor
			}
		}
		for _, p := range parcelamentos {
			if p.Ativo {
				totalParc += p.ValorParcela
			}
		}

		renderReport(w, "despesas", rDespesas{
			rBase: rBase{
				Title:      "Relatório de Despesas",
				FamilyNome: u.FamilyNome,
				GeradoEm:   geradoEm(),
			},
			Despesas:      despesas,
			Parcelamentos: parcelamentos,
			TotalFixas:    formatBRL(totalFixas),
			TotalParc:     formatBRL(totalParc),
			Total:         formatBRL(totalFixas + totalParc),
		})
	}
}

func HandleRelatorioReceitas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		receitas, _ := store.GetReceitas(db, fid)

		var total, totalRec float64
		for _, rc := range receitas {
			total += rc.Valor
			if rc.Recorrente {
				totalRec += rc.Valor
			}
		}

		renderReport(w, "receitas", rReceitas{
			rBase: rBase{
				Title:      "Relatório de Receitas",
				FamilyNome: u.FamilyNome,
				GeradoEm:   geradoEm(),
			},
			Receitas:        receitas,
			Total:           formatBRL(total),
			TotalRecorrente: formatBRL(totalRec),
		})
	}
}

func HandleRelatorioInvestimentos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		list, _ := store.GetInvestimentos(db, fid)
		totais, totalGeral, _ := store.GetInvestimentosTotais(db, fid)
		reserva, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoReserva(db, fid)

		renderReport(w, "investimentos", rInvestimentos{
			rBase: rBase{
				Title:      "Relatório de Investimentos",
				FamilyNome: u.FamilyNome,
				GeradoEm:   geradoEm(),
			},
			Investimentos:    list,
			Totais:           totais,
			TotalGeral:       formatBRL(totalGeral),
			ReservaEM:        formatBRL(reserva),
			Patrimonio:       formatBRL(totalGeral + reserva),
			HistoricoReserva: historico,
		})
	}
}

func HandleRelatorioEmprestimos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID

		list, _ := store.GetEmprestimos(db, fid)
		var totalDevo, totalAReceber float64
		for _, e := range list {
			if e.Pago {
				continue
			}
			if e.Direcao == "devo" {
				totalDevo += e.ValorRestante
			} else {
				totalAReceber += e.ValorRestante
			}
		}

		renderReport(w, "emprestimos", rEmprestimos{
			rBase: rBase{
				Title:      "Relatório de Empréstimos",
				FamilyNome: u.FamilyNome,
				GeradoEm:   geradoEm(),
			},
			Emprestimos:   list,
			TotalDevo:     formatBRL(totalDevo),
			TotalAReceber: formatBRL(totalAReceber),
		})
	}
}

func HandleRelatorioDashboard(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		mes, mesStr := mesFromRequest(r)

		if err := store.EnsureDespesasMes(db, fid, mes); err != nil {
			log.Printf("ensure despesas mes: %v", err)
		}

		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcelamentos, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		reserva, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoMeses(db, fid, mes, 6)

		var totalDespesas float64
		for _, d := range basicas { totalDespesas += d.Valor }
		for _, d := range cartao  { totalDespesas += d.Valor }
		for _, d := range vr      { totalDespesas += d.Valor }
		for _, p := range parcelamentos { totalDespesas += p.ValorParcela }

		sobra := totalReceitas - totalDespesas
		taxaPoup := 0.0
		if totalReceitas > 0 {
			taxaPoup = (sobra / totalReceitas) * 100
		}

		renderReport(w, "dashboard", rDashboard{
			rBase: rBase{
				Title:      "Relatório Geral",
				FamilyNome: u.FamilyNome,
				GeradoEm:   geradoEm(),
				MesDisplay: mesDisplay(mesStr),
			},
			TotalReceitas:   formatBRL(totalReceitas),
			TotalDespesas:   formatBRL(totalDespesas),
			Sobra:           formatBRL(sobra),
			Sobra_f:         sobra,
			TotalInvestido:  formatBRL(totalInvestido),
			ReservaEM:       formatBRL(reserva),
			TaxaPoupanca:    formatBRL(taxaPoup) + "%",
			DespesasBasicas: basicas,
			DespesasCartao:  cartao,
			DespesasVR:      vr,
			Parcelamentos:   parcelamentos,
			Historico:       historico,
		})
	}
}
