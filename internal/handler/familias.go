package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"log"
	"net/http"
	"strings"
)

func HandleFamilias(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		familias, _ := store.GetFamilies(db)
		render(w, "familias", models.FamiliasPage{
			BasePage: bp(db, r, "familias", "Famílias"),
			Familias: familias,
		})
	})
}

func HandleCreateFamilia(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		nome := strings.TrimSpace(r.FormValue("nome"))
		if nome != "" {
			if _, err := store.CreateFamily(db, nome); err != nil {
				log.Printf("create family: %v", err)
			}
		}
		http.Redirect(w, r, "/familias", http.StatusSeeOther)
	})
}

func HandleRenameFamilia(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := parseInt(r.PathValue("id"))
		nome := strings.TrimSpace(r.FormValue("nome"))
		if nome != "" {
			store.RenameFamily(db, id, nome)
		}
		http.Redirect(w, r, "/familias", http.StatusSeeOther)
	})
}

func HandleDeleteFamilia(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if err := store.DeleteFamily(db, id); err != nil {
			log.Printf("delete family: %v", err)
		}
		http.Redirect(w, r, "/familias", http.StatusSeeOther)
	})
}
