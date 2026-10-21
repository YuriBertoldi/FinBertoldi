package store

import (
	"database/sql"
	"errors"
	"fincontrol/internal/models"
	"fmt"
	"math"
	"time"
)

func CreateTransacaoBancoBatch(db *sql.DB, fid int, txns []models.TransacaoBanco) (novos int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, t := range txns {
		if t.FitID != "" {
			var exists bool
			tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM transacoes_banco WHERE family_id=$1 AND fit_id=$2)`,
				fid, t.FitID).Scan(&exists)
			if exists {
				continue
			}
		}
		_, err := tx.Exec(`INSERT INTO transacoes_banco
			(family_id, data, descricao, valor, tipo, categoria, origem, banco, fit_id, status, pluggy_item_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			fid, t.Data, t.Descricao, t.Valor, t.Tipo, t.Categoria, t.Origem, t.Banco, t.FitID, "pendente", t.PluggyItemID)
		if err != nil {
			return novos, fmt.Errorf("insert transacao: %w", err)
		}
		novos++
	}
	return novos, tx.Commit()
}

func GetTransacoesBanco(db *sql.DB, fid int, mes, origem, status, banco string) ([]models.TransacaoBanco, error) {
	q := `SELECT id, data, descricao, valor, tipo, categoria, origem, banco, fit_id, status,
	             despesa_id, receita_id, parcelamento_id, pluggy_item_id, criado_em
	      FROM transacoes_banco WHERE family_id=$1`
	args := []any{fid}
	n := 2
	if mes != "" {
		q += fmt.Sprintf(` AND TO_CHAR(data, 'YYYY-MM') = $%d`, n)
		args = append(args, mes)
		n++
	}
	if origem != "" {
		q += fmt.Sprintf(` AND origem = $%d`, n)
		args = append(args, origem)
		n++
	}
	if status != "" {
		q += fmt.Sprintf(` AND status = $%d`, n)
		args = append(args, status)
		n++
	}
	if banco != "" {
		q += fmt.Sprintf(` AND banco = $%d`, n)
		args = append(args, banco)
	}
	q += ` ORDER BY data DESC, id DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.TransacaoBanco
	for rows.Next() {
		var t models.TransacaoBanco
		rows.Scan(&t.ID, &t.Data, &t.Descricao, &t.Valor, &t.Tipo, &t.Categoria,
			&t.Origem, &t.Banco, &t.FitID, &t.Status,
			&t.DespesaID, &t.ReceitaID, &t.ParcelamentoID, &t.PluggyItemID, &t.CriadoEm)
		list = append(list, t)
	}
	return list, nil
}

func ConverterTransacaoEmDespesa(db *sql.DB, fid, id int, categoria string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var t models.TransacaoBanco
	err = tx.QueryRow(`SELECT descricao, valor FROM transacoes_banco WHERE id=$1 AND family_id=$2`, id, fid).
		Scan(&t.Descricao, &t.Valor)
	if err != nil {
		return fmt.Errorf("transacao não encontrada: %w", err)
	}

	var despesaID int
	err = tx.QueryRow(`INSERT INTO despesas_fixas (family_id, nome, valor, categoria)
		VALUES ($1,$2,$3,$4) RETURNING id`, fid, t.Descricao, abs(t.Valor), categoria).Scan(&despesaID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE transacoes_banco SET status='convertida', despesa_id=$1, categoria=$2
		WHERE id=$3 AND family_id=$4`, despesaID, categoria, id, fid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func ConverterTransacaoEmReceita(db *sql.DB, fid, id int, tipo string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var descricao string
	var valor float64
	var data time.Time
	err = tx.QueryRow(`SELECT descricao, valor, data FROM transacoes_banco WHERE id=$1 AND family_id=$2`, id, fid).
		Scan(&descricao, &valor, &data)
	if err != nil {
		return fmt.Errorf("transacao não encontrada: %w", err)
	}

	var receitaID int
	err = tx.QueryRow(`INSERT INTO receitas (family_id, descricao, valor, data, tipo, recorrente)
		VALUES ($1,$2,$3,$4,$5,false) RETURNING id`, fid, descricao, abs(valor), data, tipo).Scan(&receitaID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE transacoes_banco SET status='convertida', receita_id=$1
		WHERE id=$2 AND family_id=$3`, receitaID, id, fid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func IgnorarTransacao(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`UPDATE transacoes_banco SET status='ignorada' WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func IgnorarTodasPendentes(db *sql.DB, fid int) (int64, error) {
	res, err := db.Exec(`UPDATE transacoes_banco SET status='ignorada' WHERE family_id=$1 AND status='pendente'`, fid)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func ConverterTodasPendentes(db *sql.DB, fid int) (int, error) {
	rows, err := db.Query(`SELECT id, descricao, valor, data, tipo, categoria FROM transacoes_banco
		WHERE family_id=$1 AND status='pendente'`, fid)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type txn struct {
		id        int
		descricao string
		valor     float64
		data      time.Time
		tipo      string
		categoria string
	}
	var txns []txn
	for rows.Next() {
		var t txn
		rows.Scan(&t.id, &t.descricao, &t.valor, &t.data, &t.tipo, &t.categoria)
		txns = append(txns, t)
	}

	count := 0
	for _, t := range txns {
		tx, err := db.Begin()
		if err != nil {
			continue
		}
		if t.tipo == "credito" {
			tipo := "Outros"
			var receitaID int
			err = tx.QueryRow(`INSERT INTO receitas (family_id, descricao, valor, data, tipo, recorrente)
				VALUES ($1,$2,$3,$4,$5,false) RETURNING id`, fid, t.descricao, abs(t.valor), t.data, tipo).Scan(&receitaID)
			if err != nil {
				tx.Rollback()
				continue
			}
			_, err = tx.Exec(`UPDATE transacoes_banco SET status='convertida', receita_id=$1
				WHERE id=$2 AND family_id=$3`, receitaID, t.id, fid)
		} else {
			cat := t.categoria
			if cat == "" {
				cat = "Outros"
			}
			var despesaID int
			err = tx.QueryRow(`INSERT INTO despesas_fixas (family_id, nome, valor, categoria)
				VALUES ($1,$2,$3,$4) RETURNING id`, fid, t.descricao, abs(t.valor), cat).Scan(&despesaID)
			if err != nil {
				tx.Rollback()
				continue
			}
			_, err = tx.Exec(`UPDATE transacoes_banco SET status='convertida', despesa_id=$1, categoria=$2
				WHERE id=$3 AND family_id=$4`, despesaID, cat, t.id, fid)
		}
		if err != nil {
			tx.Rollback()
			continue
		}
		tx.Commit()
		count++
	}
	return count, nil
}

func CategorizarTransacao(db *sql.DB, fid, id int, categoria string) error {
	_, err := db.Exec(`UPDATE transacoes_banco SET categoria=$1, status='categorizada' WHERE id=$2 AND family_id=$3`,
		categoria, id, fid)
	return err
}

// ErrVinculoNaoEncontrado: a transacao ou o registro de destino nao existe na familia.
var ErrVinculoNaoEncontrado = errors.New("transação ou registro de destino não encontrado")

// VincularTransacaoDespesa vincula uma transação bancária a uma despesa fixa da familia.
func VincularTransacaoDespesa(db *sql.DB, fid, txnID, despesaID int) error {
	return vincularTransacao(db, fid, txnID, "despesa_id", "despesas_fixas", despesaID)
}

// VincularTransacaoReceita vincula uma transação bancária a uma receita da familia.
func VincularTransacaoReceita(db *sql.DB, fid, txnID, receitaID int) error {
	return vincularTransacao(db, fid, txnID, "receita_id", "receitas", receitaID)
}

// VincularTransacaoParcelamento vincula uma transação bancária a um parcelamento da familia.
func VincularTransacaoParcelamento(db *sql.DB, fid, txnID, parcelamentoID int) error {
	return vincularTransacao(db, fid, txnID, "parcelamento_id", "parcelamentos", parcelamentoID)
}

// vincularTransacao grava o vinculo so se a transacao e o registro de destino
// forem da familia. colunaDoVinculo e tabelaDeDestino sao constantes dos
// chamadores acima, nunca vem do usuario.
func vincularTransacao(db *sql.DB, fid, txnID int, colunaDoVinculo, tabelaDeDestino string, destinoID int) error {
	resultado, err := db.Exec(`UPDATE transacoes_banco SET status='convertida', `+colunaDoVinculo+`=$1
		WHERE id=$2 AND family_id=$3
		  AND EXISTS (SELECT 1 FROM `+tabelaDeDestino+` WHERE id=$1 AND family_id=$3)`, destinoID, txnID, fid)
	if err != nil {
		return err
	}
	if linhasAlteradas, _ := resultado.RowsAffected(); linhasAlteradas == 0 {
		return ErrVinculoNaoEncontrado
	}
	return nil
}

// MatchDespesa representa um match entre transação e despesa
type MatchDespesa struct {
	ID        int
	Nome      string
	Valor     float64
	Categoria string
	Score     int // 0-100
}

// MatchReceita representa um match entre transação e receita
type MatchReceita struct {
	ID        int
	Descricao string
	Valor     float64
	Data      time.Time
	Tipo      string
	Score     int
}

// BuscarMatchesDespesas busca despesas que combinam com a transação por valor e nome
func BuscarMatchesDespesas(db *sql.DB, fid int, valor float64, descricao string) ([]MatchDespesa, error) {
	// Busca despesas com valor similar (±20%) e ativas. Debito chega negativo
	// do extrato; despesa e cadastrada positiva.
	valor = math.Abs(valor)
	margem := valor * 0.20
	rows, err := db.Query(`SELECT id, nome, valor, categoria
		FROM despesas_fixas WHERE family_id=$1 AND ativa=true
		AND valor BETWEEN $2 AND $3
		ORDER BY ABS(valor - $4) ASC LIMIT 10`,
		fid, valor-margem, valor+margem, valor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []MatchDespesa
	for rows.Next() {
		var m MatchDespesa
		rows.Scan(&m.ID, &m.Nome, &m.Valor, &m.Categoria)
		// Score: 100 se valor exato, diminui com diferença
		diff := math.Abs(m.Valor-valor) / valor * 100
		m.Score = 100 - int(diff)
		if m.Score < 0 {
			m.Score = 0
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// BuscarMatchesReceitas busca receitas que combinam com a transação por valor e data
func BuscarMatchesReceitas(db *sql.DB, fid int, valor float64, data time.Time) ([]MatchReceita, error) {
	valor = math.Abs(valor)
	margem := valor * 0.20
	rows, err := db.Query(`SELECT id, descricao, valor, data, tipo
		FROM receitas WHERE family_id=$1
		AND valor BETWEEN $2 AND $3
		AND data BETWEEN $4 AND $5
		ORDER BY ABS(valor - $6) ASC LIMIT 10`,
		fid, valor-margem, valor+margem,
		data.AddDate(0, 0, -5), data.AddDate(0, 0, 5), valor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []MatchReceita
	for rows.Next() {
		var m MatchReceita
		rows.Scan(&m.ID, &m.Descricao, &m.Valor, &m.Data, &m.Tipo)
		diff := math.Abs(m.Valor-valor) / valor * 100
		m.Score = 100 - int(diff)
		if m.Score < 0 {
			m.Score = 0
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// BuscarMatchesParcelamentos busca parcelas que combinam por valor no mês da transação
func BuscarMatchesParcelamentos(db *sql.DB, fid int, valor float64, data time.Time) ([]MatchDespesa, error) {
	// parcelamentos_mes.mes e DATE (primeiro dia do mes); o texto "2026-10"
	// nao e data valida no PostgreSQL. Debito chega negativo; a parcela, positiva.
	mes := primeiroDiaDoMes(data)
	valor = math.Abs(valor)
	margem := valor * 0.20
	rows, err := db.Query(`SELECT p.id, p.descricao, p.valor_parcela, COALESCE(p.descricao,'')
		FROM parcelamentos p
		JOIN parcelamentos_mes pm ON pm.parcelamento_id = p.id AND pm.mes = $2
		WHERE p.family_id=$1 AND pm.pago=false
		AND p.valor_parcela BETWEEN $3 AND $4
		ORDER BY ABS(p.valor_parcela - $5) ASC LIMIT 10`,
		fid, mes, valor-margem, valor+margem, valor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []MatchDespesa
	for rows.Next() {
		var m MatchDespesa
		rows.Scan(&m.ID, &m.Nome, &m.Valor, &m.Categoria)
		diff := math.Abs(m.Valor-valor) / valor * 100
		m.Score = 100 - int(diff)
		if m.Score < 0 {
			m.Score = 0
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// DesvincularTransacao remove a vinculação de uma transação
func DesvincularTransacao(db *sql.DB, fid, txnID int) error {
	_, err := db.Exec(`UPDATE transacoes_banco SET status='pendente', despesa_id=NULL, receita_id=NULL, parcelamento_id=NULL
		WHERE id=$1 AND family_id=$2`, txnID, fid)
	return err
}

// AutoMatchTransacoes faz match automático de transações pendentes
func AutoMatchTransacoes(db *sql.DB, fid int) (matched int, err error) {
	txns, err := GetTransacoesBanco(db, fid, "", "", "pendente", "")
	if err != nil {
		return 0, err
	}
	for _, t := range txns {
		if t.Tipo == "debito" {
			matches, _ := BuscarMatchesDespesas(db, fid, t.Valor, t.Descricao)
			if len(matches) > 0 && matches[0].Score >= 95 {
				VincularTransacaoDespesa(db, fid, t.ID, matches[0].ID)
				matched++
				continue
			}
			pMatches, _ := BuscarMatchesParcelamentos(db, fid, t.Valor, t.Data)
			if len(pMatches) > 0 && pMatches[0].Score >= 95 {
				VincularTransacaoParcelamento(db, fid, t.ID, pMatches[0].ID)
				matched++
			}
		} else {
			matches, _ := BuscarMatchesReceitas(db, fid, t.Valor, t.Data)
			if len(matches) > 0 && matches[0].Score >= 95 {
				VincularTransacaoReceita(db, fid, t.ID, matches[0].ID)
				matched++
			}
		}
	}
	return matched, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
