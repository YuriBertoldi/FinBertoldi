package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
)

func HandleDespesas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		tab := r.URL.Query().Get("tab")
		if tab == "" {
			tab = "fixas"
		}
		mes, mesStr := mesFromRequest(r)
		despesas, _ := store.GetDespesasFixas(db, fid)
		parcelamentos, _ := store.GetParcelamentos(db, fid, false)
		categorias, _ := store.GetCategorias(db, fid)
		cartoes, _ := store.GetCartoes(db, fid)
		store.EnsureDespesasMes(db, fid, mes)
		basicas, cartao, vr, _ := store.GetDespesasMes(db, fid, mes, mesStr)
		parcelamentosMes, _ := store.GetParcelamentosVigentesNoMes(db, fid, mes)

		render(w, "despesas", models.DespesasPage{
			BasePage:         bp(db, r, "despesas", "Despesas"),
			TabAtivo:         tab,
			Despesas:         despesas,
			Parcelamentos:    parcelamentos,
			Categorias:       categorias,
			Cartoes:          cartoes,
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

func HandleCreateDespesa(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateDespesa(db, auth.CurrentUser(r).FamilyID, r.FormValue("nome"), parseFloat(r.FormValue("valor")), r.FormValue("categoria"))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func HandleUpdateDespesa(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.UpdateDespesa(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")),
			r.FormValue("nome"), parseFloat(r.FormValue("valor")), r.FormValue("categoria"))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func HandleDeleteDespesa(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteDespesa(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}

func HandleToggleDespesaAtiva(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ToggleDespesaAtiva(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/despesas", http.StatusSeeOther)
	}
}
