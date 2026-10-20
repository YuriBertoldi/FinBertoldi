package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"time"
)

func GetReservaEM(db *sql.DB, fid int) (float64, error) {
	var v float64
	err := db.QueryRow(`SELECT COALESCE(SUM(CASE WHEN tipo='deposito' THEN valor ELSE -valor END), 0) FROM reserva_emergencia WHERE family_id=$1`, fid).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return v, err
}

func GetHistoricoReserva(db *sql.DB, fid int) ([]models.ReservaEM, error) {
	rows, err := db.Query(`SELECT id, valor, tipo, data, notas, criado_em FROM reserva_emergencia WHERE family_id=$1 ORDER BY data DESC, criado_em DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.ReservaEM
	for rows.Next() {
		var r models.ReservaEM
		rows.Scan(&r.ID, &r.Valor, &r.Tipo, &r.Data, &r.Notas, &r.CriadoEm)
		list = append(list, r)
	}
	return list, nil
}

func AddReservaEM(db *sql.DB, fid int, valor float64, data time.Time, tipo, notas string) error {
	_, err := db.Exec(`INSERT INTO reserva_emergencia (family_id, valor, data, tipo, notas) VALUES ($1,$2,$3,$4,$5)`, fid, valor, data, tipo, notas)
	return err
}

func UpdateReservaEM(db *sql.DB, fid, id int, valor float64, data time.Time, tipo, notas string) error {
	_, err := db.Exec(`UPDATE reserva_emergencia SET valor=$1, data=$2, tipo=$3, notas=$4 WHERE id=$5 AND family_id=$6`, valor, data, tipo, notas, id, fid)
	return err
}

func DeleteReservaEM(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM reserva_emergencia WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}
