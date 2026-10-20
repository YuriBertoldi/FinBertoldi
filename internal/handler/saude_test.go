package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// O healthcheck so tem valor se falhar quando o banco cai: e isso que impede o
// deploy de terminar "verde" com o app sem banco.
func TestVerificarSaude_RespondeIndisponivelQuandoOBancoNaoResponde(t *testing.T) {
	// Porta 1 em localhost: conexao recusada na hora, sem depender de rede.
	bancoInacessivel, err := sql.Open("postgres", "host=127.0.0.1 port=1 connect_timeout=1 sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer bancoInacessivel.Close()

	resposta := httptest.NewRecorder()
	inicio := time.Now()
	HandleVerificarSaude(bancoInacessivel)(resposta, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if resposta.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d; esperado %d", resposta.Code, http.StatusServiceUnavailable)
	}
	if duracao := time.Since(inicio); duracao > tempoMaximoParaResponderAoBanco+time.Second {
		t.Errorf("demorou %v; o healthcheck precisa responder dentro do limite", duracao)
	}
}
