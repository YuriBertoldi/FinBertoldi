package models

import (
	"fmt"
	"time"
)

// MesProjetado e um mes do fluxo de caixa projetado. Os campos calculados
// (Despesas, Sobra, SaldoAcumulado) sao preenchidos para irem prontos ao
// grafico, que recebe o JSON da struct.
type MesProjetado struct {
	Mes            time.Time
	Rotulo         string // "Out/26"
	Receitas       float64
	DespesasFixas  float64
	Parcelas       float64
	Despesas       float64
	Sobra          float64
	SaldoAcumulado float64 // soma das sobras desde o primeiro mes projetado
}

var abreviacoesDosMeses = [...]string{"", "Jan", "Fev", "Mar", "Abr", "Mai", "Jun", "Jul", "Ago", "Set", "Out", "Nov", "Dez"}

// MontarProjecao calcula despesas, sobra e saldo acumulado de cada mes, na ordem recebida.
func MontarProjecao(meses []MesProjetado) []MesProjetado {
	saldoAcumulado := 0.0
	for i := range meses {
		mes := &meses[i]
		mes.Rotulo = fmt.Sprintf("%s/%02d", abreviacoesDosMeses[mes.Mes.Month()], mes.Mes.Year()%100)
		mes.Despesas = mes.DespesasFixas + mes.Parcelas
		mes.Sobra = mes.Receitas - mes.Despesas
		saldoAcumulado += mes.Sobra
		mes.SaldoAcumulado = saldoAcumulado
	}
	return meses
}

// PrimeiroMesNoVermelho devolve o primeiro mes em que as despesas passam as
// receitas, ou nil se nenhum passar.
func PrimeiroMesNoVermelho(meses []MesProjetado) *MesProjetado {
	for i := range meses {
		if meses[i].Sobra < 0 {
			return &meses[i]
		}
	}
	return nil
}

// UltimoMesProjetado devolve o ultimo mes da projecao do dashboard, ou nil se
// ela estiver vazia (o template nao consegue calcular o ultimo indice).
func (dados DashboardData) UltimoMesProjetado() *MesProjetado {
	if len(dados.Projecao) == 0 {
		return nil
	}
	return &dados.Projecao[len(dados.Projecao)-1]
}
