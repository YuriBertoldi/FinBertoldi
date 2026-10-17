//go:build integration

package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"fincontrol/internal/store"
	_ "github.com/lib/pq"
)

func bancoDeTeste(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err == nil {
		err = db.Ping()
	}
	if err != nil {
		if os.Getenv("INTEGRATION_REQUIRED") == "1" {
			t.Fatalf("postgres indisponível: %v", err)
		}
		t.Skipf("postgres indisponível: %v", err)
	}
	db.SetMaxOpenConns(1)
	schema := fmt.Sprintf("test_sheets_%d", time.Now().UnixNano())
	db.Exec(`CREATE SCHEMA ` + schema)
	db.Exec(`SET search_path TO ` + schema)
	t.Cleanup(func() {
		db.Exec(`SET search_path TO public`)
		db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		db.Close()
	})
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return db
}

// Fluxo completo: configuração salva no banco → token → abas criadas e gravadas com os dados da família.
func TestExportToSheet_PontaAPonta(t *testing.T) {
	db := bancoDeTeste(t)
	fid, _ := store.CreateFamily(db, "Família Sheets")
	outra, _ := store.CreateFamily(db, "Outra Família")
	store.CreateDespesa(db, fid, "Energia", 500, "Básicas")
	store.CreateDespesa(db, outra, "NÃO PODE APARECER", 999, "Básicas")
	store.CreateReceita(db, fid, "Salário", 13000, time.Now(), "salario", true)
	inicio := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	store.CreateParcelamento(db, fid, "Carro", "Boleto", 2514, 3, 36, inicio, false, 0, 0)

	var gravado struct {
		Data []struct {
			Range  string  `json:"range"`
			Values [][]any `json:"values"`
		} `json:"data"`
	}
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			io.WriteString(w, `{"access_token":"tok-e2e"}`)
		case r.Header.Get("Authorization") != "Bearer tok-e2e":
			w.WriteHeader(401)
		case r.Method == "GET":
			io.WriteString(w, `{"sheets":[{"properties":{"title":"Página1"}}]}`)
		case strings.HasSuffix(r.URL.Path, "/values:batchUpdate"):
			json.NewDecoder(r.Body).Decode(&gravado)
			io.WriteString(w, `{}`)
		default:
			io.WriteString(w, `{}`)
		}
	}))
	defer google.Close()
	antes := sheetsAPIBase
	sheetsAPIBase = google.URL
	defer func() { sheetsAPIBase = antes }()

	js, _ := gerarServiceAccountJSON(t, google.URL+"/token")
	store.SetIntegracaoKV(db, fid, "sheets", "service_account_json", js)
	store.SetIntegracaoKV(db, fid, "sheets", "spreadsheet_id", "https://docs.google.com/spreadsheets/d/abc123/edit")

	if err := ExportToSheet(db, fid); err != nil {
		t.Fatalf("ExportToSheet: %v", err)
	}
	if len(gravado.Data) != 6 {
		t.Fatalf("esperava 6 abas gravadas, got %d", len(gravado.Data))
	}
	tudo, _ := json.Marshal(gravado)
	for _, deve := range []string{"Energia", "Salário", "Carro", "3/36", "'Contas do mês'!A1"} {
		if !strings.Contains(string(tudo), deve) {
			t.Errorf("planilha deveria conter %q", deve)
		}
	}
	if strings.Contains(string(tudo), "NÃO PODE APARECER") {
		t.Error("dados de outra família vazaram para a planilha")
	}
	if store.GetIntegracaoKV(db, fid, "sheets", "ultima_exportacao") == "" {
		t.Error("deveria registrar a data da última exportação")
	}
}

func TestExportToSheet_SemConfiguracao(t *testing.T) {
	db := bancoDeTeste(t)
	fid, _ := store.CreateFamily(db, "Sem config")
	if err := ExportToSheet(db, fid); err == nil || !strings.Contains(err.Error(), "informe") {
		t.Errorf("sem configuração deveria orientar o usuário, got %v", err)
	}
}
