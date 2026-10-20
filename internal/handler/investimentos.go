package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
)

func HandleInvestimentos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid := auth.CurrentUser(r).FamilyID
		list, _ := store.GetInvestimentos(db, fid)
		totais, totalGeral, _ := store.GetInvestimentosTotais(db, fid)
		reserva, _ := store.GetReservaEM(db, fid)
		historico, _ := store.GetHistoricoReserva(db, fid)
		render(w, "investimentos", models.InvestimentosPage{
			BasePage:         bp(db, r, "investimentos", "Investimentos"),
			Investimentos:    list,
			Totais:           totais,
			TotalGeral:       totalGeral + reserva, // reserva conta como investimento
			ReservaEM:        reserva,
			HistoricoReserva: historico,
		})
	}
}

func HandleCreateInvestimento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateInvestimento(db, auth.CurrentUser(r).FamilyID, r.FormValue("instituicao"), r.FormValue("tipo"),
			parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleDeleteInvestimento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteInvestimento(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleAddReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		tipo := r.FormValue("tipo")
		if tipo == "" {
			tipo = "deposito"
		}
		store.AddReservaEM(db, fid, parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), tipo, r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleUpdateReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		tipo := r.FormValue("tipo")
		if tipo == "" {
			tipo = "deposito"
		}
		store.UpdateReservaEM(db, fid, id, parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), tipo, r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleDeleteReservaEM(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteReservaEM(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}

func HandleUpdateInvestimento(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		store.UpdateInvestimento(db, fid, id, r.FormValue("instituicao"), r.FormValue("tipo"),
			parseFloat(r.FormValue("valor")), parseDate(r.FormValue("data")), r.FormValue("notas"))
		http.Redirect(w, r, "/investimentos", http.StatusSeeOther)
	}
}
