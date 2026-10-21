//go:build integration

package store

import (
	"testing"
	"time"
)

func TestMetas_AportesERetiradasSomamNoGuardado(t *testing.T) {
	db := setupTestDB(t)
	familia, outraFamilia := criarDuasFamilias(t, db)
	outubro := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

	// O prazo e gravado como o primeiro dia do mes, qualquer que seja o dia informado.
	if err := CriarMeta(db, familia, "Viagem", 6000, time.Date(2027, 6, 20, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("criar meta: %v", err)
	}
	metas, _ := Metas(db, familia)
	if len(metas) != 1 {
		t.Fatalf("metas = %+v", metas)
	}
	viagem := metas[0]
	if !viagem.DataAlvo.Equal(time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("prazo = %v; esperado 01/06/2027", viagem.DataAlvo)
	}

	for _, valor := range []float64{1000, 500, -300} {
		if err := RegistrarAporteNaMeta(db, familia, viagem.ID, valor, outubro); err != nil {
			t.Fatalf("aporte %.2f: %v", valor, err)
		}
	}
	// Outra familia tentando aportar (ou retirar) na meta alheia: ignorado.
	RegistrarAporteNaMeta(db, outraFamilia, viagem.ID, -1000, outubro)

	metas, _ = Metas(db, familia)
	if metas[0].ValorGuardado != 1200 {
		t.Errorf("guardado = %.2f; esperado 1200 (1000 + 500 - 300)", metas[0].ValorGuardado)
	}
	if metas, _ := Metas(db, outraFamilia); len(metas) != 0 {
		t.Errorf("a meta apareceu para outra familia: %+v", metas)
	}

	// Excluir a meta leva os aportes junto.
	if err := ExcluirMeta(db, familia, viagem.ID); err != nil {
		t.Fatalf("excluir: %v", err)
	}
	var aportesRestantes int
	db.QueryRow(`SELECT count(*) FROM metas_aportes`).Scan(&aportesRestantes)
	if aportesRestantes != 0 {
		t.Errorf("sobraram %d aportes de meta excluida", aportesRestantes)
	}
}

func TestMetas_BancoRecusaValoresInvalidos(t *testing.T) {
	db := setupTestDB(t)
	familia, _ := criarDuasFamilias(t, db)
	prazo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := CriarMeta(db, familia, "Sem valor", 0, prazo); err == nil {
		t.Error("meta com valor-alvo zero foi aceita")
	}
	if err := CriarMeta(db, familia, "   ", 100, prazo); err == nil {
		t.Error("meta sem nome foi aceita")
	}
	if err := CriarMeta(db, familia, "Carro", 100, prazo); err != nil {
		t.Fatalf("criar meta: %v", err)
	}
	metas, _ := Metas(db, familia)
	if err := RegistrarAporteNaMeta(db, familia, metas[0].ID, 0, prazo); err == nil {
		t.Error("aporte de valor zero foi aceito")
	}
}
