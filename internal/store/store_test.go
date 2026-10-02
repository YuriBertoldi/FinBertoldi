package store

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashSenhaERoundTrip(t *testing.T) {
	senha := "minha-senha-super-secreta-123"
	hash, err := HashSenha(senha)
	if err != nil {
		t.Fatalf("HashSenha erro: %v", err)
	}
	if hash == senha {
		t.Error("hash não pode ser igual à senha")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("hash não parece bcrypt: %s", hash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)); err != nil {
		t.Errorf("senha correta não valida: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("senha-errada")); err == nil {
		t.Error("senha errada validou contra hash")
	}
}

func TestHashSenhaUnique(t *testing.T) {
	h1, _ := HashSenha("senha123")
	h2, _ := HashSenha("senha123")
	if h1 == h2 {
		t.Error("hashes da mesma senha devem ser diferentes (salt aleatório)")
	}
}

func TestGooglePending(t *testing.T) {
	token := "test-token-123"
	SaveGooglePending(nil, token, "user@gmail.com", "Test User")

	email, nome, ok := GetGooglePending(nil, token)
	if !ok {
		t.Fatal("GetGooglePending deveria retornar ok=true")
	}
	if email != "user@gmail.com" {
		t.Errorf("email = %q; want 'user@gmail.com'", email)
	}
	if nome != "Test User" {
		t.Errorf("nome = %q; want 'Test User'", nome)
	}

	// Segunda chamada deve retornar vazio (consumido)
	_, _, ok2 := GetGooglePending(nil, token)
	if ok2 {
		t.Error("GetGooglePending deveria retornar ok=false após consumir")
	}
}

func TestGooglePendingTokenInexistente(t *testing.T) {
	_, _, ok := GetGooglePending(nil, "nao-existe")
	if ok {
		t.Error("token inexistente deveria retornar ok=false")
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("FINC_TEST_VAR", "custom")
	defer os.Unsetenv("FINC_TEST_VAR")
	if got := getEnv("FINC_TEST_VAR", "default"); got != "custom" {
		t.Errorf("getEnv com var setada = %q; want 'custom'", got)
	}
	if got := getEnv("FINC_NAO_EXISTE_123", "default"); got != "default" {
		t.Errorf("getEnv sem var = %q; want 'default'", got)
	}
}
