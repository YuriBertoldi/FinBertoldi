package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"time"
)

func GetReceitas(db *sql.DB, fid int) ([]models.Receita, error) {
	rows, err := db.Query(`
		SELECT id, descricao, valor, data, tipo, recorrente, criado_em
		FROM receitas WHERE family_id=$1 ORDER BY data DESC, criado_em DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Receita
	for rows.Next() {
		var r models.Receita
		rows.Scan(&r.ID, &r.Descricao, &r.Valor, &r.Data, &r.Tipo, &r.Recorrente, &r.CriadoEm)
		list = append(list, r)
	}
	return list, nil
}

func GetReceitasMes(db *sql.DB, fid int, mes time.Time) (float64, error) {
	var total float64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(valor), 0)
		FROM receitas
		WHERE family_id = $2 AND (recorrente = true OR date_trunc('month', data) = $1::date)`, mes, fid).Scan(&total)
	return total, err
}

func CreateReceita(db *sql.DB, fid int, descricao string, valor float64, data time.Time, tipo string, recorrente bool) error {
	_, err := db.Exec(`INSERT INTO receitas (family_id, descricao, valor, data, tipo, recorrente) VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, descricao, valor, data, tipo, recorrente)
	return err
}

func UpdateReceita(db *sql.DB, fid, id int, descricao string, valor float64, data time.Time, tipo string, recorrente bool) error {
	_, err := db.Exec(`UPDATE receitas SET descricao=$1, valor=$2, data=$3, tipo=$4, recorrente=$5 WHERE id=$6 AND family_id=$7`,
		descricao, valor, data, tipo, recorrente, id, fid)
	return err
}

func DeleteReceita(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM receitas WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}
