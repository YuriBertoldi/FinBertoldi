package handler

import (
	"bytes"
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"log"
	"net/http"
)

func HandleDashboard(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		mes, mesStr := mesFromRequest(r)
		prevMes := mes.AddDate(0, -1, 0).Format("2006-01")
		nextMes := mes.AddDate(0, 1, 0).Format("2006-01")

		if err := store.EnsureDespesasMes(db, fid, mes); err != nil {
			log.Printf("ensure despesas mes: %v", err)
		}
		if err := store.AutoFinalizarParcelamentos(db, fid); err != nil {
			log.Printf("auto finalizar parcelamentos: %v", err)
		}

		basicas, cartao, vr, err := store.GetDespesasMes(db, fid, mes, mesStr)
		if err != nil {
			log.Printf("get despesas mes: %v", err)
		}
		parcelamentos, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		investidoNoMes := store.GetInvestidoNoMes(db, fid, mes)
		reservaEM, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoMeses(db, fid, mes, 6)

		// Reserva de emergência conta como investimento no total
		resumo := calcResumo(basicas, cartao, vr, parcelamentos, totalReceitas, totalInvestido+reservaEM, investidoNoMes)

		integ := store.GetDadosEconomicos(db)
		// Carregar feriados próximos
		feriados, _ := store.GetFeriadosProximos(db, 5)
		integ.Feriados = feriados

		u := auth.CurrentUser(r)
		layout := store.GetDashboardLayout(db, u.ID)
		widgets := map[string]bool{}
		var esquerda, direita []string
		for _, wd := range layout {
			widgets[wd.Key] = wd.Visivel
			if !wd.Visivel {
				continue
			}
			switch wd.Coluna {
			case "esquerda":
				esquerda = append(esquerda, wd.Key)
			case "direita":
				direita = append(direita, wd.Key)
			}
		}

		render(w, "dashboard", models.DashboardData{
			BasePage:        bp(db, r, "dashboard", "Dashboard"),
			DashboardResumo: resumo,
			Mes:             mesStr,
			MesDisplay:      mesDisplay(mesStr),
			MesPrev:         prevMes,
			MesNext:         nextMes,
			ReservaEM:       reservaEM,
			DespesasBasicas: basicas,
			DespesasCartao:  cartao,
			DespesasVR:      vr,
			Parcelamentos:   parcelamentos,
			Historico:       historico,
			Integracoes:     integ,
			Widgets:         widgets,
			WidgetLayout:    layout,
			WidgetsEsquerda: esquerda,
			WidgetsDireita:  direita,
		})
	}
}

func HandleSaveWidgets(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		r.ParseForm()

		layout := montarLayoutWidgets(store.GetDashboardLayout(db, u.ID), r.Form["ordem"], r.Form["widgets"], r.FormValue)
		if err := store.SaveDashboardLayout(db, u.ID, layout); err != nil {
			log.Printf("salvar widgets: %v", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// montarLayoutWidgets aplica o que veio do diálogo (ordem, marcados e coluna de cada um)
// sobre o layout atual. Chaves desconhecidas são ignoradas; widgets que não vieram no
// formulário mantêm a configuração e vão para o fim; o "resumo" continua no topo.
func montarLayoutWidgets(atual []models.DashboardWidget, ordem, marcados []string, coluna func(string) string) []models.DashboardWidget {
	visiveis := map[string]bool{}
	for _, k := range marcados {
		visiveis[k] = true
	}
	pendentes := map[string]models.DashboardWidget{}
	for _, wd := range atual {
		pendentes[wd.Key] = wd
	}
	var layout []models.DashboardWidget
	for _, k := range ordem {
		wd, ok := pendentes[k]
		if !ok {
			continue
		}
		delete(pendentes, k)
		wd.Visivel = visiveis[k]
		if c := coluna("coluna_" + k); wd.Coluna != "topo" && (c == "esquerda" || c == "direita") {
			wd.Coluna = c
		}
		layout = append(layout, wd)
	}
	for _, wd := range atual {
		if _, falta := pendentes[wd.Key]; falta {
			layout = append(layout, wd)
		}
	}
	return store.OrdenarTopo(layout)
}

// calcResumo agrega os totais do mês para os cards de resumo.
func calcResumo(basicas, cartao, vr []models.DespesaMes, parcs []models.Parcelamento, totalReceitas, totalInvestido, investidoNoMes float64) models.DashboardResumo {
	var totalDespesas, totalPago, totalPendente float64
	for _, d := range append(append(basicas, cartao...), vr...) {
		totalDespesas += d.Valor
		if d.Pago {
			totalPago += d.Valor
		} else {
			totalPendente += d.Valor
		}
	}
	for _, p := range parcs {
		totalDespesas += p.ValorParcela
		if p.Pago {
			totalPago += p.ValorParcela
		} else {
			totalPendente += p.ValorParcela
		}
	}
	return models.DashboardResumo{
		TotalReceitas:  totalReceitas,
		TotalDespesas:  totalDespesas,
		Sobra:          totalReceitas - totalDespesas,
		Caixa:          totalReceitas - totalPago - investidoNoMes,
		TotalInvestido: totalInvestido,
		TotalPago:      totalPago,
		TotalPendente:  totalPendente,
	}
}

func HandleToggleParcelamentoPago(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		mes, mesStr := mesFromRequest(r)

		pago, err := store.ToggleParcelamentoMesPago(db, fid, id, mes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var p models.Parcelamento
		db.QueryRow(`SELECT id, descricao, cartao, valor_parcela FROM parcelamentos WHERE id=$1 AND family_id=$2`, id, fid).
			Scan(&p.ID, &p.Descricao, &p.Cartao, &p.ValorParcela)
		p.Mes = mesStr
		p.Pago = pago

		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcs, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		investidoNoMes := store.GetInvestidoNoMes(db, fid, mes)
		reservaEMp, _ := store.GetReservaEM(db, fid)
		resumo := calcResumo(basicas, cartao, vr, parcs, totalReceitas, totalInvestido+reservaEMp, investidoNoMes)

		t := tmplPages["dashboard"]
		var buf bytes.Buffer
		t.ExecuteTemplate(&buf, "toggle-parc-btn", p)
		t.ExecuteTemplate(&buf, "dash-resumo-oob", resumo)
		w.Write(buf.Bytes())
	}
}

func HandleToggleDespesaPago(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		mes, mesStr := mesFromRequest(r)

		pago, err := store.ToggleDespesaMesPago(db, fid, id, mes)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var d models.DespesaMes
		db.QueryRow(`SELECT id, nome, valor, categoria FROM despesas_fixas WHERE id=$1 AND family_id=$2`, id, fid).
			Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria)
		d.Mes = mesStr
		d.Pago = pago

		// Recalcula totais para o OOB swap dos cards
		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcs, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		totalReceitas, _ := store.GetReceitasMes(db, fid, mes)
		_, totalInvestido, _ := store.GetInvestimentosTotais(db, fid)
		investidoNoMes2 := store.GetInvestidoNoMes(db, fid, mes)
		reservaEMd, _ := store.GetReservaEM(db, fid)
		resumo := calcResumo(basicas, cartao, vr, parcs, totalReceitas, totalInvestido+reservaEMd, investidoNoMes2)

		t := tmplPages["dashboard"]
		var buf bytes.Buffer
		t.ExecuteTemplate(&buf, "toggle-btn", d)
		t.ExecuteTemplate(&buf, "dash-resumo-oob", resumo)
		w.Write(buf.Bytes())
	}
}
