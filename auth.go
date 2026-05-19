package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

type ctxKey string

const ctxUserKey ctxKey = "user"

// generateToken gera um token aleatório de 64 chars hex
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// protected exige sessão válida; injeta usuário no contexto
func protected(app *App, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := authUser(app, r)
		if u == nil || !u.Ativo {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey, u)
		h(w, r.WithContext(ctx))
	}
}

// adminOnly exige sessão válida + flag admin = true
func adminOnly(app *App, h http.HandlerFunc) http.HandlerFunc {
	return protected(app, func(w http.ResponseWriter, r *http.Request) {
		if u := currentUser(r); u == nil || !u.Admin {
			http.Error(w, "Acesso restrito a administradores.", http.StatusForbidden)
			return
		}
		h(w, r)
	})
}

// authUser lê o cookie de sessão e retorna o usuário autenticado (ou nil)
func authUser(app *App, r *http.Request) *User {
	c, err := r.Cookie("session")
	if err != nil {
		return nil
	}
	u, err := dbGetUserBySession(app.db, c.Value)
	if err != nil {
		return nil
	}
	return u
}

// currentUser retorna o usuário injetado no contexto pelo middleware protected
func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(ctxUserKey).(*User)
	return u
}

func setSessionCookie(w http.ResponseWriter, token string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
