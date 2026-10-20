package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"time"
)

func GetInvestimentos(db *sql.DB, fid int) ([]models.Investimento, error) {
	rows, err := db.Query(`
		SELECT id, instituicao, tipo, valor, data, notas, criado_em
		FROM investimentos WHERE family_id=$1 ORDER BY data DESC, criado_em DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Investimento
	for rows.Next() {
		var i models.Investimento
		rows.Scan(&i.ID, &i.Instituicao, &i.Tipo, &i.Valor, &i.Data, &i.Notas, &i.CriadoEm)
		list = append(list, i)
	}
	return list, nil
}

func GetInvestimentosTotais(db *sql.DB, fid int) (map[string]float64, float64, error) {
	rows, err := db.Query(`
		SELECT instituicao, SUM(valor) FROM investimentos WHERE family_id=$1 GROUP BY instituicao ORDER BY instituicao`, fid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	totais := map[string]float64{}
	var total float64
	for rows.Next() {
		var inst string
		var v float64
		rows.Scan(&inst, &v)
		totais[inst] = v
		total += v
	}
	return totais, total, nil
}

func CreateInvestimento(db *sql.DB, fid int, instituicao, tipo string, valor float64, data time.Time, notas string) error {
	_, err := db.Exec(`INSERT INTO investimentos (family_id, instituicao, tipo, valor, data, notas) VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, instituicao, tipo, valor, data, notas)
	return err
}

func UpdateInvestimento(db *sql.DB, fid, id int, instituicao, tipo string, valor float64, data time.Time, notas string) error {
	_, err := db.Exec(`UPDATE investimentos SET instituicao=$1, tipo=$2, valor=$3, data=$4, notas=$5 WHERE id=$6 AND family_id=$7`,
		instituicao, tipo, valor, data, notas, id, fid)
	return err
}

func DeleteInvestimento(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM investimentos WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func GetInvestidoNoMes(db *sql.DB, fid int, mes time.Time) float64 {
	var totalInv, totalReserva float64
	db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM investimentos WHERE family_id=$1 AND date_trunc('month', data) = date_trunc('month', $2::date)`, fid, mes).Scan(&totalInv)
	db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM reserva_emergencia WHERE family_id=$1 AND tipo='deposito' AND date_trunc('month', data) = date_trunc('month', $2::date)`, fid, mes).Scan(&totalReserva)
	return totalInv + totalReserva
}
