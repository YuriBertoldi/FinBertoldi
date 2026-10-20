package store

import (
	"database/sql"
	"fincontrol/internal/models"
)

func GetCartoes(db *sql.DB, fid int) ([]models.Cartao, error) {
	rows, err := db.Query(`
		SELECT id, nome, bandeira, limite, cor, ativo, criado_em
		FROM cartoes WHERE family_id=$1 ORDER BY nome`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Cartao
	for rows.Next() {
		var c models.Cartao
		rows.Scan(&c.ID, &c.Nome, &c.Bandeira, &c.Limite, &c.Cor, &c.Ativo, &c.CriadoEm)
		list = append(list, c)
	}
	return list, nil
}

func CreateCartao(db *sql.DB, fid int, nome, bandeira string, limite float64, cor string) error {
	_, err := db.Exec(`INSERT INTO cartoes (family_id, nome, bandeira, limite, cor) VALUES ($1,$2,$3,$4,$5)`,
		fid, nome, bandeira, limite, cor)
	return err
}

func UpdateCartao(db *sql.DB, fid, id int, nome, bandeira string, limite float64, cor string) error {
	_, err := db.Exec(`UPDATE cartoes SET nome=$1, bandeira=$2, limite=$3, cor=$4 WHERE id=$5 AND family_id=$6`,
		nome, bandeira, limite, cor, id, fid)
	return err
}

func DeleteCartao(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM cartoes WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func ToggleCartaoAtivo(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`UPDATE cartoes SET ativo = NOT ativo WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}
