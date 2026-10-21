//go:build integration

package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"fincontrol/internal/models"
)

// Transacao de debito como chega do OFX/Pluggy: valor negativo.
func criarDebito(t *testing.T, db *sql.DB, fid int, fitID string, data time.Time, valor float64) int {
	t.Helper()
	if _, err := CreateTransacaoBancoBatch(db, fid, []models.TransacaoBanco{{
		Data: data, Descricao: fitID, Valor: -valor, Tipo: "debito", Origem: "ofx", FitID: fitID,
	}}); err != nil {
		t.Fatalf("transacao %s: %v", fitID, err)
	}
	return idDaTransacao(t, db, fid, fitID)
}

func colunasDoVinculo(t *testing.T, db *sql.DB, transacaoID int) (despesaID, receitaID, parcelamentoID sql.NullInt64, status string) {
	t.Helper()
	if err := db.QueryRow(`SELECT despesa_id, receita_id, parcelamento_id, status FROM transacoes_banco WHERE id=$1`, transacaoID).
		Scan(&despesaID, &receitaID, &parcelamentoID, &status); err != nil {
		t.Fatalf("ler vinculo: %v", err)
	}
	return
}

// Antes, o vinculo com parcelamento gravava o id do parcelamento em
// despesa_id, coluna que aponta para despesas_fixas.
func TestVincularTransacaoParcelamento_GravaNaColunaDoParcelamento(t *testing.T) {
	db := setupTestDB(t)
	familia, _ := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := CreateParcelamento(db, familia, "Geladeira", "Cartao", 300, 1, 10, outubro, false, 0, 0); err != nil {
		t.Fatalf("parcelamento: %v", err)
	}
	var geladeiraID int
	db.QueryRow(`SELECT id FROM parcelamentos WHERE family_id=$1`, familia).Scan(&geladeiraID)
	transacaoID := criarDebito(t, db, familia, "parcela-geladeira", outubro.AddDate(0, 0, 10), 300)

	if err := VincularTransacaoParcelamento(db, familia, transacaoID, geladeiraID); err != nil {
		t.Fatalf("vincular: %v", err)
	}
	despesaID, _, parcelamentoID, status := colunasDoVinculo(t, db, transacaoID)
	if despesaID.Valid || !parcelamentoID.Valid || int(parcelamentoID.Int64) != geladeiraID || status != "convertida" {
		t.Errorf("despesa_id=%v parcelamento_id=%v status=%s; esperado so parcelamento_id=%d e convertida", despesaID, parcelamentoID, status, geladeiraID)
	}

	if err := DesvincularTransacao(db, familia, transacaoID); err != nil {
		t.Fatalf("desvincular: %v", err)
	}
	if _, _, parcelamentoID, status := colunasDoVinculo(t, db, transacaoID); parcelamentoID.Valid || status != "pendente" {
		t.Errorf("depois de desvincular: parcelamento_id=%v status=%s", parcelamentoID, status)
	}
}

// Antes, qualquer id era aceito: dava para apontar a transacao para despesa,
// receita ou parcelamento de outra familia.
func TestVincularTransacao_RecusaRegistroDeOutraFamilia(t *testing.T) {
	db := setupTestDB(t)
	familia, outraFamilia := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	CreateDespesa(db, outraFamilia, "Aluguel", 2000, "Moradia")
	CreateReceita(db, outraFamilia, "Salario", 9000, outubro, "salario", true)
	CreateParcelamento(db, outraFamilia, "Moto", "Banco", 500, 1, 10, outubro, false, 0, 0)
	var despesaAlheia, receitaAlheia, parcelamentoAlheio int
	db.QueryRow(`SELECT id FROM despesas_fixas WHERE family_id=$1`, outraFamilia).Scan(&despesaAlheia)
	db.QueryRow(`SELECT id FROM receitas WHERE family_id=$1`, outraFamilia).Scan(&receitaAlheia)
	db.QueryRow(`SELECT id FROM parcelamentos WHERE family_id=$1`, outraFamilia).Scan(&parcelamentoAlheio)
	transacaoID := criarDebito(t, db, familia, "debito", outubro, 100)

	tentativas := map[string]error{
		"despesa":      VincularTransacaoDespesa(db, familia, transacaoID, despesaAlheia),
		"receita":      VincularTransacaoReceita(db, familia, transacaoID, receitaAlheia),
		"parcelamento": VincularTransacaoParcelamento(db, familia, transacaoID, parcelamentoAlheio),
	}
	for destino, err := range tentativas {
		if !errors.Is(err, ErrVinculoNaoEncontrado) {
			t.Errorf("vincular a %s de outra familia: erro %v; esperado ErrVinculoNaoEncontrado", destino, err)
		}
	}
	despesaID, receitaID, parcelamentoID, status := colunasDoVinculo(t, db, transacaoID)
	if despesaID.Valid || receitaID.Valid || parcelamentoID.Valid || status != "pendente" {
		t.Errorf("a transacao mudou: despesa=%v receita=%v parcelamento=%v status=%s", despesaID, receitaID, parcelamentoID, status)
	}
}

// Debito chega negativo; despesa e parcela sao cadastradas positivas. As
// buscas comparavam o valor com sinal e nunca achavam nada; a de
// parcelamentos ainda comparava a data com o texto "2026-10".
func TestSugestoesEVinculoAutomatico_EncontramDebitosNegativos(t *testing.T) {
	db := setupTestDB(t)
	familia, _ := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := CreateDespesa(db, familia, "Internet", 120, "Moradia"); err != nil {
		t.Fatalf("despesa: %v", err)
	}
	if err := CreateParcelamento(db, familia, "Geladeira", "Cartao", 300, 1, 10, outubro, false, 0, 0); err != nil {
		t.Fatalf("parcelamento: %v", err)
	}
	var geladeiraID int
	db.QueryRow(`SELECT id FROM parcelamentos WHERE family_id=$1`, familia).Scan(&geladeiraID)
	// A parcela do mes em aberto, como o dashboard cria ao abrir o mes.
	if _, err := db.Exec(`INSERT INTO parcelamentos_mes (parcelamento_id, mes, pago) VALUES ($1, $2, false)`, geladeiraID, outubro); err != nil {
		t.Fatalf("parcela do mes: %v", err)
	}

	sugestoesDeDespesa, err := BuscarMatchesDespesas(db, familia, -120, "INTERNET")
	if err != nil || len(sugestoesDeDespesa) != 1 || sugestoesDeDespesa[0].Score != 100 {
		t.Errorf("sugestoes de despesa para debito de -120: %+v (erro %v); esperado Internet com 100%%", sugestoesDeDespesa, err)
	}
	sugestoesDeParcela, err := BuscarMatchesParcelamentos(db, familia, -300, outubro.AddDate(0, 0, 12))
	if err != nil || len(sugestoesDeParcela) != 1 || sugestoesDeParcela[0].Score != 100 {
		t.Errorf("sugestoes de parcelamento para debito de -300: %+v (erro %v); esperado Geladeira com 100%%", sugestoesDeParcela, err)
	}

	internet := criarDebito(t, db, familia, "internet", outubro.AddDate(0, 0, 5), 120)
	parcela := criarDebito(t, db, familia, "parcela", outubro.AddDate(0, 0, 12), 300)
	if vinculadas, err := AutoMatchTransacoes(db, familia); err != nil || vinculadas != 2 {
		t.Errorf("vinculo automatico: %d (erro %v); esperado 2", vinculadas, err)
	}
	if despesaID, _, _, _ := colunasDoVinculo(t, db, internet); !despesaID.Valid {
		t.Error("o debito da internet nao foi vinculado a despesa")
	}
	if despesaID, _, parcelamentoID, _ := colunasDoVinculo(t, db, parcela); despesaID.Valid || int(parcelamentoID.Int64) != geladeiraID {
		t.Errorf("parcela vinculada errado: despesa_id=%v parcelamento_id=%v", despesaID, parcelamentoID)
	}
}
