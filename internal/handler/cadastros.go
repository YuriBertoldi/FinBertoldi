package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"net/http"
)

func HandleCadastros(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		fid := u.FamilyID
		tab := r.URL.Query().Get("tab")
		if tab == "" {
			tab = "categorias"
		}
		categorias, _ := store.GetCategorias(db, fid)
		cartoes, _ := store.GetCartoes(db, fid)

		page := models.CadastrosPage{
			BasePage:   bp(db, r, "cadastros", "Cadastros"),
			TabAtivo:   tab,
			Categorias: categorias,
			Cartoes:    cartoes,
		}

		render(w, "cadastros", page)
	}
}

func HandleCreateCategoria(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" {
			cor = "#607d8b"
		}
		store.CreateCategoria(db, fid, r.FormValue("nome"), r.FormValue("grupo"), cor)
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleUpdateCategoria(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" {
			cor = "#607d8b"
		}
		store.UpdateCategoria(db, fid, parseInt(r.PathValue("id")), r.FormValue("nome"), r.FormValue("grupo"), cor)
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleDeleteCategoria(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteCategoria(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleToggleCategoriaAtiva(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ToggleCategoriaAtiva(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=categorias", http.StatusSeeOther)
	}
}

func HandleCreateCartao(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" {
			cor = "#607d8b"
		}
		store.CreateCartao(db, fid, r.FormValue("nome"), r.FormValue("bandeira"),
			parseFloat(r.FormValue("limite")), cor)
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

func HandleUpdateCartao(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		fid := auth.CurrentUser(r).FamilyID
		cor := r.FormValue("cor")
		if cor == "" {
			cor = "#607d8b"
		}
		store.UpdateCartao(db, fid, parseInt(r.PathValue("id")), r.FormValue("nome"),
			r.FormValue("bandeira"), parseFloat(r.FormValue("limite")), cor)
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

func HandleDeleteCartao(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.DeleteCartao(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}

func HandleToggleCartaoAtivo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.ToggleCartaoAtivo(db, auth.CurrentUser(r).FamilyID, parseInt(r.PathValue("id")))
		http.Redirect(w, r, "/cadastros?tab=cartoes", http.StatusSeeOther)
	}
}
