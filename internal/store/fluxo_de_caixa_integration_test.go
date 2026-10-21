//go:build integration

package store

import (
	"math"
	"testing"
	"time"
)

// Cada lancamento abaixo cobre uma regra da projecao. O parcelamento comeca na
// 3a de 5 parcelas: pela regra do dashboard (que soma a parcela atual) ele
// termina em dezembro; pela regra antiga do historico, que ignora a parcela
// atual, iria ate fevereiro. A projecao precisa seguir o dashboard.
func TestProjecaoDoFluxoDeCaixa_RegrasDoMes(t *testing.T) {
	db := setupTestDB(t)
	familia, outraFamilia := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	falharSeErro := func(descricao string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", descricao, err)
		}
	}
	falharSeErro("salario", CreateReceita(db, familia, "Salario", 5000, outubro, "salario", true))
	falharSeErro("decimo terceiro", CreateReceita(db, familia, "13o", 1000, outubro.AddDate(0, 2, 5), "extra", false))
	falharSeErro("bonus do mes passado", CreateReceita(db, familia, "Bonus", 700, outubro.AddDate(0, -1, 0), "extra", false))
	falharSeErro("aluguel", CreateDespesa(db, familia, "Aluguel", 2000, "Moradia"))
	falharSeErro("despesa desativada", CreateDespesa(db, familia, "Academia", 999, "Lazer"))
	var academiaID int
	db.QueryRow(`SELECT id FROM despesas_fixas WHERE nome='Academia'`).Scan(&academiaID)
	falharSeErro("desativar", ToggleDespesaAtiva(db, familia, academiaID))
	falharSeErro("geladeira", CreateParcelamento(db, familia, "Geladeira", "Cartao", 300, 3, 5, outubro, false, 0, 0))
	falharSeErro("receita de outra familia", CreateReceita(db, outraFamilia, "Salario", 9000, outubro, "salario", true))
	falharSeErro("despesa de outra familia", CreateDespesa(db, outraFamilia, "Aluguel", 4000, "Moradia"))

	meses, err := ProjecaoDoFluxoDeCaixa(db, familia, outubro.AddDate(0, 0, 14), 12)
	if err != nil {
		t.Fatalf("projecao: %v", err)
	}
	if len(meses) != 12 || meses[0].Rotulo != "Out/26" || meses[11].Rotulo != "Set/27" {
		t.Fatalf("meses = %d (%s a %s); esperado 12, de Out/26 a Set/27", len(meses), meses[0].Rotulo, meses[len(meses)-1].Rotulo)
	}

	esperados := []struct {
		rotulo             string
		receitas, parcelas float64
	}{
		{"Out/26", 5000, 300}, // 3a parcela
		{"Nov/26", 5000, 300}, // 4a
		{"Dez/26", 6000, 300}, // 5a e ultima; 13o so neste mes
		{"Jan/27", 5000, 0},   // parcelamento terminou
	}
	for i, esperado := range esperados {
		mes := meses[i]
		if mes.Rotulo != esperado.rotulo || mes.Receitas != esperado.receitas || mes.Parcelas != esperado.parcelas || mes.DespesasFixas != 2000 {
			t.Errorf("%s: receitas %.2f, fixas %.2f, parcelas %.2f; esperado %s com receitas %.2f, fixas 2000, parcelas %.2f",
				mes.Rotulo, mes.Receitas, mes.DespesasFixas, mes.Parcelas, esperado.rotulo, esperado.receitas, esperado.parcelas)
		}
	}

	// Sobra: 2700 x 3 meses com parcela (+1000 do 13o) e 3000 x 9 meses sem.
	saldoEsperado := 2700*3 + 1000 + 3000*9.0
	if ultimo := meses[11]; math.Abs(ultimo.SaldoAcumulado-saldoEsperado) > 0.001 {
		t.Errorf("saldo acumulado em 12 meses = %.2f; esperado %.2f", ultimo.SaldoAcumulado, saldoEsperado)
	}
}

// Parcela antecipada ja foi paga: nao pode sair de novo no mes original.
func TestProjecaoDoFluxoDeCaixa_ParcelaAntecipadaFicaDeFora(t *testing.T) {
	db := setupTestDB(t)
	familia, _ := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	if err := CreateParcelamento(db, familia, "Moto", "Banco", 500, 1, 6, outubro, true, 2800, 1.5); err != nil {
		t.Fatalf("parcelamento: %v", err)
	}
	var motoID int
	db.QueryRow(`SELECT id FROM parcelamentos WHERE family_id=$1`, familia).Scan(&motoID)
	if _, err := AnteciparParcelas(db, familia, motoID, 2); err != nil {
		t.Fatalf("antecipar: %v", err)
	}
	mesesAntecipados := map[string]bool{}
	linhas, _ := db.Query(`SELECT to_char(mes, 'YYYY-MM') FROM parcelamentos_mes WHERE parcelamento_id=$1 AND antecipada`, motoID)
	for linhas.Next() {
		var mes string
		linhas.Scan(&mes)
		mesesAntecipados[mes] = true
	}
	linhas.Close()
	if len(mesesAntecipados) != 2 {
		t.Fatalf("meses antecipados = %v; esperado 2", mesesAntecipados)
	}

	meses, err := ProjecaoDoFluxoDeCaixa(db, familia, outubro, 12)
	if err != nil {
		t.Fatalf("projecao: %v", err)
	}
	var totalDeParcelas float64
	for _, mes := range meses {
		if mesesAntecipados[mes.Mes.Format("2006-01")] && mes.Parcelas != 0 {
			t.Errorf("%s: parcela antecipada contada de novo (%.2f)", mes.Rotulo, mes.Parcelas)
		}
		totalDeParcelas += mes.Parcelas
	}
	if totalDeParcelas != 500*4 {
		t.Errorf("parcelas no periodo = %.2f; esperado 2000 (6 parcelas, 2 antecipadas)", totalDeParcelas)
	}
}
