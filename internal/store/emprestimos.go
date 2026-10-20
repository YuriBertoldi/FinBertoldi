package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"fmt"
	"time"
)

func GetEmprestimos(db *sql.DB, fid int) ([]models.Emprestimo, error) {
	rows, err := db.Query(`
		SELECT e.id, e.pessoa, e.valor, e.direcao, e.data, e.pago, e.notas, e.criado_em,
		       COALESCE(SUM(ep.valor), 0) AS valor_pago
		FROM emprestimos e
		LEFT JOIN emprestimo_pagamentos ep ON ep.emprestimo_id = e.id
		WHERE e.family_id = $1
		GROUP BY e.id, e.pessoa, e.valor, e.direcao, e.data, e.pago, e.notas, e.criado_em
		ORDER BY e.pago ASC, e.data DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Emprestimo
	for rows.Next() {
		var e models.Emprestimo
		rows.Scan(&e.ID, &e.Pessoa, &e.Valor, &e.Direcao, &e.Data, &e.Pago, &e.Notas, &e.CriadoEm, &e.ValorPago)
		e.ValorRestante = e.Valor - e.ValorPago
		if e.ValorRestante < 0 {
			e.ValorRestante = 0
		}
		list = append(list, e)
	}
	return list, nil
}

func AddPagamentoEmprestimo(db *sql.DB, fid, empID int, valor float64, data time.Time, notas string) error {
	var ok bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM emprestimos WHERE id=$1 AND family_id=$2)`, empID, fid).Scan(&ok)
	if !ok {
		return fmt.Errorf("empréstimo não encontrado")
	}
	_, err := db.Exec(`INSERT INTO emprestimo_pagamentos (emprestimo_id, family_id, valor, data, notas) VALUES ($1,$2,$3,$4,$5)`,
		empID, fid, valor, data, notas)
	if err != nil {
		return err
	}
	var valorOrig, totalPago float64
	db.QueryRow(`SELECT valor FROM emprestimos WHERE id=$1`, empID).Scan(&valorOrig)
	db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM emprestimo_pagamentos WHERE emprestimo_id=$1`, empID).Scan(&totalPago)
	if totalPago >= valorOrig {
		db.Exec(`UPDATE emprestimos SET pago = true, pago_em = NOW() WHERE id=$1 AND family_id=$2 AND NOT pago`, empID, fid)
	}
	return nil
}

func CreateEmprestimo(db *sql.DB, fid int, pessoa string, valor float64, direcao string, data time.Time, notas string) error {
	_, err := db.Exec(`INSERT INTO emprestimos (family_id, pessoa, valor, direcao, data, notas) VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, pessoa, valor, direcao, data, notas)
	return err
}

func DeleteEmprestimo(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM emprestimos WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func ToggleEmprestimoPago(db *sql.DB, fid, id int) (bool, error) {
	var pago bool
	err := db.QueryRow(`
		UPDATE emprestimos SET pago = NOT pago, pago_em = CASE WHEN NOT pago THEN NOW() ELSE NULL END
		WHERE id=$1 AND family_id=$2 RETURNING pago`, id, fid).Scan(&pago)
	return pago, err
}
