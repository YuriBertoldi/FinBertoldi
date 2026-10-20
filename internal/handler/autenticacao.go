package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// hashParaEmailSemConta e comparado quando o e-mail nao tem conta, para que a
// resposta leve o mesmo tempo de uma senha errada e nao revele quem tem conta.
var hashParaEmailSemConta, _ = bcrypt.GenerateFromPassword([]byte("senha-que-nunca-confere"), bcrypt.DefaultCost)

const mensagemDeLoginBloqueado = "Muitas tentativas de login. Aguarde 15 minutos e tente de novo."

func HandleLogin(db *sql.DB, limiteDeTentativas *auth.LimiteDeTentativasDeLogin) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := auth.AuthUser(db, r); u != nil && u.Ativo {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			erro := ""
			if r.URL.Query().Get("erro") != "" {
				erro = "Email ou senha incorretos."
			}
			if r.URL.Query().Get("reset") != "" {
				erro = "Senha alterada com sucesso! Faca login."
			}
			renderLogin(w, models.LoginPage{Title: "Login — FinBertoldi", Erro: erro, GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID")})
			return
		}
		r.ParseForm()
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")
		enderecoIP := auth.EnderecoIPDoCliente(r)

		if limiteDeTentativas.EstaBloqueado(email, enderecoIP) {
			w.WriteHeader(http.StatusTooManyRequests)
			renderLogin(w, models.LoginPage{Title: "Login — FinBertoldi", Erro: mensagemDeLoginBloqueado, GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID")})
			return
		}

		user, hash, err := store.GetUserByEmail(db, email)
		contaPodeEntrar := err == nil && user.Ativo
		if !contaPodeEntrar {
			hash = string(hashParaEmailSemConta)
		}
		senhaConfere := bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil
		if !contaPodeEntrar || !senhaConfere {
			limiteDeTentativas.RegistrarFalha(email, enderecoIP)
			log.Printf("login: falha para o IP %s", enderecoIP)
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}
		limiteDeTentativas.RegistrarSucesso(email)

		token := auth.GenerateToken()
		expiry := time.Now().Add(7 * 24 * time.Hour)
		if err := store.CreateSession(db, user.ID, token, expiry); err != nil {
			log.Printf("create session: %v", err)
		}
		auth.SetSessionCookie(w, token, expiry)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func HandleLogout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err == nil {
			store.DeleteSession(db, c.Value)
		}
		auth.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func HandleGoogleCallback(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		idToken := r.FormValue("credential")
		if idToken == "" {
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}

		claims, err := auth.VerifyGoogleIDToken(idToken)
		if err != nil {
			log.Printf("google auth error: %v", err)
			http.Redirect(w, r, "/login?erro=1", http.StatusSeeOther)
			return
		}

		user, _, err := store.GetUserByEmail(db, claims.Email)
		if err == nil && user.Ativo {
			// Usuário existe — fazer login
			token := auth.GenerateToken()
			expiry := time.Now().Add(7 * 24 * time.Hour)
			store.CreateSession(db, user.ID, token, expiry)
			auth.SetSessionCookie(w, token, expiry)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Usuário não existe — redirecionar para cadastro com dados do Google
		googleToken := auth.GenerateToken()
		http.SetCookie(w, &http.Cookie{
			Name:     "google_pending",
			Value:    googleToken,
			Path:     "/",
			MaxAge:   600, // 10 minutos
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		store.SaveGooglePending(db, googleToken, claims.Email, claims.Name)
		http.Redirect(w, r, "/register?google=1", http.StatusSeeOther)
	}
}

func HandleRegister(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := auth.AuthUser(db, r); u != nil && u.Ativo {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodGet {
			data := models.RegisterPage{Title: "Cadastro"}
			if r.URL.Query().Get("google") == "1" {
				if c, err := r.Cookie("google_pending"); err == nil {
					email, nome, _ := store.GetGooglePending(db, c.Value)
					data.GoogleEmail = email
					data.GoogleNome = nome
					data.IsGoogle = true
				}
			}
			renderStandalone(w, "register", data)
			return
		}

		// POST — criar família + usuário
		r.ParseForm()
		nome := strings.TrimSpace(r.FormValue("nome"))
		email := strings.TrimSpace(r.FormValue("email"))
		senha := r.FormValue("senha")
		familyNome := strings.TrimSpace(r.FormValue("familia"))
		isGoogle := r.FormValue("is_google") == "1"

		if nome == "" || email == "" || familyNome == "" {
			renderStandalone(w, "register", models.RegisterPage{
				Title: "Cadastro", Erro: "Preencha todos os campos.",
			})
			return
		}

		// Verificar email duplicado
		if existing, _, err := store.GetUserByEmail(db, email); err == nil && existing.ID > 0 {
			renderStandalone(w, "register", models.RegisterPage{
				Title: "Cadastro", Erro: "Este email já está cadastrado.",
			})
			return
		}

		// Senha obrigatória para cadastro sem Google
		senhaHash := ""
		if isGoogle {
			// Google users: senha aleatória (login via Google)
			b := make([]byte, 32)
			rand.Read(b)
			h, _ := bcrypt.GenerateFromPassword(b, bcrypt.DefaultCost)
			senhaHash = string(h)
		} else {
			if len(senha) < 6 {
				renderStandalone(w, "register", models.RegisterPage{
					Title: "Cadastro", Erro: "Senha deve ter no mínimo 6 caracteres.",
				})
				return
			}
			h, _ := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
			senhaHash = string(h)
		}

		// Criar família
		familyID, err := store.CreateFamily(db, familyNome)
		if err != nil {
			renderStandalone(w, "register", models.RegisterPage{
				Title: "Cadastro", Erro: "Erro ao criar família: " + err.Error(),
			})
			return
		}

		// Criar usuário como family_admin
		userID, err := store.CreateUserReturningID(db, nome, email, senhaHash, true, familyID)
		if err != nil {
			renderStandalone(w, "register", models.RegisterPage{
				Title: "Cadastro", Erro: "Erro ao criar usuário: " + err.Error(),
			})
			return
		}

		// Login automático
		token := auth.GenerateToken()
		expiry := time.Now().Add(7 * 24 * time.Hour)
		store.CreateSession(db, userID, token, expiry)
		auth.SetSessionCookie(w, token, expiry)

		// Limpar cookie do Google
		http.SetCookie(w, &http.Cookie{
			Name:   "google_pending",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func HandleForgotPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			msg := ""
			if r.URL.Query().Get("ok") != "" {
				msg = "Se o email estiver cadastrado, enviaremos um link de recuperacao."
			}
			renderStandalone(w, "forgot_password", map[string]string{"Msg": msg})
			return
		}
		r.ParseForm()
		email := strings.TrimSpace(r.FormValue("email"))

		// Sempre redireciona com mensagem genérica (segurança)
		userID, err := store.GetUserIDByEmail(db, email)
		if err == nil && userID > 0 {
			tokenBytes := make([]byte, 32)
			rand.Read(tokenBytes)
			token := hex.EncodeToString(tokenBytes)
			expiry := time.Now().Add(1 * time.Hour)

			if err := store.CreatePasswordReset(db, userID, token, expiry); err != nil {
				log.Printf("[password-reset] erro ao criar token: %v", err)
			} else {
				baseURL := os.Getenv("BASE_URL")
				if baseURL == "" {
					baseURL = "http://localhost:8080"
				}
				if err := auth.SendResetEmail(email, token, baseURL); err != nil {
					log.Printf("[password-reset] erro ao enviar email: %v", err)
				}
			}
		}
		http.Redirect(w, r, "/forgot-password?ok=1", http.StatusSeeOther)
	}
}

func HandleResetPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if r.Method == http.MethodGet {
			if token == "" {
				renderStandalone(w, "reset_password", map[string]string{"Erro": "Link invalido."})
				return
			}
			_, err := store.GetPasswordReset(db, token)
			if err != nil {
				renderStandalone(w, "reset_password", map[string]string{"Erro": "Link expirado ou invalido."})
				return
			}
			renderStandalone(w, "reset_password", map[string]string{"Token": token})
			return
		}
		r.ParseForm()
		token = r.FormValue("token")
		senha := r.FormValue("senha")
		confirmar := r.FormValue("confirmar")

		if senha != confirmar {
			renderStandalone(w, "reset_password", map[string]string{"Token": token, "Erro": "As senhas nao coincidem."})
			return
		}
		if len(senha) < 6 {
			renderStandalone(w, "reset_password", map[string]string{"Token": token, "Erro": "A senha deve ter pelo menos 6 caracteres."})
			return
		}

		userID, err := store.GetPasswordReset(db, token)
		if err != nil {
			renderStandalone(w, "reset_password", map[string]string{"Erro": "Link expirado ou invalido."})
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
		if err != nil {
			renderStandalone(w, "reset_password", map[string]string{"Erro": "Erro interno."})
			return
		}

		store.UpdateUserPassword(db, userID, string(hashed))
		store.UsePasswordReset(db, token)

		http.Redirect(w, r, "/login?reset=1", http.StatusSeeOther)
	}
}
