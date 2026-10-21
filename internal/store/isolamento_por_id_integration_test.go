//go:build integration

package store

import (
	"database/sql"
	"testing"
	"time"

	"fincontrol/internal/models"
)

// Toda operacao que recebe o id de um registro precisa filtrar por
// family_id: senao, quem troca o id na URL altera dado de outra familia.
// Os testes antigos cobrem despesas, receitas, investimentos e reserva em
// update; este cobre as demais operacoes por id.
//
// Cada caso roda duas vezes contra o mesmo registro da familia A:
//  1. pela familia B: nada da familia A pode mudar;
//  2. pela familia A (controle): algo precisa mudar, para provar que a
//     operacao foi de fato exercitada e o caso 1 nao passou por acaso.
func TestIsolamento_OperacoesPorIDNaoAlteramOutraFamilia(t *testing.T) {
	mesDoTeste := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	casos := []struct {
		nome     string
		executar func(db *sql.DB, familiaID int, registros registrosDaFamilia) error
	}{
		{"atualizar categoria", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return UpdateCategoria(db, fid, r.categoriaID, "Alterada", "basica", "#000000")
		}},
		{"excluir categoria", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteCategoria(db, fid, r.categoriaID)
		}},
		{"ativar/desativar categoria", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return ToggleCategoriaAtiva(db, fid, r.categoriaID)
		}},
		{"definir orcamento da categoria", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			limite := 750.0
			return DefinirOrcamentoMensalDaCategoria(db, fid, r.categoriaID, &limite)
		}},
		{"atualizar cartao", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return UpdateCartao(db, fid, r.cartaoID, "Alterado", "visa", 1, "#000000")
		}},
		{"excluir cartao", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteCartao(db, fid, r.cartaoID)
		}},
		{"ativar/desativar cartao", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return ToggleCartaoAtivo(db, fid, r.cartaoID)
		}},
		{"excluir investimento", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteInvestimento(db, fid, r.investimentoID)
		}},
		{"excluir receita", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteReceita(db, fid, r.receitaID)
		}},
		{"excluir movimento da reserva", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteReservaEM(db, fid, r.reservaID)
		}},
		{"excluir emprestimo", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteEmprestimo(db, fid, r.emprestimoID)
		}},
		{"registrar pagamento de emprestimo", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return AddPagamentoEmprestimo(db, fid, r.emprestimoID, 10, mesDoTeste, "")
		}},
		{"atualizar parcelamento", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return UpdateParcelamento(db, fid, r.parcelamentoID, 5, false)
		}},
		{"excluir parcelamento", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeleteParcelamento(db, fid, r.parcelamentoID)
		}},
		{"marcar parcela do mes como paga", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			_, err := ToggleParcelamentoMesPago(db, fid, r.parcelamentoID, mesDoTeste)
			return err
		}},
		{"antecipar parcelas", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			_, err := AnteciparParcelas(db, fid, r.parcelamentoID, 2)
			return err
		}},
		{"converter transacao em despesa", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return ConverterTransacaoEmDespesa(db, fid, r.transacaoID, "Mercado")
		}},
		{"converter transacao em receita", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return ConverterTransacaoEmReceita(db, fid, r.transacaoID, "extra")
		}},
		{"ignorar transacao", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return IgnorarTransacao(db, fid, r.transacaoID)
		}},
		{"categorizar transacao", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return CategorizarTransacao(db, fid, r.transacaoID, "Mercado")
		}},
		{"excluir regra de categorizacao", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return ExcluirRegraDeCategorizacao(db, fid, r.regraID)
		}},
		{"excluir conexao Pluggy", func(db *sql.DB, fid int, r registrosDaFamilia) error {
			return DeletePluggyItem(db, fid, r.pluggyItemID)
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			db := setupTestDB(t)
			familiaA, familiaB := criarDuasFamilias(t, db)
			registrosDeA := criarRegistrosDaFamilia(t, db, familiaA, mesDoTeste)

			antes := fotografiaDaFamilia(t, db, familiaA)
			caso.executar(db, familiaB, registrosDeA) // o erro nao importa: importa o que mudou
			if depois := fotografiaDaFamilia(t, db, familiaA); depois != antes {
				t.Errorf("a familia B alterou dados da familia A")
			}

			if err := caso.executar(db, familiaA, registrosDeA); err != nil {
				t.Fatalf("controle pela propria familia falhou: %v", err)
			}
			if depois := fotografiaDaFamilia(t, db, familiaA); depois == antes {
				t.Error("nem a propria familia conseguiu alterar; o caso nao esta exercitando a operacao")
			}
		})
	}
}

type registrosDaFamilia struct {
	categoriaID, cartaoID, investimentoID, receitaID, reservaID int
	emprestimoID, parcelamentoID, transacaoID, regraID          int
	pluggyItemID                                                string
}

func criarRegistrosDaFamilia(t *testing.T, db *sql.DB, fid int, mes time.Time) registrosDaFamilia {
	t.Helper()
	falharSeErro := func(descricao string, err error) {
		if err != nil {
			t.Fatalf("%s: %v", descricao, err)
		}
	}
	falharSeErro("categoria", CreateCategoria(db, fid, "Mercado", "basica", "#111111"))
	falharSeErro("cartao", CreateCartao(db, fid, "Cartao", "visa", 5000, "#222222"))
	falharSeErro("investimento", CreateInvestimento(db, fid, "Banco", "CDB", 1000, mes, ""))
	falharSeErro("receita", CreateReceita(db, fid, "Salario", 5000, mes, "salario", true))
	falharSeErro("reserva", AddReservaEM(db, fid, 300, mes, "deposito", ""))
	falharSeErro("emprestimo", CreateEmprestimo(db, fid, "Fulano", 200, "emprestei", mes, ""))
	// Financiamento com juros e parcelas futuras, para a antecipacao ter o que antecipar.
	falharSeErro("parcelamento", CreateParcelamento(db, fid, "Carro", "Cartao", 500, 3, 24, mes.AddDate(0, -2, 0), true, 12000, 1.5))
	_, err := CreateTransacaoBancoBatch(db, fid, []models.TransacaoBanco{{
		Data: mes, Descricao: "COMPRA MERCADO", Valor: -80, Tipo: "debito", Origem: "ofx", Banco: "Banco", FitID: "transacao-de-teste",
	}})
	falharSeErro("transacao", err)
	falharSeErro("pluggy", CreatePluggyItem(db, fid, "item-de-teste", "Banco"))
	var categoriaID int
	db.QueryRow(`SELECT id FROM categorias WHERE family_id=$1`, fid).Scan(&categoriaID)
	falharSeErro("regra", CriarRegraDeCategorizacao(db, fid, "descricao", "MERCADO", categoriaID))

	var registros registrosDaFamilia
	registros.pluggyItemID = "item-de-teste"
	for tabela, destino := range map[string]*int{
		"categorias":              &registros.categoriaID,
		"cartoes":                 &registros.cartaoID,
		"investimentos":           &registros.investimentoID,
		"receitas":                &registros.receitaID,
		"reserva_emergencia":      &registros.reservaID,
		"emprestimos":             &registros.emprestimoID,
		"parcelamentos":           &registros.parcelamentoID,
		"transacoes_banco":        &registros.transacaoID,
		"regras_de_categorizacao": &registros.regraID,
	} {
		if err := db.QueryRow(`SELECT id FROM `+tabela+` WHERE family_id=$1 ORDER BY id LIMIT 1`, fid).Scan(destino); err != nil {
			t.Fatalf("ler id em %s: %v", tabela, err)
		}
	}
	return registros
}

// fotografiaDaFamilia serializa todas as linhas da familia nas tabelas que as
// operacoes do teste podem tocar, incluindo as tabelas filhas sem family_id.
func fotografiaDaFamilia(t *testing.T, db *sql.DB, fid int) string {
	t.Helper()
	consultas := []string{
		`SELECT to_jsonb(x)::text FROM categorias x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM cartoes x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM despesas_fixas x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM receitas x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM investimentos x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM reserva_emergencia x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM emprestimos x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM emprestimo_pagamentos x WHERE emprestimo_id IN (SELECT id FROM emprestimos WHERE family_id=$1)`,
		`SELECT to_jsonb(x)::text FROM parcelamentos x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM parcelamentos_mes x WHERE parcelamento_id IN (SELECT id FROM parcelamentos WHERE family_id=$1)`,
		`SELECT to_jsonb(x)::text FROM transacoes_banco x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM pluggy_items x WHERE family_id=$1`,
		`SELECT to_jsonb(x)::text FROM regras_de_categorizacao x WHERE family_id=$1`,
	}
	var fotografia string
	for _, consulta := range consultas {
		linhas, err := db.Query(consulta+` ORDER BY 1`, fid)
		if err != nil {
			t.Fatalf("fotografia (%s): %v", consulta, err)
		}
		for linhas.Next() {
			var linha string
			if err := linhas.Scan(&linha); err != nil {
				t.Fatalf("fotografia: %v", err)
			}
			fotografia += linha + "\n"
		}
		linhas.Close()
	}
	return fotografia
}
