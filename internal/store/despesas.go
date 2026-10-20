package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"fmt"
	"time"
)

func GetDespesasFixas(db *sql.DB, fid int) ([]models.DespesaFixa, error) {
	rows, err := db.Query(`
		SELECT df.id, df.nome, df.valor, df.categoria,
		       COALESCE(c.grupo, 'basica') AS grupo,
		       df.ativa, df.criado_em
		FROM despesas_fixas df
		LEFT JOIN categorias c ON c.family_id = $1 AND c.nome = df.categoria AND c.ativo = true
		WHERE df.family_id=$1 ORDER BY df.categoria, df.nome`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.DespesaFixa
	for rows.Next() {
		var d models.DespesaFixa
		rows.Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria, &d.Grupo, &d.Ativa, &d.CriadoEm)
		list = append(list, d)
	}
	return list, nil
}

func CreateDespesa(db *sql.DB, fid int, nome string, valor float64, categoria string) error {
	_, err := db.Exec(`INSERT INTO despesas_fixas (family_id, nome, valor, categoria) VALUES ($1,$2,$3,$4)`,
		fid, nome, valor, categoria)
	return err
}

func UpdateDespesa(db *sql.DB, fid, id int, nome string, valor float64, categoria string) error {
	_, err := db.Exec(`UPDATE despesas_fixas SET nome=$1, valor=$2, categoria=$3 WHERE id=$4 AND family_id=$5`,
		nome, valor, categoria, id, fid)
	return err
}

func DeleteDespesa(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM despesas_fixas WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func ToggleDespesaAtiva(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`UPDATE despesas_fixas SET ativa = NOT ativa WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func EnsureDespesasMes(db *sql.DB, fid int, mes time.Time) error {
	_, err := db.Exec(`
		INSERT INTO despesas_fixas_mes (despesa_id, mes)
		SELECT id, $1 FROM despesas_fixas WHERE ativa = true AND family_id = $2
		ON CONFLICT (despesa_id, mes) DO NOTHING`, mes, fid)
	return err
}

func GetDespesasMes(db *sql.DB, fid int, mes time.Time, mesStr string) ([]models.DespesaMes, []models.DespesaMes, []models.DespesaMes, error) {
	rows, err := db.Query(`
		SELECT df.id, df.nome, df.valor, df.categoria,
		       COALESCE(c.grupo, 'basica') AS grupo,
		       COALESCE(dfm.pago, false) AS pago
		FROM despesas_fixas df
		LEFT JOIN categorias c ON c.family_id = $2 AND c.nome = df.categoria AND c.ativo = true
		LEFT JOIN despesas_fixas_mes dfm ON dfm.despesa_id = df.id AND dfm.mes = $1
		WHERE df.ativa = true AND df.family_id = $2
		ORDER BY df.categoria, df.nome`, mes, fid)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	var basicas, cartao, vr []models.DespesaMes
	for rows.Next() {
		var d models.DespesaMes
		rows.Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria, &d.Grupo, &d.Pago)
		d.Mes = mesStr
		switch d.Grupo {
		case "cartao":
			cartao = append(cartao, d)
		case "vr":
			vr = append(vr, d)
		default:
			basicas = append(basicas, d)
		}
	}
	return basicas, cartao, vr, nil
}

func ToggleDespesaMesPago(db *sql.DB, fid, despesaID int, mes time.Time) (bool, error) {
	var ok bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM despesas_fixas WHERE id=$1 AND family_id=$2)`, despesaID, fid).Scan(&ok)
	if !ok {
		return false, fmt.Errorf("despesa não pertence à família")
	}
	_, err := db.Exec(`
		INSERT INTO despesas_fixas_mes (despesa_id, mes, pago) VALUES ($1,$2,false)
		ON CONFLICT (despesa_id, mes) DO NOTHING`, despesaID, mes)
	if err != nil {
		return false, err
	}
	var pago bool
	err = db.QueryRow(`
		UPDATE despesas_fixas_mes SET pago = NOT pago, pago_em = CASE WHEN NOT pago THEN NOW() ELSE NULL END
		WHERE despesa_id=$1 AND mes=$2
		RETURNING pago`, despesaID, mes).Scan(&pago)
	return pago, err
}
