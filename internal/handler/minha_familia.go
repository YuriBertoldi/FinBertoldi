package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// TelasDispo lista as telas que podem ter acesso controlado.
var TelasDispo = []models.ScreenInfo{
	{Key: "despesas", Label: "💸 Despesas"},
	{Key: "receitas", Label: "💵 Receitas"},
	{Key: "investimentos", Label: "📈 Investimentos"},
	{Key: "planejamento", Label: "🎯 Planejamento"},
	{Key: "emprestimos", Label: "🤝 Empréstimos"},
	{Key: "cadastros", Label: "🗂️ Cadastros"},
}

func HandleMeuTime(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		cu := auth.CurrentUser(r)
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		for i := range membros {
			membros[i].BlockedScreens, _ = store.GetBlockedScreens(db, membros[i].ID)
		}
		var erro, sucesso string
		if e := r.URL.Query().Get("erro"); e != "" {
			switch e {
			case "campos":
				erro = "Preencha nome, e-mail e senha."
			case "email":
				erro = "E-mail já cadastrado ou inválido."
			case "hash":
				erro = "Erro ao processar senha."
			default:
				erro = e
			}
		}
		if r.URL.Query().Get("ok") == "1" {
			sucesso = "Operação realizada com sucesso."
		}
		render(w, "minha-familia", models.MeuTimePage{
			BasePage: bp(db, r, "minha-familia", "Minha Família"),
			Membros:  membros,
			Telas:    TelasDispo,
			Erro:     erro,
			Sucesso:  sucesso,
		})
	})
}

func HandleToggleScreenPermission(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		cu := auth.CurrentUser(r)
		idStr := r.PathValue("id")
		tela := r.PathValue("tela")
		targetID, err := strconv.Atoi(idStr)
		if err != nil {
			http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
			return
		}
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		var targetOK bool
		for _, m := range membros {
			if m.ID == targetID && !m.Admin {
				targetOK = true
				break
			}
		}
		if !targetOK {
			http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
			return
		}
		for _, t := range TelasDispo {
			if t.Key == tela {
				store.ToggleScreenPermission(db, targetID, tela)
				break
			}
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMeuTimeCreateMembro(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cu := auth.CurrentUser(r)
		nome := strings.TrimSpace(r.FormValue("nome"))
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")
		if nome == "" || email == "" || senha == "" {
			http.Redirect(w, r, "/minha-familia?erro=campos", http.StatusSeeOther)
			return
		}
		hash, err := store.HashSenha(senha)
		if err != nil {
			http.Redirect(w, r, "/minha-familia?erro=hash", http.StatusSeeOther)
			return
		}
		if err := store.CreateUser(db, nome, email, hash, false, cu.FamilyID); err != nil {
			log.Printf("minha-familia create user: %v", err)
			http.Redirect(w, r, "/minha-familia?erro=email", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMeuTimeToggleAtivo(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		cu := auth.CurrentUser(r)
		id := parseInt(r.PathValue("id"))
		if id == cu.ID {
			http.Redirect(w, r, "/minha-familia", http.StatusSeeOther)
			return
		}
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		for _, m := range membros {
			if m.ID == id {
				store.ToggleUserAtivo(db, id)
				break
			}
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

func HandleMeuTimeResetSenha(db *sql.DB) http.HandlerFunc {
	return auth.FamilyAdminOnly(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cu := auth.CurrentUser(r)
		id := parseInt(r.PathValue("id"))
		senha := r.FormValue("nova_senha")
		if msg := validarNovaSenha(senha, ""); msg != "" {
			http.Redirect(w, r, "/minha-familia?erro="+url.QueryEscape(msg), http.StatusSeeOther)
			return
		}
		membros, _ := store.GetFamilyMembers(db, cu.FamilyID)
		for _, m := range membros {
			if m.ID == id {
				if m.Admin && !cu.Admin {
					http.Redirect(w, r, "/minha-familia", http.StatusSeeOther)
					return
				}
				if hash, err := store.HashSenha(senha); err == nil {
					store.ResetSenha(db, id, hash)
				}
				break
			}
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}
