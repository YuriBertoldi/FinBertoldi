package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LimiteDeTentativasDeLogin bloqueia temporariamente um e-mail ou um IP depois
// de muitas senhas erradas seguidas. Fica em memoria: o app roda em uma unica
// instancia, e zerar os contadores quando o app reinicia e aceitavel.
//
// O bloqueio vale para qualquer e-mail digitado, exista ele ou nao, para que a
// resposta nao revele quais e-mails tem conta.
type LimiteDeTentativasDeLogin struct {
	maximoDeFalhasPorEmail int
	maximoDeFalhasPorIP    int
	janelaDeContagem       time.Duration
	duracaoDoBloqueio      time.Duration
	agora                  func() time.Time

	mutex               sync.Mutex
	falhasPorChave      map[string]*falhasRecentes
	ultimaLimpezaDoMapa time.Time
}

type falhasRecentes struct {
	quantidade     int
	inicioDaJanela time.Time
	bloqueadoAte   time.Time
}

const intervaloEntreLimpezasDoMapa = time.Minute

// NovoLimiteDeTentativasDeLogin usa os valores de producao: 5 erros por e-mail
// ou 20 por IP em 15 minutos bloqueiam por 15 minutos.
func NovoLimiteDeTentativasDeLogin() *LimiteDeTentativasDeLogin {
	return &LimiteDeTentativasDeLogin{
		maximoDeFalhasPorEmail: 5,
		maximoDeFalhasPorIP:    20,
		janelaDeContagem:       15 * time.Minute,
		duracaoDoBloqueio:      15 * time.Minute,
		agora:                  time.Now,
		falhasPorChave:         make(map[string]*falhasRecentes),
	}
}

// EstaBloqueado informa se o e-mail ou o IP esta impedido de tentar login agora.
func (limite *LimiteDeTentativasDeLogin) EstaBloqueado(email, enderecoIP string) bool {
	limite.mutex.Lock()
	defer limite.mutex.Unlock()

	agora := limite.agora()
	for _, chave := range chavesDeContagem(email, enderecoIP) {
		if falhas, existe := limite.falhasPorChave[chave]; existe && agora.Before(falhas.bloqueadoAte) {
			return true
		}
	}
	return false
}

// RegistrarFalha conta uma senha errada para o e-mail e para o IP, e bloqueia
// quem passar do limite dentro da janela.
func (limite *LimiteDeTentativasDeLogin) RegistrarFalha(email, enderecoIP string) {
	limite.mutex.Lock()
	defer limite.mutex.Unlock()

	agora := limite.agora()
	limite.removerRegistrosExpirados(agora)

	chaves := chavesDeContagem(email, enderecoIP)
	limite.contarFalha(chaves[0], limite.maximoDeFalhasPorEmail, agora)
	limite.contarFalha(chaves[1], limite.maximoDeFalhasPorIP, agora)
}

// RegistrarSucesso zera a contagem do e-mail. A do IP continua, para que um
// atacante com uma conta propria nao consiga zerar o proprio contador.
func (limite *LimiteDeTentativasDeLogin) RegistrarSucesso(email string) {
	limite.mutex.Lock()
	defer limite.mutex.Unlock()

	delete(limite.falhasPorChave, chaveDoEmail(email))
}

func (limite *LimiteDeTentativasDeLogin) contarFalha(chave string, maximoDeFalhas int, agora time.Time) {
	falhas, existe := limite.falhasPorChave[chave]
	if !existe || agora.Sub(falhas.inicioDaJanela) > limite.janelaDeContagem {
		falhas = &falhasRecentes{inicioDaJanela: agora}
		limite.falhasPorChave[chave] = falhas
	}
	falhas.quantidade++
	if falhas.quantidade >= maximoDeFalhas {
		falhas.bloqueadoAte = agora.Add(limite.duracaoDoBloqueio)
	}
}

// removerRegistrosExpirados impede que o mapa cresca sem limite com IPs e
// e-mails que nunca mais voltam.
func (limite *LimiteDeTentativasDeLogin) removerRegistrosExpirados(agora time.Time) {
	if agora.Sub(limite.ultimaLimpezaDoMapa) < intervaloEntreLimpezasDoMapa {
		return
	}
	limite.ultimaLimpezaDoMapa = agora
	for chave, falhas := range limite.falhasPorChave {
		janelaExpirou := agora.Sub(falhas.inicioDaJanela) > limite.janelaDeContagem
		bloqueioExpirou := !agora.Before(falhas.bloqueadoAte)
		if janelaExpirou && bloqueioExpirou {
			delete(limite.falhasPorChave, chave)
		}
	}
}

func chavesDeContagem(email, enderecoIP string) [2]string {
	return [2]string{chaveDoEmail(email), "ip:" + enderecoIP}
}

func chaveDoEmail(email string) string {
	return "email:" + strings.ToLower(strings.TrimSpace(email))
}

// EnderecoIPDoCliente devolve o IP de quem fez a requisicao. Atras do Caddy a
// conexao chega de um endereco local ou privado e o IP real e o ultimo item do
// X-Forwarded-For (o que o proprio Caddy acrescentou). Conexao vinda de IP
// publico e direta: o cabecalho e ignorado, porque o cliente poderia forja-lo.
func EnderecoIPDoCliente(r *http.Request) string {
	enderecoDaConexao, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		enderecoDaConexao = r.RemoteAddr
	}

	ipDaConexao := net.ParseIP(enderecoDaConexao)
	conexaoVemDoProxy := ipDaConexao != nil && (ipDaConexao.IsLoopback() || ipDaConexao.IsPrivate())
	if !conexaoVemDoProxy {
		return enderecoDaConexao
	}

	encaminhadoPara := r.Header.Get("X-Forwarded-For")
	if encaminhadoPara == "" {
		return enderecoDaConexao
	}
	enderecos := strings.Split(encaminhadoPara, ",")
	return strings.TrimSpace(enderecos[len(enderecos)-1])
}
