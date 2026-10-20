package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
)

func HandleReceitas(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, _ := store.GetReceitas(db, auth.CurrentUser(r).FamilyID)
		render(w, "receitas", models.ReceitasPage{BasePage: bp(db, r, "receitas", "Receitas"), Receitas: list})
	}
}

func HandleCreateReceita(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		store.CreateReceita(db, auth.CurrentUser(r).FamilyID, r.FormValue("descricao"), parseFloat(r.FormValue("valor")),
			parseDate(r.FormValue("data")), r.FormValue("tipo"), r.FormValue("recorrente") == "on")
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

func HandleDeleteReceita(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteReceita(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}

func HandleUpdateReceita(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		id := parseInt(r.PathValue("id"))
		store.UpdateReceita(db, fid, id, r.FormValue("descricao"), parseFloat(r.FormValue("valor")),
			parseDate(r.FormValue("data")), r.FormValue("tipo"), r.FormValue("recorrente") == "on")
		http.Redirect(w, r, "/receitas", http.StatusSeeOther)
	}
}
