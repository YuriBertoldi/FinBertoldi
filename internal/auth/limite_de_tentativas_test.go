package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

// relogioDeTeste deixa o teste avancar o tempo sem dormir.
type relogioDeTeste struct{ momento time.Time }

func (relogio *relogioDeTeste) agora() time.Time        { return relogio.momento }
func (relogio *relogioDeTeste) avancar(d time.Duration) { relogio.momento = relogio.momento.Add(d) }

func novoLimiteComRelogio() (*LimiteDeTentativasDeLogin, *relogioDeTeste) {
	relogio := &relogioDeTeste{momento: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)}
	limite := NovoLimiteDeTentativasDeLogin()
	limite.agora = relogio.agora
	return limite, relogio
}

func registrarFalhas(limite *LimiteDeTentativasDeLogin, quantidade int, email, enderecoIP string) {
	for i := 0; i < quantidade; i++ {
		limite.RegistrarFalha(email, enderecoIP)
	}
}

func TestLimiteDeLogin_BloqueiaOEmailNaQuintaSenhaErrada(t *testing.T) {
	limite, _ := novoLimiteComRelogio()

	registrarFalhas(limite, 4, "ana@exemplo.com", "203.0.113.1")
	if limite.EstaBloqueado("ana@exemplo.com", "203.0.113.2") {
		t.Fatal("bloqueou com 4 falhas; o limite e 5")
	}

	limite.RegistrarFalha("ana@exemplo.com", "203.0.113.1")
	if !limite.EstaBloqueado("ana@exemplo.com", "203.0.113.2") {
		t.Error("nao bloqueou o e-mail na 5a falha, mesmo vindo de outro IP")
	}
}

// Trocar maiusculas ou colocar espacos nao pode virar um contador novo.
func TestLimiteDeLogin_ContaOMesmoEmailIndependenteDeMaiusculasEEspacos(t *testing.T) {
	limite, _ := novoLimiteComRelogio()

	for _, variacao := range []string{"ana@exemplo.com", "ANA@exemplo.com", " Ana@Exemplo.com", "ana@EXEMPLO.COM ", "aNa@exemplo.com"} {
		limite.RegistrarFalha(variacao, "203.0.113.1")
	}

	if !limite.EstaBloqueado("ana@exemplo.com", "198.51.100.9") {
		t.Error("variacoes do mesmo e-mail escaparam do limite")
	}
}

// Ataque de "spray": uma senha contra muitos e-mails, a partir do mesmo IP.
func TestLimiteDeLogin_BloqueiaOIPQueErraEmMuitosEmailsDiferentes(t *testing.T) {
	limite, _ := novoLimiteComRelogio()

	for i := 0; i < 20; i++ {
		limite.RegistrarFalha("usuario"+string(rune('a'+i))+"@exemplo.com", "203.0.113.1")
	}

	if !limite.EstaBloqueado("nunca-tentado@exemplo.com", "203.0.113.1") {
		t.Error("IP com 20 falhas em e-mails diferentes nao foi bloqueado")
	}
	if limite.EstaBloqueado("nunca-tentado@exemplo.com", "203.0.113.2") {
		t.Error("o bloqueio de um IP atingiu outro IP")
	}
}

func TestLimiteDeLogin_LiberaQuandoOBloqueioTermina(t *testing.T) {
	limite, relogio := novoLimiteComRelogio()
	registrarFalhas(limite, 5, "ana@exemplo.com", "203.0.113.1")

	relogio.avancar(15*time.Minute - time.Second)
	if !limite.EstaBloqueado("ana@exemplo.com", "203.0.113.1") {
		t.Fatal("liberou antes dos 15 minutos")
	}

	relogio.avancar(time.Second)
	if limite.EstaBloqueado("ana@exemplo.com", "203.0.113.1") {
		t.Error("continuou bloqueado depois dos 15 minutos")
	}
}

// Quem erra a senha de vez em quando nunca deve ser bloqueado.
func TestLimiteDeLogin_FalhasForaDaJanelaNaoSeAcumulam(t *testing.T) {
	limite, relogio := novoLimiteComRelogio()

	for i := 0; i < 10; i++ {
		limite.RegistrarFalha("ana@exemplo.com", "203.0.113.1")
		relogio.avancar(16 * time.Minute)
	}

	if limite.EstaBloqueado("ana@exemplo.com", "203.0.113.1") {
		t.Error("bloqueou falhas espalhadas em janelas diferentes")
	}
}

func TestLimiteDeLogin_SucessoZeraOEmailMasNaoOIP(t *testing.T) {
	limite, _ := novoLimiteComRelogio()

	registrarFalhas(limite, 4, "ana@exemplo.com", "203.0.113.1")
	limite.RegistrarSucesso("ana@exemplo.com")
	limite.RegistrarFalha("ana@exemplo.com", "203.0.113.1")
	if limite.EstaBloqueado("ana@exemplo.com", "198.51.100.9") {
		t.Error("o login certo nao zerou as falhas do e-mail")
	}

	// Atacante alternando com a propria conta para tentar zerar o IP.
	registrarFalhas(limite, 15, "vitima@exemplo.com", "203.0.113.1")
	limite.RegistrarSucesso("conta-do-atacante@exemplo.com")
	if !limite.EstaBloqueado("outra@exemplo.com", "203.0.113.1") {
		t.Error("login certo de outra conta zerou o contador do IP")
	}
}

func TestEnderecoIPDoCliente(t *testing.T) {
	casos := []struct {
		nome            string
		enderecoRemoto  string
		encaminhadoPara string
		esperado        string
	}{
		{"atras do Caddy usa o IP que ele acrescentou", "172.18.0.1:51234", "203.0.113.7", "203.0.113.7"},
		{"atras do Caddy ignora valores forjados antes do dele", "127.0.0.1:51234", "1.2.3.4, 203.0.113.7", "203.0.113.7"},
		{"conexao direta de IP publico ignora o cabecalho forjado", "198.51.100.20:40000", "127.0.0.1", "198.51.100.20"},
		{"sem cabecalho usa o IP da conexao", "172.18.0.1:51234", "", "172.18.0.1"},
		{"IPv6 local tambem e tratado como proxy", "[::1]:51234", "2001:db8::1", "2001:db8::1"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			requisicao := httptest.NewRequest("POST", "/login", nil)
			requisicao.RemoteAddr = caso.enderecoRemoto
			if caso.encaminhadoPara != "" {
				requisicao.Header.Set("X-Forwarded-For", caso.encaminhadoPara)
			}
			if obtido := EnderecoIPDoCliente(requisicao); obtido != caso.esperado {
				t.Errorf("EnderecoIPDoCliente = %q; esperado %q", obtido, caso.esperado)
			}
		})
	}
}
