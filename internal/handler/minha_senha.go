package handler

import (
	"database/sql"
	"fincontrol/internal/auth"
	"fincontrol/internal/store"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HandleMinhaSenha troca a senha do próprio usuário: exige a senha atual e encerra
// as sessões abertas em outros aparelhos (a atual continua).
func HandleMinhaSenha(db *sql.DB) http.HandlerFunc {
	return auth.Protected(db, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cu := auth.CurrentUser(r)
		volta := func(erro string) {
			http.Redirect(w, r, "/minha-familia?erro="+url.QueryEscape(erro), http.StatusSeeOther)
		}
		_, hashAtual, err := store.GetUserByEmail(db, cu.Email)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hashAtual), []byte(r.FormValue("senha_atual"))) != nil {
			volta("Senha atual incorreta.")
			return
		}
		senha := r.FormValue("nova_senha")
		if msg := validarNovaSenha(senha, r.FormValue("senha_atual")); msg != "" {
			volta(msg)
			return
		}
		hash, err := store.HashSenha(senha)
		if err != nil {
			volta("Erro ao processar senha.")
			return
		}
		atual := ""
		if c, err := r.Cookie("session"); err == nil {
			atual = c.Value
		}
		if err := store.TrocarSenha(db, cu.ID, hash, atual); err != nil {
			volta("Erro ao salvar a nova senha.")
			return
		}
		http.Redirect(w, r, "/minha-familia?ok=1", http.StatusSeeOther)
	})
}

// validarNovaSenha devolve a mensagem de erro, ou "" se a senha nova é aceitável.
func validarNovaSenha(nova, atual string) string {
	switch {
	case len([]rune(nova)) < 8:
		return "A nova senha precisa ter pelo menos 8 caracteres."
	case nova == atual:
		return "A nova senha precisa ser diferente da atual."
	case strings.EqualFold(nova, "admin123") || strings.EqualFold(nova, "12345678") || strings.EqualFold(nova, "changeme"):
		return "Essa senha é conhecida demais. Escolha outra."
	}
	return ""
}
