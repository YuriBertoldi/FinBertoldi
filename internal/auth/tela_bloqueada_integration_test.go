//go:build integration

package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fincontrol/internal/auth"
	"fincontrol/internal/bancodeteste"
	"fincontrol/internal/store"
)

// O caminho de bloqueio do ScreenProtected depende do banco
// (user_permissions) e nao tinha teste: so o caminho que libera o admin.
func TestScreenProtected_RecusaTelaBloqueadaParaOUsuario(t *testing.T) {
	db := bancodeteste.AbrirBancoIsolado(t)
	familiaID := bancodeteste.CriarFamilia(t, db, "Familia")
	membroID := bancodeteste.CriarUsuario(t, db, "membro@exemplo.com", false, familiaID)
	if _, err := store.ToggleScreenPermission(db, membroID, "despesas"); err != nil {
		t.Fatalf("bloquear tela: %v", err)
	}

	handlerFoiChamado := false
	rotaProtegida := auth.ScreenProtected(db, "despesas", func(w http.ResponseWriter, r *http.Request) {
		handlerFoiChamado = true
	})

	requisicao := httptest.NewRequest(http.MethodGet, "/despesas", nil)
	requisicao.AddCookie(bancodeteste.CookieDeSessao(t, db, membroID))
	resposta := httptest.NewRecorder()
	rotaProtegida(resposta, requisicao)

	if resposta.Code != http.StatusForbidden {
		t.Errorf("status = %d; esperado %d", resposta.Code, http.StatusForbidden)
	}
	if handlerFoiChamado {
		t.Error("a tela bloqueada chegou a executar o handler")
	}

	// A mesma rota, para uma tela que nao foi bloqueada, continua liberada.
	outraTela := auth.ScreenProtected(db, "receitas", func(w http.ResponseWriter, r *http.Request) {
		handlerFoiChamado = true
	})
	requisicao = httptest.NewRequest(http.MethodGet, "/receitas", nil)
	requisicao.AddCookie(bancodeteste.CookieDeSessao(t, db, membroID))
	outraTela(httptest.NewRecorder(), requisicao)
	if !handlerFoiChamado {
		t.Error("bloquear uma tela bloqueou tambem outra")
	}
}
