package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

type ctxKey string

const ctxUserKey ctxKey = "user"

// GenerateToken gera um token aleatório de 64 chars hex.
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SetSessionCookie define o cookie de sessão.
func SetSessionCookie(w http.ResponseWriter, token string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie remove o cookie de sessão.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// CurrentUser retorna o usuário injetado no contexto pelo middleware Protected.
func CurrentUser(r *http.Request) *models.User {
	u, _ := r.Context().Value(ctxUserKey).(*models.User)
	return u
}

// AuthUser lê o cookie de sessão e retorna o usuário autenticado (ou nil).
func AuthUser(db *sql.DB, r *http.Request) *models.User {
	c, err := r.Cookie("session")
	if err != nil {
		return nil
	}
	u, err := store.GetUserBySession(db, c.Value)
	if err != nil {
		return nil
	}
	return u
}

// Protected exige sessão válida; injeta usuário no contexto.
func Protected(db *sql.DB, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := AuthUser(db, r)
		if u == nil || !u.Ativo {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, u)
		h(w, r.WithContext(ctx))
	}
}

// AdminOnly exige sessão válida + flag admin = true.
func AdminOnly(db *sql.DB, h http.HandlerFunc) http.HandlerFunc {
	return Protected(db, func(w http.ResponseWriter, r *http.Request) {
		if u := CurrentUser(r); u == nil || !u.Admin {
			http.Error(w, "Acesso restrito a administradores.", http.StatusForbidden)
			return
		}
		h(w, r)
	})
}

// FamilyAdminOnly exige sessão válida + (family_admin = true OU admin = true).
func FamilyAdminOnly(db *sql.DB, h http.HandlerFunc) http.HandlerFunc {
	return Protected(db, func(w http.ResponseWriter, r *http.Request) {
		if u := CurrentUser(r); u == nil || (!u.FamilyAdmin && !u.Admin) {
			http.Error(w, "Acesso restrito a administradores de família.", http.StatusForbidden)
			return
		}
		h(w, r)
	})
}

// ScreenProtected exige sessão válida e verifica se a tela está liberada para o usuário.
// Admin e family_admin sempre têm acesso. Usuários comuns são bloqueados se a tela
// estiver em user_permissions para o seu ID.
func ScreenProtected(db *sql.DB, screen string, h http.HandlerFunc) http.HandlerFunc {
	return Protected(db, func(w http.ResponseWriter, r *http.Request) {
		u := CurrentUser(r)
		if !u.Admin && !u.FamilyAdmin {
			blocked, _ := store.GetBlockedScreens(db, u.ID)
			for _, s := range blocked {
				if s == screen {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprintf(w, `<!DOCTYPE html><html lang="pt-BR"><head><meta charset="UTF-8"><title>Acesso bloqueado</title></head><body style="font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#0f1419;color:#e6e9ed"><div style="text-align:center;padding:2rem"><p style="font-size:2rem">🔒</p><h2>Acesso bloqueado</h2><p style="color:#6a7385">O administrador da sua família bloqueou o acesso a esta tela.</p><a href="/" style="color:#27ae60">← Voltar ao Dashboard</a></div></body></html>`)
					return
				}
			}
		}
		h(w, r)
	})
}
