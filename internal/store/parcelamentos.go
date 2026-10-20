package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"fmt"
	"math"
	"time"
)

func ToggleParcelamentoMesPago(db *sql.DB, fid, parcID int, mes time.Time) (bool, error) {
	var ok bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM parcelamentos WHERE id=$1 AND family_id=$2)`, parcID, fid).Scan(&ok)
	if !ok {
		return false, fmt.Errorf("parcelamento não pertence à família")
	}
	_, err := db.Exec(`
		INSERT INTO parcelamentos_mes (parcelamento_id, mes, pago) VALUES ($1,$2,false)
		ON CONFLICT (parcelamento_id, mes) DO NOTHING`, parcID, mes)
	if err != nil {
		return false, err
	}
	var pago bool
	err = db.QueryRow(`
		UPDATE parcelamentos_mes SET pago = NOT pago, pago_em = CASE WHEN NOT pago THEN NOW() ELSE NULL END
		WHERE parcelamento_id=$1 AND mes=$2
		RETURNING pago`, parcID, mes).Scan(&pago)
	return pago, err
}

func GetParcelamentosVigentesNoMes(db *sql.DB, fid int, mes time.Time) ([]models.Parcelamento, error) {
	mesStr := mes.Format("2006-01")
	rows, err := db.Query(`
		SELECT p.id, p.descricao, p.cartao, p.valor_parcela, p.total_parcelas, p.data_inicio, p.ativo,
		       p.parcela_atual
		     + (EXTRACT(YEAR  FROM $1::date) - EXTRACT(YEAR  FROM p.data_inicio)) * 12
		     + (EXTRACT(MONTH FROM $1::date) - EXTRACT(MONTH FROM p.data_inicio)) AS parcela_mes,
		       COALESCE(pm.pago, false) AS pago,
		       p.financiamento, p.valor_original, p.taxa_juros, p.total_economizado,
		       COALESCE(pm.antecipada, false), pm.valor_pago
		FROM parcelamentos p
		LEFT JOIN parcelamentos_mes pm ON pm.parcelamento_id = p.id AND pm.mes = $1
		WHERE p.ativo = true
		  AND p.family_id = $2
		  AND p.data_inicio <= $1::date
		  AND p.parcela_atual
		    + (EXTRACT(YEAR  FROM $1::date) - EXTRACT(YEAR  FROM p.data_inicio)) * 12
		    + (EXTRACT(MONTH FROM $1::date) - EXTRACT(MONTH FROM p.data_inicio)) <= p.total_parcelas
		ORDER BY p.descricao`, mes, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Parcelamento
	for rows.Next() {
		var p models.Parcelamento
		var parcelaMes int
		rows.Scan(&p.ID, &p.Descricao, &p.Cartao, &p.ValorParcela,
			&p.TotalParcelas, &p.DataInicio, &p.Ativo, &parcelaMes, &p.Pago,
			&p.Financiamento, &p.ValorOriginal, &p.TaxaJuros, &p.TotalEconomizado,
			&p.Antecipada, &p.ValorPago)
		p.ParcelaAtual = parcelaMes
		p.Mes = mesStr
		p.Restantes = p.TotalParcelas - parcelaMes
		if p.Restantes < 0 {
			p.Restantes = 0
		}
		p.ValorRestante = float64(p.Restantes+1) * p.ValorParcela
		if p.Financiamento && p.ValorOriginal > 0 {
			p.TotalJuros = (p.ValorParcela * float64(p.TotalParcelas)) - p.ValorOriginal
		}
		list = append(list, p)
	}
	return list, nil
}

func AutoFinalizarParcelamentos(db *sql.DB, fid int) error {
	_, err := db.Exec(`
		UPDATE parcelamentos SET ativo = false
		WHERE ativo = true AND family_id = $1
		  AND parcela_atual
		    + (EXTRACT(YEAR  FROM NOW()) - EXTRACT(YEAR  FROM data_inicio)) * 12
		    + (EXTRACT(MONTH FROM NOW()) - EXTRACT(MONTH FROM data_inicio)) > total_parcelas`, fid)
	return err
}

func GetParcelamentos(db *sql.DB, fid int, apenasAtivos bool) ([]models.Parcelamento, error) {
	q := `SELECT id, descricao, cartao, valor_parcela, parcela_atual, total_parcelas, data_inicio, ativo,
	       LEAST(parcela_atual
	         + (EXTRACT(YEAR FROM NOW()) - EXTRACT(YEAR FROM data_inicio))::int * 12
	         + (EXTRACT(MONTH FROM NOW()) - EXTRACT(MONTH FROM data_inicio))::int,
	         total_parcelas) AS parcela_hoje,
	       financiamento, valor_original, taxa_juros, total_economizado
	      FROM parcelamentos WHERE family_id = $1`
	if apenasAtivos {
		q += ` AND ativo = true`
	}
	q += ` ORDER BY ativo DESC, criado_em DESC`

	rows, err := db.Query(q, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Parcelamento
	for rows.Next() {
		var p models.Parcelamento
		var parcelaHoje int
		rows.Scan(&p.ID, &p.Descricao, &p.Cartao, &p.ValorParcela,
			&p.ParcelaAtual, &p.TotalParcelas, &p.DataInicio, &p.Ativo, &parcelaHoje,
			&p.Financiamento, &p.ValorOriginal, &p.TaxaJuros, &p.TotalEconomizado)
		p.ParcelaAtual = parcelaHoje
		p.Restantes = p.TotalParcelas - p.ParcelaAtual
		if p.Restantes < 0 {
			p.Restantes = 0
		}
		p.ValorRestante = float64(p.Restantes+1) * p.ValorParcela
		if !p.Ativo {
			p.ValorRestante = 0
		}
		if p.Financiamento && p.ValorOriginal > 0 {
			p.TotalJuros = (p.ValorParcela * float64(p.TotalParcelas)) - p.ValorOriginal
		}
		list = append(list, p)
	}
	return list, nil
}

func CreateParcelamento(db *sql.DB, fid int, descricao, cartao string, valorParcela float64, parcelaAtual, totalParcelas int, dataInicio time.Time, financiamento bool, valorOriginal, taxaJuros float64) error {
	_, err := db.Exec(`
		INSERT INTO parcelamentos (family_id, descricao, cartao, valor_parcela, parcela_atual, total_parcelas, data_inicio, financiamento, valor_original, taxa_juros)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		fid, descricao, cartao, valorParcela, parcelaAtual, totalParcelas, dataInicio, financiamento, valorOriginal, taxaJuros)
	return err
}

func UpdateParcelamento(db *sql.DB, fid, id, parcelaAtual int, ativo bool) error {
	_, err := db.Exec(`UPDATE parcelamentos SET parcela_atual=$1, ativo=$2 WHERE id=$3 AND family_id=$4`,
		parcelaAtual, ativo, id, fid)
	return err
}

func DeleteParcelamento(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM parcelamentos WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

// DescontoRacional calcula o desconto racional composto (padrão brasileiro)
// para pagamento antecipado de uma parcela.
func DescontoRacional(valorParcela, taxaJurosMensal float64, mesesAntecipados int) (valorComDesconto, economia float64) {
	if taxaJurosMensal <= 0 || mesesAntecipados <= 0 {
		return valorParcela, 0
	}
	taxa := taxaJurosMensal / 100.0
	fator := math.Pow(1+taxa, float64(mesesAntecipados))
	valorComDesconto = math.Round(valorParcela/fator*100) / 100
	economia = math.Round((valorParcela-valorComDesconto)*100) / 100
	return
}

// GetAntecipacoesPreview retorna as parcelas disponíveis para antecipação com desconto calculado.
func GetAntecipacoesPreview(db *sql.DB, fid, parcID int) ([]models.AntecipacaoPreview, error) {
	var financiamento bool
	var valorParcela, taxaJuros float64
	var totalParcelas, parcelaAtual int
	var dataInicio time.Time
	err := db.QueryRow(`
		SELECT financiamento, valor_parcela, taxa_juros, total_parcelas, parcela_atual, data_inicio
		FROM parcelamentos WHERE id=$1 AND family_id=$2`, parcID, fid).
		Scan(&financiamento, &valorParcela, &taxaJuros, &totalParcelas, &parcelaAtual, &dataInicio)
	if err != nil {
		return nil, err
	}
	if !financiamento {
		return nil, fmt.Errorf("parcelamento não é financiamento")
	}

	// Calcular parcela atual baseada no mês corrente
	now := time.Now()
	parcelaHoje := parcelaAtual +
		(now.Year()-dataInicio.Year())*12 +
		int(now.Month()-dataInicio.Month())
	if parcelaHoje > totalParcelas {
		parcelaHoje = totalParcelas
	}

	// Buscar parcelas já pagas/antecipadas
	pagas := map[int]bool{}
	rows, err := db.Query(`
		SELECT EXTRACT(YEAR FROM mes)::int * 12 + EXTRACT(MONTH FROM mes)::int
		FROM parcelamentos_mes WHERE parcelamento_id=$1 AND (pago=true OR antecipada=true)`, parcID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var m int
			rows.Scan(&m)
			pagas[m] = true
		}
	}

	var previews []models.AntecipacaoPreview
	for i := parcelaHoje + 1; i <= totalParcelas; i++ {
		// Verificar se já foi paga
		mesParcela := dataInicio.AddDate(0, i-parcelaAtual, 0)
		chave := mesParcela.Year()*12 + int(mesParcela.Month())
		if pagas[chave] {
			continue
		}
		mesesAntecipados := i - parcelaHoje
		valorDesc, desconto := DescontoRacional(valorParcela, taxaJuros, mesesAntecipados)
		previews = append(previews, models.AntecipacaoPreview{
			ParcelaNum:       i,
			MesesAntecipados: mesesAntecipados,
			ValorOriginal:    valorParcela,
			Desconto:         desconto,
			ValorComDesconto: valorDesc,
		})
	}
	return previews, nil
}

// AnteciparParcelas executa a antecipação de parcelas com desconto de juros.
func AnteciparParcelas(db *sql.DB, fid, parcID, qtdParcelas int) (float64, error) {
	previews, err := GetAntecipacoesPreview(db, fid, parcID)
	if err != nil {
		return 0, err
	}
	if qtdParcelas > len(previews) {
		qtdParcelas = len(previews)
	}
	if qtdParcelas <= 0 {
		return 0, fmt.Errorf("nenhuma parcela disponível para antecipação")
	}

	var parcelaAtual int
	var dataInicio time.Time
	db.QueryRow(`SELECT parcela_atual, data_inicio FROM parcelamentos WHERE id=$1`, parcID).
		Scan(&parcelaAtual, &dataInicio)

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	var totalEconomia float64
	for i := 0; i < qtdParcelas; i++ {
		p := previews[i]
		mesParcela := dataInicio.AddDate(0, p.ParcelaNum-parcelaAtual, 0)
		mesParcela = time.Date(mesParcela.Year(), mesParcela.Month(), 1, 0, 0, 0, 0, time.UTC)

		_, err := tx.Exec(`
			INSERT INTO parcelamentos_mes (parcelamento_id, mes, pago, pago_em, antecipada, valor_pago)
			VALUES ($1, $2, true, NOW(), true, $3)
			ON CONFLICT (parcelamento_id, mes)
			DO UPDATE SET pago = true, pago_em = NOW(), antecipada = true, valor_pago = $3`,
			parcID, mesParcela, p.ValorComDesconto)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		totalEconomia += p.Desconto
	}

	// Atualizar total_economizado
	_, err = tx.Exec(`
		UPDATE parcelamentos SET total_economizado = total_economizado + $1 WHERE id = $2`,
		totalEconomia, parcID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Verificar se todas as parcelas foram pagas → finalizar
	var totalParcelas, totalPagas int
	tx.QueryRow(`SELECT total_parcelas FROM parcelamentos WHERE id=$1`, parcID).Scan(&totalParcelas)
	tx.QueryRow(`SELECT COUNT(*) FROM parcelamentos_mes WHERE parcelamento_id=$1 AND (pago=true OR antecipada=true)`, parcID).Scan(&totalPagas)

	// Contar parcela atual (mês corrente) como potencialmente paga
	now := time.Now()
	parcelaHoje := parcelaAtual +
		(now.Year()-dataInicio.Year())*12 +
		int(now.Month()-dataInicio.Month())
	parcelasRestantesSemPagar := totalParcelas - parcelaHoje - totalPagas
	if totalPagas >= totalParcelas-parcelaHoje {
		// Todas as futuras foram antecipadas
		if parcelasRestantesSemPagar <= 0 {
			tx.Exec(`UPDATE parcelamentos SET ativo = false WHERE id = $1`, parcID)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return math.Round(totalEconomia*100) / 100, nil
}
