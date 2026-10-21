package contas

import (
	"testing"

	"fincontrol/internal/models"
)

// Erro aqui vira mensagem repetida (spam) ou alerta perdido.
func TestAlertasNovos_CadaFaixaSaiUmaVezPorCategoria(t *testing.T) {
	const mercado = 7
	orcamentoComGasto := func(gasto float64) models.OrcamentoDaCategoria {
		return models.OrcamentoDaCategoria{CategoriaID: mercado, Categoria: "Mercado", Limite: 1000, GastoEmTransacoes: gasto}
	}
	jaEnviou := func(faixas ...string) map[int]map[string]bool {
		enviados := map[int]map[string]bool{mercado: {}}
		for _, faixa := range faixas {
			enviados[mercado][faixa] = true
		}
		return enviados
	}

	casos := []struct {
		nome          string
		gasto         float64
		jaEnviados    map[int]map[string]bool
		faixaEsperada string // vazio = nenhum alerta
	}{
		{"abaixo de 80% nao avisa", 799, nil, ""},
		{"chegou a 80% avisa alerta", 800, nil, "alerta"},
		{"alerta ja enviado nao repete", 900, jaEnviou("alerta"), ""},
		{"passou de 100% avisa estourado", 1001, jaEnviou("alerta"), "estourado"},
		{"pulou direto para acima de 100% avisa so estourado", 1500, nil, "estourado"},
		{"estourado ja enviado nao repete", 1500, jaEnviou("alerta", "estourado"), ""},
		{"limite aumentado depois de estourar nao volta a avisar 80%", 850, jaEnviou("estourado"), ""},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			novos := alertasNovos([]models.OrcamentoDaCategoria{orcamentoComGasto(caso.gasto)}, caso.jaEnviados)
			switch {
			case caso.faixaEsperada == "" && len(novos) != 0:
				t.Errorf("nao esperava alerta; veio %q", novos[0].Faixa)
			case caso.faixaEsperada != "" && len(novos) != 1:
				t.Errorf("esperava um alerta %q; vieram %d", caso.faixaEsperada, len(novos))
			case caso.faixaEsperada != "" && novos[0].Faixa != caso.faixaEsperada:
				t.Errorf("faixa = %q; esperado %q", novos[0].Faixa, caso.faixaEsperada)
			}
		})
	}
}
