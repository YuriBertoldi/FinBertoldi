package store

import (
	"database/sql"
	"fincontrol/internal/models"
)

func GetCategorias(db *sql.DB, fid int) ([]models.Categoria, error) {
	rows, err := db.Query(`
		SELECT id, nome, grupo, cor, ativo, criado_em, orcamento_mensal
		FROM categorias WHERE family_id=$1 ORDER BY grupo, nome`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Categoria
	for rows.Next() {
		var c models.Categoria
		rows.Scan(&c.ID, &c.Nome, &c.Grupo, &c.Cor, &c.Ativo, &c.CriadoEm, &c.OrcamentoMensal)
		list = append(list, c)
	}
	return list, nil
}

func CreateCategoria(db *sql.DB, fid int, nome, grupo, cor string) error {
	_, err := db.Exec(`INSERT INTO categorias (family_id, nome, grupo, cor) VALUES ($1,$2,$3,$4)`,
		fid, nome, grupo, cor)
	return err
}

func UpdateCategoria(db *sql.DB, fid, id int, nome, grupo, cor string) error {
	_, err := db.Exec(`UPDATE categorias SET nome=$1, grupo=$2, cor=$3 WHERE id=$4 AND family_id=$5`,
		nome, grupo, cor, id, fid)
	return err
}

func DeleteCategoria(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM categorias WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func ToggleCategoriaAtiva(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`UPDATE categorias SET ativo = NOT ativo WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}
