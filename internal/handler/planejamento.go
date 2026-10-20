package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
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
		})
	}
}
