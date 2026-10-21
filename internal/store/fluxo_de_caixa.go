package store

import (
	"database/sql"
	"time"

	"fincontrol/internal/models"
)

// ProjecaoDoFluxoDeCaixa projeta os proximos meses a partir de primeiroMes,
// com as mesmas regras do dashboard: receitas recorrentes mais as unicas do
// mes, despesas fixas ativas e parcelas vigentes no mes (considerando a
// parcela atual). Parcela ja antecipada fica de fora: foi paga antes.
func ProjecaoDoFluxoDeCaixa(db *sql.DB, fid int, primeiroMes time.Time, quantidadeDeMeses int) ([]models.MesProjetado, error) {
	var despesasFixas float64
	if err := db.QueryRow(`SELECT COALESCE(SUM(valor), 0) FROM despesas_fixas WHERE ativa AND family_id = $1`, fid).
		Scan(&despesasFixas); err != nil {
		return nil, err
	}

	inicio := primeiroDiaDoMes(primeiroMes)
	meses := make([]models.MesProjetado, 0, quantidadeDeMeses)
	for i := 0; i < quantidadeDeMeses; i++ {
		mes := inicio.AddDate(0, i, 0)
		receitas, err := GetReceitasMes(db, fid, mes)
		if err != nil {
			return nil, err
		}
		parcelamentos, err := GetParcelamentosVigentesNoMes(db, fid, mes)
		if err != nil {
			return nil, err
		}
		var parcelas float64
		for _, parcelamento := range parcelamentos {
			if !parcelamento.Antecipada {
				parcelas += parcelamento.ValorParcela
			}
		}
		meses = append(meses, models.MesProjetado{Mes: mes, Receitas: receitas, DespesasFixas: despesasFixas, Parcelas: parcelas})
	}
	return models.MontarProjecao(meses), nil
}
