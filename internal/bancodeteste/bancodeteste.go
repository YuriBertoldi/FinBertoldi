//go:build integration

// Package bancodeteste abre um PostgreSQL isolado para os testes de
// integracao dos pacotes que dependem do store (handler, auth). O proprio
// store usa a copia interna dele (setupTestDB), porque importar este pacote
// de dentro do store criaria um ciclo de import.
package bancodeteste

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"fincontrol/internal/store"

	_ "github.com/lib/pq"
)

// AbrirBancoIsolado cria um schema temporario com todas as migrations e o
// apaga no fim do teste. Sem banco, pula o teste; no CI (INTEGRATION_REQUIRED=1)
// a falta do banco e erro.
func AbrirBancoIsolado(t *testing.T) *sql.DB {
	t.Helper()
	enderecoDoBanco := os.Getenv("DATABASE_URL")
	if enderecoDoBanco == "" {
		enderecoDoBanco = "host=localhost port=5432 user=fincontrol password=fincontrol dbname=fincontrol sslmode=disable"
	}
	pularOuFalhar := t.Skipf
	if os.Getenv("INTEGRATION_REQUIRED") == "1" {
		pularOuFalhar = t.Fatalf
	}
	db, err := sql.Open("postgres", enderecoDoBanco)
	if err != nil {
		pularOuFalhar("postgres nao disponivel: %v", err)
	}
	if err := db.Ping(); err != nil {
		pularOuFalhar("postgres nao responde: %v", err)
	}

	// Uma conexao so: o search_path vale por conexao.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	schema := fmt.Sprintf("teste_%d", time.Now().UnixNano())
	for _, comando := range []string{
		fmt.Sprintf(`CREATE SCHEMA %s`, schema),
		fmt.Sprintf(`SET search_path TO %s`, schema),
		`SET TIME ZONE 'UTC'`,
	} {
		if _, err := db.Exec(comando); err != nil {
			t.Fatalf("%s: %v", comando, err)
		}
	}
	t.Cleanup(func() {
		db.Exec(`SET search_path TO public`)
		db.Exec(fmt.Sprintf(`DROP SCHEMA %s CASCADE`, schema))
		db.Close()
	})

	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return db
}

// CriarFamilia cria uma familia e devolve o id.
func CriarFamilia(t *testing.T, db *sql.DB, nome string) int {
	t.Helper()
	id, err := store.CreateFamily(db, nome)
	if err != nil {
		t.Fatalf("criar familia %q: %v", nome, err)
	}
	return id
}

// CriarUsuario cria um usuario ativo na familia e devolve o id.
func CriarUsuario(t *testing.T, db *sql.DB, email string, administradorDaFamilia bool, familiaID int) int {
	t.Helper()
	hash, err := store.HashSenha("senha-de-teste-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	id, err := store.CreateUserReturningID(db, email, email, hash, administradorDaFamilia, familiaID)
	if err != nil {
		t.Fatalf("criar usuario %q: %v", email, err)
	}
	return id
}

// CookieDeSessao cria uma sessao valida para o usuario, como o login faria.
func CookieDeSessao(t *testing.T, db *sql.DB, usuarioID int) *http.Cookie {
	t.Helper()
	token := fmt.Sprintf("sessao-de-teste-%d-%d", usuarioID, time.Now().UnixNano())
	if err := store.CreateSession(db, usuarioID, token, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("criar sessao: %v", err)
	}
	return &http.Cookie{Name: "session", Value: token}
}
