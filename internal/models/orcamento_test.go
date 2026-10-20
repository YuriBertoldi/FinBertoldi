package models

import "testing"

// As faixas decidem a cor da barra e, no proximo passo, quando o alerta e
// enviado. Os casos sao as bordas exatas de cada faixa.
func TestOrcamentoDaCategoria_SituacaoNasBordasDasFaixas(t *testing.T) {
	casos := []struct {
		nome              string
		despesas, extrato float64
		situacaoEsperada  string
		larguraEsperada   float64
	}{
		{"abaixo de 80%", 300, 499.99, "dentro", 79.999},
		{"exatamente 80% ja e alerta", 300, 500, "alerta", 80},
		{"exatamente 100% ainda e alerta", 600, 400, "alerta", 100},
		{"acima de 100% e estourado, barra para em 100", 600, 900, "estourado", 100},
		{"sem gasto", 0, 0, "dentro", 0},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			orcamento := OrcamentoDaCategoria{Limite: 1000, GastoEmDespesasFixas: caso.despesas, GastoEmTransacoes: caso.extrato}
			if situacao := orcamento.Situacao(); situacao != caso.situacaoEsperada {
				t.Errorf("Situacao() = %q; esperado %q (usado %.3f%%)", situacao, caso.situacaoEsperada, orcamento.PercentualUsado())
			}
			if largura := orcamento.LarguraDaBarra(); largura < caso.larguraEsperada-0.01 || largura > caso.larguraEsperada+0.01 {
				t.Errorf("LarguraDaBarra() = %.3f; esperado %.3f", largura, caso.larguraEsperada)
			}
		})
	}
}
