package models

import (
	"math"
	"testing"
	"time"
)

// O aporte sugerido e o numero que o usuario segue todo mes: erro de um mes
// na contagem muda o valor de todas as parcelas.
func TestMeta_MesesRestantesEAporteSugerido(t *testing.T) {
	outubro := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	mes := func(ano int, mes time.Month) time.Time { return time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC) }

	casos := []struct {
		nome             string
		prazo            time.Time
		guardado         float64
		mesesEsperados   int
		aporteEsperado   float64
		situacaoEsperada string
	}{
		{"prazo no mes atual: um aporte so", mes(2026, 10), 400, 1, 600, "em_andamento"},
		{"prazo em dezembro: outubro, novembro e dezembro", mes(2026, 12), 400, 3, 200, "em_andamento"},
		{"virada de ano conta certo", mes(2027, 3), 0, 6, 1000.0 / 6, "em_andamento"},
		{"prazo no mes passado: vencida, sem sugestao", mes(2026, 9), 400, 0, 0, "vencida"},
		{"valor atingido: concluida, sem sugestao", mes(2027, 3), 1000, 6, 0, "concluida"},
		{"passou do valor tambem e concluida", mes(2027, 3), 1200, 6, 0, "concluida"},
		{"retiradas deixaram o guardado negativo: falta o alvo inteiro", mes(2026, 12), -200, 3, 1000.0 / 3, "em_andamento"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			meta := Meta{ValorAlvo: 1000, DataAlvo: caso.prazo, ValorGuardado: caso.guardado}
			if meses := meta.MesesRestantes(outubro); max(meses, 0) != caso.mesesEsperados {
				t.Errorf("MesesRestantes = %d; esperado %d", meses, caso.mesesEsperados)
			}
			if aporte := meta.AporteMensalSugerido(outubro); math.Abs(aporte-caso.aporteEsperado) > 0.005 {
				t.Errorf("AporteMensalSugerido = %.2f; esperado %.2f", aporte, caso.aporteEsperado)
			}
			if situacao := meta.Situacao(outubro); situacao != caso.situacaoEsperada {
				t.Errorf("Situacao = %q; esperado %q", situacao, caso.situacaoEsperada)
			}
		})
	}
}
