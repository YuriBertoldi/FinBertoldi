//go:build integration

package store

import (
	"testing"
	"time"
)

// Parcelamento cadastrado na 3a de 5 parcelas, a partir de outubro: as
// parcelas sao outubro, novembro e dezembro. A regra antiga do historico
// ignorava a parcela atual e contava de outubro ate fevereiro (5 meses).
// O historico (graficos do dashboard e relatorio) precisa bater com o
// dashboard do mes, que considera a parcela atual.
func TestHistoricoMeses_ParcelasConsideramAParcelaAtual(t *testing.T) {
	db := setupTestDB(t)
	familia, _ := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	if err := CreateDespesa(db, familia, "Aluguel", 2000, "Moradia"); err != nil {
		t.Fatalf("despesa: %v", err)
	}
	if err := CreateParcelamento(db, familia, "Geladeira", "Cartao", 300, 3, 5, outubro, false, 0, 0); err != nil {
		t.Fatalf("parcelamento: %v", err)
	}

	fevereiro := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	historico, err := GetHistoricoMeses(db, familia, fevereiro, 6) // Set/26 a Fev/27
	if err != nil {
		t.Fatalf("historico: %v", err)
	}
	despesasEsperadas := map[string]float64{
		"2026-09": 2000, "2026-10": 2300, "2026-11": 2300, "2026-12": 2300, "2027-01": 2000, "2027-02": 2000,
	}
	for _, mes := range historico {
		if mes.Despesas != despesasEsperadas[mes.Mes] {
			t.Errorf("%s: despesas %.2f; esperado %.2f", mes.MesLabel, mes.Despesas, despesasEsperadas[mes.Mes])
		}

		// E o mesmo numero que o dashboard do mes soma.
		primeiroDiaDoMesDoHistorico, _ := time.Parse("2006-01", mes.Mes)
		vigentes, _ := GetParcelamentosVigentesNoMes(db, familia, primeiroDiaDoMesDoHistorico)
		var parcelasDoDashboard float64
		for _, parcelamento := range vigentes {
			parcelasDoDashboard += parcelamento.ValorParcela
		}
		if mes.Despesas-2000 != parcelasDoDashboard {
			t.Errorf("%s: historico tem %.2f de parcelas; o dashboard do mes soma %.2f", mes.MesLabel, mes.Despesas-2000, parcelasDoDashboard)
		}
	}
}
