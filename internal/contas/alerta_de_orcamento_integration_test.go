//go:build integration

package contas_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"fincontrol/internal/bancodeteste"
	"fincontrol/internal/contas"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
)

// Fluxo do agendador contra o banco: so quem pediu o alerta, da familia dona
// do orcamento, recebe; nada antes das 8h; e rodar de hora em hora nao
// repete o aviso. Os enderecos sao .invalid e o CI nao tem SMTP nem bot:
// o envio falha, mas o registro e gravado do mesmo jeito (e isso que evita
// repetir o erro a cada hora).
func TestEnviarAlertasDeOrcamento_SoParaQuemPediuESemRepetir(t *testing.T) {
	db := bancodeteste.AbrirBancoIsolado(t)
	familia := bancodeteste.CriarFamilia(t, db, "Familia com orcamento")
	outraFamilia := bancodeteste.CriarFamilia(t, db, "Outra familia")
	quemPediu := bancodeteste.CriarUsuario(t, db, "pediu@exemplo.invalid", false, familia)
	// Mesma familia, sem pedir o alerta: nao pode aparecer nos registros.
	bancodeteste.CriarUsuario(t, db, "nao-pediu@exemplo.invalid", false, familia)
	deOutraFamilia := bancodeteste.CriarUsuario(t, db, "outra@exemplo.invalid", false, outraFamilia)
	for _, usuarioID := range []int{quemPediu, deOutraFamilia} {
		if err := store.SaveNotificacoes(db, usuarioID, models.Notificacoes{DiaEnvio: 1, AlertaOrcamentoEmail: true}); err != nil {
			t.Fatalf("preferencias: %v", err)
		}
	}

	if err := store.CreateCategoria(db, familia, "Mercado", "basica", "#2ecc71"); err != nil {
		t.Fatalf("categoria: %v", err)
	}
	var mercadoID int
	db.QueryRow(`SELECT id FROM categorias WHERE family_id=$1`, familia).Scan(&mercadoID)
	limite := 1000.0
	if err := store.DefinirOrcamentoMensalDaCategoria(db, familia, mercadoID, &limite); err != nil {
		t.Fatalf("orcamento: %v", err)
	}
	if err := store.CreateDespesa(db, familia, "Feira", 850, "Mercado"); err != nil { // 85%
		t.Fatalf("despesa: %v", err)
	}

	madrugada := time.Date(2026, 10, 21, 7, 59, 0, 0, time.UTC)
	contas.EnviarAlertasDeOrcamento(db, madrugada)
	if registros := alertasRegistrados(t, db); len(registros) != 0 {
		t.Fatalf("enviou antes das 8h: %v", registros)
	}

	manha := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	contas.EnviarAlertasDeOrcamento(db, manha)
	contas.EnviarAlertasDeOrcamento(db, manha.Add(time.Hour)) // proxima volta do agendador
	if registros := alertasRegistrados(t, db); !mesmos(registros, []string{registro(quemPediu, "alerta")}) {
		t.Errorf("depois de chegar a 85%%: %v; esperado so o alerta de quem pediu", registros)
	}

	if err := store.CreateDespesa(db, familia, "Atacado", 300, "Mercado"); err != nil { // 115%
		t.Fatalf("despesa: %v", err)
	}
	contas.EnviarAlertasDeOrcamento(db, manha.Add(2*time.Hour))
	contas.EnviarAlertasDeOrcamento(db, manha.Add(3*time.Hour))
	esperado := []string{registro(quemPediu, "alerta"), registro(quemPediu, "estourado")}
	if registros := alertasRegistrados(t, db); !mesmos(registros, esperado) {
		t.Errorf("depois de passar de 100%%: %v; esperado %v", registros, esperado)
	}
}

func registro(usuarioID int, faixa string) string {
	return fmt.Sprintf("usuario %d: %s", usuarioID, faixa)
}

func alertasRegistrados(t *testing.T, db *sql.DB) []string {
	t.Helper()
	linhas, err := db.Query(`SELECT user_id, faixa FROM alertas_de_orcamento_enviados ORDER BY user_id, faixa`)
	if err != nil {
		t.Fatalf("ler alertas: %v", err)
	}
	defer linhas.Close()
	var registros []string
	for linhas.Next() {
		var usuarioID int
		var faixa string
		linhas.Scan(&usuarioID, &faixa)
		registros = append(registros, registro(usuarioID, faixa))
	}
	return registros
}

func mesmos(obtidos, esperados []string) bool {
	if len(obtidos) != len(esperados) {
		return false
	}
	for i := range obtidos {
		if obtidos[i] != esperados[i] {
			return false
		}
	}
	return true
}
