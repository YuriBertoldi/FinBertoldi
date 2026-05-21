package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fincontrol/internal/models"
)

func TestGenerateToken(t *testing.T) {
	tok1 := GenerateToken()
	tok2 := GenerateToken()

	if len(tok1) != 64 {
		t.Errorf("token deve ter 64 chars hex, tem %d", len(tok1))
	}
	if tok1 == tok2 {
		t.Error("dois tokens consecutivos não devem ser iguais")
	}
	for _, c := range tok1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contém char não-hex: %c", c)
		}
	}
}

func TestGenerateTokenUnique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		tok := GenerateToken()
		if seen[tok] {
			t.Fatalf("colisão de token na iteração %d: %s", i, tok)
		}
		seen[tok] = true
	}
}

func TestSetSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	expiry := time.Now().Add(7 * 24 * time.Hour)
	SetSessionCookie(w, "meu-token-abc123", expiry)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("esperado 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "session" {
		t.Errorf("nome do cookie = %q; want 'session'", c.Name)
	}
	if c.Value != "meu-token-abc123" {
		t.Errorf("valor = %q; want 'meu-token-abc123'", c.Value)
	}
	if !c.HttpOnly {
		t.Error("cookie de sessão deve ser HttpOnly")
	}
	if c.Path != "/" {
		t.Errorf("path = %q; want '/'", c.Path)
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Error("SameSite deve ser Lax")
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("esperado 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != "session" {
		t.Errorf("nome = %q; want 'session'", c.Name)
	}
	if c.Value != "" {
		t.Errorf("valor deve ser vazio para limpar, got %q", c.Value)
	}
	if c.MaxAge != -1 {
		t.Errorf("MaxAge = %d; want -1", c.MaxAge)
	}
}

func TestCurrentUserContext(t *testing.T) {
	u := &models.User{ID: 42, Nome: "Teste", Email: "t@t.com", FamilyID: 7}
	r := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(r.Context(), ctxUserKey, u)
	r = r.WithContext(ctx)

	got := CurrentUser(r)
	if got == nil {
		t.Fatal("CurrentUser retornou nil; esperado user")
	}
	if got.ID != 42 || got.FamilyID != 7 {
		t.Errorf("user retornado errado: %+v", got)
	}
}

func TestCurrentUserSemContexto(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	got := CurrentUser(r)
	if got != nil {
		t.Errorf("CurrentUser sem contexto deve retornar nil, got %+v", got)
	}
}

func TestProtectedRedirectsSemSessao(t *testing.T) {
	called := false
	h := Protected(nil, func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h(w, r)

	if called {
		t.Error("handler interno foi chamado mesmo sem sessão")
	}
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d; want %d (SeeOther)", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q; want '/login'", loc)
	}
}

func TestAdminOnlyBloqueiaUsuarioComum(t *testing.T) {
	called := false
	h := func(w http.ResponseWriter, r *http.Request) {
		called = true
	}

	u := &models.User{ID: 1, Nome: "User", Admin: false, Ativo: true}
	r := httptest.NewRequest("GET", "/usuarios", nil)
	ctx := context.WithValue(r.Context(), ctxUserKey, u)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	inner := func(w http.ResponseWriter, r *http.Request) {
		if cu := CurrentUser(r); cu == nil || !cu.Admin {
			http.Error(w, "Acesso restrito a administradores.", http.StatusForbidden)
			return
		}
		h(w, r)
	}
	inner(w, r)

	if called {
		t.Error("handler admin foi chamado para usuário não-admin")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d; want %d (Forbidden)", w.Code, http.StatusForbidden)
	}
}
