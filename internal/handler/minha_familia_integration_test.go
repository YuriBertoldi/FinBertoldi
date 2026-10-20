//go:build integration

package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"fincontrol/internal/bancodeteste"
)

// As rotas de "Minha Familia" chamam o store so com o id do usuario-alvo (sem
// family_id). Quem impede o administrador de uma familia de mexer em
// usuario de outra e a checagem de pertencimento dentro do handler. Se ela
// sumir, um administrador consegue, por exemplo, trocar a senha de alguem de
// outra familia e entrar na conta dele.
func TestMinhaFamilia_AdministradorNaoAlteraUsuarioDeOutraFamilia(t *testing.T) {
	casos := []struct {
		nome       string
		handler    func(*sql.DB) http.HandlerFunc
		caminho    string
		valores    map[string]string
		formulario url.Values
	}{
		{
			nome:    "desativar",
			handler: HandleMeuTimeToggleAtivo,
			caminho: "/minha-familia/{id}/toggle-ativo",
		},
		{
			nome:       "trocar a senha",
			handler:    HandleMeuTimeResetSenha,
			caminho:    "/minha-familia/{id}/reset-senha",
			formulario: url.Values{"nova_senha": {"senha-do-invasor-987"}},
		},
		{
			nome:    "bloquear uma tela",
			handler: HandleToggleScreenPermission,
			caminho: "/minha-familia/{id}/permissoes/{tela}/toggle",
			valores: map[string]string{"tela": "despesas"},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			db := bancodeteste.AbrirBancoIsolado(t)
			familiaDaVitima := bancodeteste.CriarFamilia(t, db, "Familia da vitima")
			familiaDoAdministrador := bancodeteste.CriarFamilia(t, db, "Familia do administrador")
			vitimaID := bancodeteste.CriarUsuario(t, db, "vitima@exemplo.com", false, familiaDaVitima)
			administradorID := bancodeteste.CriarUsuario(t, db, "admin-b@exemplo.com", true, familiaDoAdministrador)

			antes := estadoDoUsuario(t, db, vitimaID)
			executarComoUsuario(t, db, administradorID, caso.handler, vitimaID, caso.valores, caso.formulario)
			if depois := estadoDoUsuario(t, db, vitimaID); depois != antes {
				t.Errorf("POST %s: administrador de outra familia alterou o usuario:\nantes:  %s\ndepois: %s", caso.caminho, antes, depois)
			}

			// Controle: o mesmo pedido, feito pelo administrador da familia
			// certa, precisa mudar algo. Sem isso o teste passaria mesmo com
			// a rota quebrada para todo mundo.
			administradorDaVitimaID := bancodeteste.CriarUsuario(t, db, "admin-a@exemplo.com", true, familiaDaVitima)
			executarComoUsuario(t, db, administradorDaVitimaID, caso.handler, vitimaID, caso.valores, caso.formulario)
			if depois := estadoDoUsuario(t, db, vitimaID); depois == antes {
				t.Error("o administrador da propria familia tambem nao conseguiu alterar; o teste nao esta exercitando a rota")
			}
		})
	}
}

func executarComoUsuario(t *testing.T, db *sql.DB, usuarioID int, handler func(*sql.DB) http.HandlerFunc, alvoID int, valoresDoCaminho map[string]string, formulario url.Values) {
	t.Helper()
	requisicao := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formulario.Encode()))
	requisicao.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requisicao.AddCookie(bancodeteste.CookieDeSessao(t, db, usuarioID))
	requisicao.SetPathValue("id", strconv.Itoa(alvoID))
	for nome, valor := range valoresDoCaminho {
		requisicao.SetPathValue(nome, valor)
	}
	handler(db)(httptest.NewRecorder(), requisicao)
}

// estadoDoUsuario resume tudo o que essas rotas podem alterar.
func estadoDoUsuario(t *testing.T, db *sql.DB, usuarioID int) string {
	t.Helper()
	var estado string
	err := db.QueryRow(`
		SELECT u.ativo::text || '|' || u.senha_hash || '|' ||
		       COALESCE((SELECT string_agg(tela, ',' ORDER BY tela) FROM user_permissions WHERE user_id = u.id), '')
		FROM users u WHERE u.id = $1`, usuarioID).Scan(&estado)
	if err != nil {
		t.Fatalf("ler estado do usuario: %v", err)
	}
	return estado
}
