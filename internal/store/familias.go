package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"fmt"
)

func GetFamilies(db *sql.DB) ([]models.Family, error) {
	rows, err := db.Query(`
		SELECT f.id, f.nome, COALESCE(COUNT(u.id),0), f.criado_em
		FROM families f
		LEFT JOIN users u ON u.family_id = f.id
		GROUP BY f.id, f.nome, f.criado_em
		ORDER BY f.nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Family
	for rows.Next() {
		var f models.Family
		rows.Scan(&f.ID, &f.Nome, &f.NumMembros, &f.CriadoEm)
		list = append(list, f)
	}
	return list, nil
}

func CreateFamily(db *sql.DB, nome string) (int, error) {
	var id int
	err := db.QueryRow(`INSERT INTO families (nome) VALUES ($1) RETURNING id`, nome).Scan(&id)
	return id, err
}

func RenameFamily(db *sql.DB, id int, nome string) error {
	_, err := db.Exec(`UPDATE families SET nome=$1 WHERE id=$2`, nome, id)
	return err
}

func DeleteFamily(db *sql.DB, id int) error {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE family_id=$1`, id).Scan(&n)
	if n > 0 {
		return fmt.Errorf("não é possível excluir família com %d usuário(s)", n)
	}
	_, err := db.Exec(`DELETE FROM families WHERE id=$1`, id)
	return err
}
