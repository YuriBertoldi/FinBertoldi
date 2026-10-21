package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"fmt"
	"time"
)

func GetHistoricoMeses(db *sql.DB, fid int, mesRef time.Time, numMeses int) ([]models.MesResumo, error) {
	var resumos []models.MesResumo
	mesesPt := []string{"", "Jan", "Fev", "Mar", "Abr", "Mai", "Jun",
		"Jul", "Ago", "Set", "Out", "Nov", "Dez"}

	for i := numMeses - 1; i >= 0; i-- {
		mes := mesRef.AddDate(0, -i, 0)
		mesStr := mes.Format("2006-01")
		label := fmt.Sprintf("%s/%02d", mesesPt[int(mes.Month())], mes.Year()%100)

		var receitas float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor), 0) FROM receitas
			WHERE family_id=$2 AND (recorrente = true OR date_trunc('month', data) = $1::date)`, mes, fid).Scan(&receitas)

		var despesasFixas float64
		db.QueryRow(`SELECT COALESCE(SUM(valor), 0) FROM despesas_fixas WHERE ativa = true AND family_id=$1`, fid).Scan(&despesasFixas)

		despesasParc, err := TotalDeParcelasNoMes(db, fid, mes)
		if err != nil {
			return nil, err
		}

		var invest float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor), 0) FROM investimentos
			WHERE date_trunc('month', data) = $1::date AND family_id=$2`, mes, fid).Scan(&invest)

		despesas := despesasFixas + despesasParc
		resumos = append(resumos, models.MesResumo{
			Mes:       mesStr,
			MesLabel:  label,
			Receitas:  receitas,
			Despesas:  despesas,
			Sobra:     receitas - despesas,
			Investido: invest,
		})
	}
	return resumos, nil
}
