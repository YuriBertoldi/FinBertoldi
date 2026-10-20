//go:build integration

package store

import (
	"database/sql"
	"testing"
	"time"

	"fincontrol/internal/models"
)

// O gasto do orcamento soma despesas fixas e extrato. O risco e contar a mesma
// conta duas vezes (transacao ja convertida em despesa) ou somar o que nao e
// gasto da categoria no mes. Cada transacao abaixo testa uma dessas regras.
func TestOrcamentosDoMes_SomaDespesasFixasEExtratoSemContarDuasVezes(t *testing.T) {
	db := setupTestDB(t)
	familia, outraFamilia := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	for _, fid := range []int{familia, outraFamilia} {
		if err := CreateCategoria(db, fid, "Mercado", "basica", "#2ecc71"); err != nil {
			t.Fatalf("categoria: %v", err)
		}
	}
	if err := CreateCategoria(db, familia, "Lazer", "basica", "#9b59b6"); err != nil {
		t.Fatalf("categoria sem orcamento: %v", err)
	}
	mercadoID := idDaCategoria(t, db, familia, "Mercado")
	limite := 1000.0
	if err := DefinirOrcamentoMensalDaCategoria(db, familia, mercadoID, &limite); err != nil {
		t.Fatalf("definir orcamento: %v", err)
	}

	if err := CreateDespesa(db, familia, "Feira do mes", 300, "Mercado"); err != nil {
		t.Fatalf("despesa: %v", err)
	}
	transacao := func(fid int, fitID string, data time.Time, valor float64, tipo, categoria string) {
		t.Helper()
		if _, err := CreateTransacaoBancoBatch(db, fid, []models.TransacaoBanco{{
			Data: data, Descricao: fitID, Valor: valor, Tipo: tipo, Categoria: categoria, Origem: "ofx", FitID: fitID,
		}}); err != nil {
			t.Fatalf("transacao %s: %v", fitID, err)
		}
	}
	transacao(familia, "conta", outubro.AddDate(0, 0, 10), -200, "debito", "Mercado")              // conta
	transacao(familia, "mes-anterior", outubro.AddDate(0, 0, -3), -50, "debito", "Mercado")        // outro mes
	transacao(familia, "estorno", outubro.AddDate(0, 0, 11), 100, "credito", "Mercado")            // credito nao e gasto
	transacao(familia, "outra-categoria", outubro.AddDate(0, 0, 12), -40, "debito", "Lazer")       // outra categoria
	transacao(outraFamilia, "outra-familia", outubro.AddDate(0, 0, 13), -500, "debito", "Mercado") // outra familia
	transacao(familia, "ja-convertida", outubro.AddDate(0, 0, 14), -70, "debito", "Mercado")       // vira despesa
	transacao(familia, "ignorada", outubro.AddDate(0, 0, 15), -30, "debito", "Mercado")            // ignorada
	if err := ConverterTransacaoEmDespesa(db, familia, idDaTransacao(t, db, familia, "ja-convertida"), "Mercado"); err != nil {
		t.Fatalf("converter: %v", err)
	}
	if err := IgnorarTransacao(db, familia, idDaTransacao(t, db, familia, "ignorada")); err != nil {
		t.Fatalf("ignorar: %v", err)
	}

	orcamentos, err := OrcamentosDoMes(db, familia, outubro)
	if err != nil {
		t.Fatalf("OrcamentosDoMes: %v", err)
	}
	if len(orcamentos) != 1 {
		t.Fatalf("esperado so Mercado (Lazer nao tem orcamento); veio %+v", orcamentos)
	}
	mercado := orcamentos[0]
	// Despesas fixas: 300 da feira + 70 da transacao convertida (que agora e despesa).
	if mercado.GastoEmDespesasFixas != 370 {
		t.Errorf("despesas fixas = %.2f; esperado 370", mercado.GastoEmDespesasFixas)
	}
	// Extrato: so o debito de 200 do mes, da categoria, da familia, nem convertido nem ignorado.
	if mercado.GastoEmTransacoes != 200 {
		t.Errorf("extrato = %.2f; esperado 200", mercado.GastoEmTransacoes)
	}
	if mercado.Limite != 1000 {
		t.Errorf("limite = %.2f; esperado 1000", mercado.Limite)
	}

	// Remover o orcamento tira a categoria da lista.
	if err := DefinirOrcamentoMensalDaCategoria(db, familia, mercadoID, nil); err != nil {
		t.Fatalf("remover orcamento: %v", err)
	}
	if orcamentos, _ := OrcamentosDoMes(db, familia, outubro); len(orcamentos) != 0 {
		t.Errorf("sem orcamento a categoria nao deveria aparecer; veio %+v", orcamentos)
	}
}

func TestOrcamento_BancoRecusaLimiteZeroOuNegativo(t *testing.T) {
	db := setupTestDB(t)
	familia, _ := criarDuasFamilias(t, db)
	if err := CreateCategoria(db, familia, "Mercado", "basica", "#2ecc71"); err != nil {
		t.Fatalf("categoria: %v", err)
	}
	for _, limiteInvalido := range []float64{0, -10} {
		if err := DefinirOrcamentoMensalDaCategoria(db, familia, idDaCategoria(t, db, familia, "Mercado"), &limiteInvalido); err == nil {
			t.Errorf("limite %.2f foi aceito", limiteInvalido)
		}
	}
}

func idDaCategoria(t *testing.T, db *sql.DB, fid int, nome string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM categorias WHERE family_id=$1 AND nome=$2`, fid, nome).Scan(&id); err != nil {
		t.Fatalf("id da categoria %q: %v", nome, err)
	}
	return id
}

func idDaTransacao(t *testing.T, db *sql.DB, fid int, fitID string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM transacoes_banco WHERE family_id=$1 AND fit_id=$2`, fid, fitID).Scan(&id); err != nil {
		t.Fatalf("id da transacao %q: %v", fitID, err)
	}
	return id
}
