package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func HandlePlanejamento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		mes := time.Now()
		mes = time.Date(mes.Year(), mes.Month(), 1, 0, 0, 0, 0, time.UTC)

		store.AutoFinalizarParcelamentos(db, fid)

		var receitaMensal float64
		db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM receitas WHERE recorrente = true AND family_id = $1`, fid).Scan(&receitaMensal)

		var despesasFixas float64
		db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM despesas_fixas WHERE ativa = true AND family_id = $1`, fid).Scan(&despesasFixas)

		var despesasParc float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor_parcela),0) FROM parcelamentos
			WHERE ativo = true AND family_id = $2
			  AND data_inicio <= $1::date
			  AND (EXTRACT(YEAR FROM $1::date) - EXTRACT(YEAR FROM data_inicio)) * 12
			    + (EXTRACT(MONTH FROM $1::date) - EXTRACT(MONTH FROM data_inicio)) + 1 <= total_parcelas`, mes, fid).Scan(&despesasParc)

		despesaMensal := despesasFixas + despesasParc
		sobra := receitaMensal - despesaMensal
		taxaPoup := 0.0
		if receitaMensal > 0 {
			taxaPoup = (sobra / receitaMensal) * 100
		}

		_, totalInvest, _ := store.GetInvestimentosTotais(db, fid)
		reserva, _ := store.GetReservaEM(db, fid)
		patrimonio := totalInvest + reserva
		despesaAnual := despesaMensal * 12
		historico, _ := store.GetHistoricoMeses(db, fid, mes, 6)
		metas, err := store.Metas(db, fid)
		if err != nil {
			log.Printf("metas da familia %d: %v", fid, err)
		}

		render(w, "planejamento", models.PlanejamentoPage{
			BasePage:        bp(db, r, "planejamento", "Planejamento"),
			ReceitaMensal:   receitaMensal,
			DespesaMensal:   despesaMensal,
			SobraMensal:     sobra,
			TaxaPoupanca:    taxaPoup,
			PatrimonioAtual: patrimonio,
			TotalInvestido:  totalInvest,
			ReservaAtual:    reserva,
			ReservaIdeal6m:  despesaMensal * 6,
			ReservaIdeal12m: despesaMensal * 12,
			DespesaAnual:    despesaAnual,
			FireAgressivo:   despesaAnual * 25,
			FireModerado:    despesaAnual * 28.5,
			FireConservador: despesaAnual * 33.3,
			HistoricoSobra:  historico,
			Metas:           metas,
			Hoje:            time.Now(),
			Mensagem:        r.URL.Query().Get("msg"),
		})
	}
}

func HandleCreateMeta(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		prazo, err := time.Parse("2006-01", r.FormValue("prazo"))
		valorAlvo := parseFloat(r.FormValue("valor_alvo"))
		if err != nil || valorAlvo <= 0 || strings.TrimSpace(r.FormValue("nome")) == "" {
			redirecionarParaMetas(w, r, "Informe nome, valor-alvo maior que zero e prazo (mês/ano).")
			return
		}
		if err := store.CriarMeta(db, fid, r.FormValue("nome"), valorAlvo, prazo); err != nil {
			log.Printf("criar meta: %v", err)
			redirecionarParaMetas(w, r, "Não foi possível criar a meta.")
			return
		}
		redirecionarParaMetas(w, r, "Meta criada.")
	}
}

// HandleAporteNaMeta registra um aporte; com "retirada" marcada, o valor entra negativo.
func HandleAporteNaMeta(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		valor := parseFloat(r.FormValue("valor"))
		if valor <= 0 {
			redirecionarParaMetas(w, r, "Informe um valor maior que zero.")
			return
		}
		if r.FormValue("retirada") == "on" {
			valor = -valor
		}
		fid := auth.CurrentUser(r).FamilyID
		if err := store.RegistrarAporteNaMeta(db, fid, parseInt(r.PathValue("id")), valor, time.Now()); err != nil {
			log.Printf("aporte na meta: %v", err)
			redirecionarParaMetas(w, r, "Não foi possível registrar.")
			return
		}
		redirecionarParaMetas(w, r, "Registrado.")
	}
}

func HandleDeleteMeta(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ExcluirMeta(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		redirecionarParaMetas(w, r, "Meta excluída.")
	}
}

func redirecionarParaMetas(w http.ResponseWriter, r *http.Request, mensagem string) {
	http.Redirect(w, r, "/planejamento?msg="+url.QueryEscape(mensagem)+"#metas", http.StatusSeeOther)
}
