package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"time"
)

func UpsertFeriado(db *sql.DB, data time.Time, nome, tipo string) error {
	_, err := db.Exec(`INSERT INTO feriados (data, nome, tipo) VALUES ($1,$2,$3)
		ON CONFLICT (data) DO UPDATE SET nome=$2, tipo=$3`, data, nome, tipo)
	return err
}

func GetFeriadosAno(db *sql.DB, ano int) ([]models.Feriado, error) {
	rows, err := db.Query(`SELECT data, nome, tipo FROM feriados
		WHERE EXTRACT(YEAR FROM data) = $1 ORDER BY data`, ano)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Feriado
	for rows.Next() {
		var f models.Feriado
		rows.Scan(&f.Data, &f.Nome, &f.Tipo)
		list = append(list, f)
	}
	return list, nil
}

func GetFeriadosProximos(db *sql.DB, limit int) ([]models.Feriado, error) {
	rows, err := db.Query(`SELECT data, nome, tipo FROM feriados
		WHERE data >= CURRENT_DATE ORDER BY data LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Feriado
	for rows.Next() {
		var f models.Feriado
		rows.Scan(&f.Data, &f.Nome, &f.Tipo)
		list = append(list, f)
	}
	return list, nil
}
