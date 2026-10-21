package store

import (
	"database/sql"
	"strings"
	"time"

	"fincontrol/internal/models"
)

func CriarMeta(db *sql.DB, fid int, nome string, valorAlvo float64, dataAlvo time.Time) error {
	_, err := db.Exec(`INSERT INTO metas (family_id, nome, valor_alvo, data_alvo) VALUES ($1,$2,$3,$4)`,
		fid, strings.TrimSpace(nome), valorAlvo, primeiroDiaDoMes(dataAlvo))
	return err
}

func ExcluirMeta(db *sql.DB, fid, metaID int) error {
	_, err := db.Exec(`DELETE FROM metas WHERE id=$1 AND family_id=$2`, metaID, fid)
	return err
}

// RegistrarAporteNaMeta grava o aporte (valor negativo = retirada) se a meta
// for da familia.
func RegistrarAporteNaMeta(db *sql.DB, fid, metaID int, valor float64, data time.Time) error {
	_, err := db.Exec(`INSERT INTO metas_aportes (meta_id, family_id, valor, data)
		SELECT id, family_id, $3, $4 FROM metas WHERE id=$1 AND family_id=$2`, metaID, fid, valor, data)
	return err
}

// Metas devolve as metas da familia com o total guardado, pelo prazo mais proximo.
func Metas(db *sql.DB, fid int) ([]models.Meta, error) {
	linhas, err := db.Query(`
		SELECT m.id, m.nome, m.valor_alvo, m.data_alvo,
		       COALESCE((SELECT SUM(a.valor) FROM metas_aportes a WHERE a.meta_id = m.id), 0)
		FROM metas m WHERE m.family_id = $1
		ORDER BY m.data_alvo, m.id`, fid)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()

	var metas []models.Meta
	for linhas.Next() {
		var meta models.Meta
		if err := linhas.Scan(&meta.ID, &meta.Nome, &meta.ValorAlvo, &meta.DataAlvo, &meta.ValorGuardado); err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	return metas, linhas.Err()
}

func primeiroDiaDoMes(data time.Time) time.Time {
	return time.Date(data.Year(), data.Month(), 1, 0, 0, 0, 0, time.UTC)
}
