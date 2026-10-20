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
)

func HandleUsuarios(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		users, _ := store.GetUsers(db)
		familias, _ := store.GetFamilies(db)
		render(w, "usuarios", models.UsuariosPage{
			BasePage: bp(db, r, "usuarios", "Usuários"),
			Usuarios: users,
			Familias: familias,
			Erro:     r.URL.Query().Get("erro"),
		})
	})
}

func HandleCreateUsuario(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		nome := strings.TrimSpace(r.FormValue("nome"))
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")
		admin := r.FormValue("admin") == "on"
		familyOpt := r.FormValue("family_opt")
		familyIDStr := r.FormValue("family_id")
		newFamilyName := strings.TrimSpace(r.FormValue("new_family_name"))

		if nome == "" || email == "" || senha == "" {
			http.Redirect(w, r, "/usuarios?erro=campos", http.StatusSeeOther)
			return
		}
		hash, err := store.HashSenha(senha)
		if err != nil {
			http.Redirect(w, r, "/usuarios?erro=hash", http.StatusSeeOther)
			return
		}

		var familyID int
		if familyOpt == "nova" {
			if newFamilyName == "" {
				newFamilyName = nome
			}
			familyID, err = store.CreateFamily(db, newFamilyName)
			if err != nil {
				log.Printf("create family: %v", err)
				http.Redirect(w, r, "/usuarios?erro=familia", http.StatusSeeOther)
				return
			}
		} else {
			familyID = parseInt(familyIDStr)
			if familyID == 0 {
				db.QueryRow(`SELECT id FROM families WHERE nome='Família Padrão'`).Scan(&familyID)
			}
		}

		if err := store.CreateUser(db, nome, email, hash, admin, familyID); err != nil {
			log.Printf("create user: %v", err)
			http.Redirect(w, r, "/usuarios?erro=email", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleChangeUserFamily(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		uid := parseInt(r.PathValue("id"))
		fid := parseInt(r.FormValue("family_id"))
		if fid > 0 {
			store.SetUserFamily(db, uid, fid)
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleToggleUsuarioAdmin(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == auth.CurrentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		store.ToggleUserAdmin(db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleToggleUsuarioAtivo(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == auth.CurrentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		store.ToggleUserAtivo(db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleResetSenha(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		id := parseInt(r.PathValue("id"))
		senha := r.FormValue("nova_senha")
		if msg := validarNovaSenha(senha, ""); msg != "" {
			http.Redirect(w, r, "/usuarios?erro="+url.QueryEscape(msg), http.StatusSeeOther)
			return
		}
		if hash, err := store.HashSenha(senha); err == nil {
			store.ResetSenha(db, id, hash)
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleDeleteUsuario(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id == auth.CurrentUser(r).ID {
			http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
			return
		}
		store.DeleteUser(db, id)
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}

func HandleToggleFamilyAdminFlag(db *sql.DB) http.HandlerFunc {
	return auth.AdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		id := parseInt(r.PathValue("id"))
		if id != auth.CurrentUser(r).ID {
			store.ToggleFamilyAdmin(db, id)
		}
		http.Redirect(w, r, "/usuarios", http.StatusSeeOther)
	})
}
