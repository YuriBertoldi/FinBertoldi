//go:build integration

package store

import (
	"database/sql"
	"testing"
	"time"

	"fincontrol/internal/models"
)

// Cada transacao abaixo testa o que a aplicacao das regras pode e nao pode
// mudar. No fim, confere que o gasto categorizado passa a contar no orcamento,
// que e o motivo de as regras existirem.
func TestAplicarRegrasDeCategorizacao(t *testing.T) {
	db := setupTestDB(t)
	familia, outraFamilia := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	for _, fid := range []int{familia, outraFamilia} {
		for _, nome := range []string{"Alimentação", "Lazer"} {
			if err := CreateCategoria(db, fid, nome, "basica", "#2ecc71"); err != nil {
				t.Fatalf("categoria: %v", err)
			}
		}
	}
	alimentacaoID := idDaCategoria(t, db, familia, "Alimentação")

	transacao := func(fid int, fitID, categoria string) {
		t.Helper()
		if _, err := CreateTransacaoBancoBatch(db, fid, []models.TransacaoBanco{{
			Data: outubro.AddDate(0, 0, 5), Descricao: "MERCADO " + fitID, Valor: -100, Tipo: "debito",
			Categoria: categoria, Origem: "pluggy", FitID: fitID,
		}}); err != nil {
			t.Fatalf("transacao %s: %v", fitID, err)
		}
	}
	transacao(familia, "do-pluggy", "Groceries")          // deve virar Alimentação
	transacao(familia, "ja-da-familia", "Lazer")          // ja e categoria da familia: nao muda
	transacao(familia, "escolhida-a-mao", "Groceries")    // categorizada a mao: nao muda
	transacao(familia, "ignorada", "Groceries")           // ignorada: nao muda
	transacao(outraFamilia, "outra-familia", "Groceries") // regra e de outra familia: nao muda
	if err := CategorizarTransacao(db, familia, idDaTransacao(t, db, familia, "escolhida-a-mao"), "Lazer"); err != nil {
		t.Fatalf("categorizar a mao: %v", err)
	}
	if err := IgnorarTransacao(db, familia, idDaTransacao(t, db, familia, "ignorada")); err != nil {
		t.Fatalf("ignorar: %v", err)
	}

	if err := CriarRegraDeCategorizacao(db, familia, "categoria_do_banco", "groceries", alimentacaoID); err != nil {
		t.Fatalf("criar regra: %v", err)
	}
	// Regra que casaria com a descricao de todas, para provar que
	// "ja-da-familia" fica intacta por ja ter categoria da familia.
	if err := CriarRegraDeCategorizacao(db, familia, "descricao", "MERCADO", alimentacaoID); err != nil {
		t.Fatalf("criar regra: %v", err)
	}

	categorizadas, err := AplicarRegrasDeCategorizacao(db, familia)
	if err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if categorizadas != 1 {
		t.Errorf("categorizadas = %d; esperado 1 (so a do Pluggy)", categorizadas)
	}
	esperado := map[string]string{
		"do-pluggy": "Alimentação", "ja-da-familia": "Lazer", "escolhida-a-mao": "Lazer", "ignorada": "Groceries",
	}
	for fitID, categoriaEsperada := range esperado {
		if obtida := categoriaDaTransacao(t, db, familia, fitID); obtida != categoriaEsperada {
			t.Errorf("%s: categoria = %q; esperado %q", fitID, obtida, categoriaEsperada)
		}
	}
	if obtida := categoriaDaTransacao(t, db, outraFamilia, "outra-familia"); obtida != "Groceries" {
		t.Errorf("transacao de outra familia mudou para %q", obtida)
	}
	var categoriaDoBanco string
	db.QueryRow(`SELECT categoria_do_banco FROM transacoes_banco WHERE fit_id='do-pluggy'`).Scan(&categoriaDoBanco)
	if categoriaDoBanco != "Groceries" {
		t.Errorf("categoria do banco = %q; a original precisa ser guardada", categoriaDoBanco)
	}

	if categorizadas, _ := AplicarRegrasDeCategorizacao(db, familia); categorizadas != 0 {
		t.Errorf("segunda aplicacao mudou %d transacoes; precisa ser idempotente", categorizadas)
	}

	// O motivo de tudo: o gasto categorizado entra no orcamento.
	limite := 1000.0
	if err := DefinirOrcamentoMensalDaCategoria(db, familia, alimentacaoID, &limite); err != nil {
		t.Fatalf("orcamento: %v", err)
	}
	orcamentos, _ := OrcamentosDoMes(db, familia, outubro)
	if len(orcamentos) != 1 || orcamentos[0].GastoEmTransacoes != 100 {
		t.Errorf("orcamento de Alimentação = %+v; esperado extrato de 100", orcamentos)
	}
}

func TestCriarRegraDeCategorizacao_RecusaCategoriaDeOutraFamilia(t *testing.T) {
	db := setupTestDB(t)
	familia, outraFamilia := criarDuasFamilias(t, db)
	if err := CreateCategoria(db, familia, "Alimentação", "basica", "#2ecc71"); err != nil {
		t.Fatalf("categoria: %v", err)
	}
	categoriaDaFamilia := idDaCategoria(t, db, familia, "Alimentação")

	CriarRegraDeCategorizacao(db, outraFamilia, "descricao", "MERCADO", categoriaDaFamilia)

	if regras, _ := RegrasDeCategorizacao(db, outraFamilia); len(regras) != 0 {
		t.Errorf("outra familia criou regra apontando para categoria alheia: %+v", regras)
	}
}

func categoriaDaTransacao(t *testing.T, db *sql.DB, fid int, fitID string) string {
	t.Helper()
	var categoria string
	if err := db.QueryRow(`SELECT categoria FROM transacoes_banco WHERE family_id=$1 AND fit_id=$2`, fid, fitID).Scan(&categoria); err != nil {
		t.Fatalf("categoria de %s: %v", fitID, err)
	}
	return categoria
}
