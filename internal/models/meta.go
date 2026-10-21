package models

import "time"

// Meta e um objetivo financeiro da familia (viagem, carro, entrada de
// imovel...) acompanhado por aportes registrados.
type Meta struct {
	ID            int
	Nome          string
	ValorAlvo     float64
	DataAlvo      time.Time // primeiro dia do mes do prazo
	ValorGuardado float64   // soma dos aportes (retirada entra negativa)
}

// Falta e o que ainda precisa ser guardado. Retiradas alem do guardado nao
// aumentam a meta: o pior caso e faltar o valor-alvo inteiro.
func (meta Meta) Falta() float64 {
	return max(meta.ValorAlvo-max(meta.ValorGuardado, 0), 0)
}

func (meta Meta) PercentualAtingido() float64 {
	return meta.ValorGuardado / meta.ValorAlvo * 100
}

// LarguraDaBarra e o percentual atingido entre 0 e 100, para a barra nao vazar.
func (meta Meta) LarguraDaBarra() float64 {
	return min(max(meta.PercentualAtingido(), 0), 100)
}

// MesesRestantes conta os aportes mensais que ainda cabem ate o prazo,
// incluindo o mes atual e o mes do prazo: em outubro, prazo em dezembro = 3.
// Zero ou negativo significa prazo vencido.
func (meta Meta) MesesRestantes(hoje time.Time) int {
	return (meta.DataAlvo.Year()-hoje.Year())*12 + int(meta.DataAlvo.Month()-hoje.Month()) + 1
}

// AporteMensalSugerido divide o que falta pelos meses restantes. Meta
// concluida ou com prazo vencido nao tem sugestao.
func (meta Meta) AporteMensalSugerido(hoje time.Time) float64 {
	meses := meta.MesesRestantes(hoje)
	if meses <= 0 || meta.Falta() == 0 {
		return 0
	}
	return meta.Falta() / float64(meses)
}

// Situacao devolve "concluida", "vencida" ou "em_andamento".
func (meta Meta) Situacao(hoje time.Time) string {
	switch {
	case meta.Falta() == 0:
		return "concluida"
	case meta.MesesRestantes(hoje) <= 0:
		return "vencida"
	default:
		return "em_andamento"
	}
}
