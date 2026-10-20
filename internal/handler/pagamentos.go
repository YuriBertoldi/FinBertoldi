package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
)

func HandlePagamentos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		mes, mesStr := mesFromRequest(r)
		store.EnsureDespesasMes(db, fid, mes)
		store.AutoFinalizarParcelamentos(db, fid)
		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcelamentosMes, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)
		render(w, "pagamentos", models.DespesasPage{
			BasePage:         bp(db, r, "pagamentos", "Pagamentos do Mês"),
			MesStr:           mesStr,
			MesDisplay:       mesDisplay(mesStr),
			MesPrev:          mes.AddDate(0, -1, 0).Format("2006-01"),
			MesNext:          mes.AddDate(0, 1, 0).Format("2006-01"),
			DespesasBasicas:  basicas,
			DespesasCartao:   cartao,
			DespesasVR:       vr,
			ParcelamentosMes: parcelamentosMes,
		})
	}
}
