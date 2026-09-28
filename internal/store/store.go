package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"fincontrol/internal/models"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func NewDB() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		pass := getEnv("DB_PASS", "postgres")
		name := getEnv("DB_NAME", "fincontrol")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, pass, name)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("db open:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("db ping:", err)
	}
	return db
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- Categorias ---

func GetCategorias(db *sql.DB, fid int) ([]models.Categoria, error) {
	rows, err := db.Query(`
		SELECT id, nome, grupo, cor, ativo, criado_em
		FROM categorias WHERE family_id=$1 ORDER BY grupo, nome`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Categoria
	for rows.Next() {
		var c models.Categoria
		rows.Scan(&c.ID, &c.Nome, &c.Grupo, &c.Cor, &c.Ativo, &c.CriadoEm)
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

// --- Cartões ---

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

// --- Despesas Fixas ---

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

// --- Parcelamentos ---

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
		       COALESCE(pm.pago, false) AS pago
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
			&p.TotalParcelas, &p.DataInicio, &p.Ativo, &parcelaMes, &p.Pago)
		p.ParcelaAtual = parcelaMes
		p.Mes = mesStr
		p.Restantes = p.TotalParcelas - parcelaMes
		if p.Restantes < 0 {
			p.Restantes = 0
		}
		p.ValorRestante = float64(p.Restantes+1) * p.ValorParcela
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
	         total_parcelas) AS parcela_hoje
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
			&p.ParcelaAtual, &p.TotalParcelas, &p.DataInicio, &p.Ativo, &parcelaHoje)
		p.ParcelaAtual = parcelaHoje
		p.Restantes = p.TotalParcelas - p.ParcelaAtual
		if p.Restantes < 0 {
			p.Restantes = 0
		}
		p.ValorRestante = float64(p.Restantes+1) * p.ValorParcela
		if !p.Ativo {
			p.ValorRestante = 0
		}
		list = append(list, p)
	}
	return list, nil
}

func CreateParcelamento(db *sql.DB, fid int, descricao, cartao string, valorParcela float64, parcelaAtual, totalParcelas int, dataInicio time.Time) error {
	_, err := db.Exec(`
		INSERT INTO parcelamentos (family_id, descricao, cartao, valor_parcela, parcela_atual, total_parcelas, data_inicio)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		fid, descricao, cartao, valorParcela, parcelaAtual, totalParcelas, dataInicio)
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

// --- Receitas ---

func GetReceitas(db *sql.DB, fid int) ([]models.Receita, error) {
	rows, err := db.Query(`
		SELECT id, descricao, valor, data, tipo, recorrente, criado_em
		FROM receitas WHERE family_id=$1 ORDER BY data DESC, criado_em DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Receita
	for rows.Next() {
		var r models.Receita
		rows.Scan(&r.ID, &r.Descricao, &r.Valor, &r.Data, &r.Tipo, &r.Recorrente, &r.CriadoEm)
		list = append(list, r)
	}
	return list, nil
}

func GetReceitasMes(db *sql.DB, fid int, mes time.Time) (float64, error) {
	var total float64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(valor), 0)
		FROM receitas
		WHERE family_id = $2 AND (recorrente = true OR date_trunc('month', data) = $1::date)`, mes, fid).Scan(&total)
	return total, err
}

func CreateReceita(db *sql.DB, fid int, descricao string, valor float64, data time.Time, tipo string, recorrente bool) error {
	_, err := db.Exec(`INSERT INTO receitas (family_id, descricao, valor, data, tipo, recorrente) VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, descricao, valor, data, tipo, recorrente)
	return err
}

func UpdateReceita(db *sql.DB, fid, id int, descricao string, valor float64, data time.Time, tipo string, recorrente bool) error {
	_, err := db.Exec(`UPDATE receitas SET descricao=$1, valor=$2, data=$3, tipo=$4, recorrente=$5 WHERE id=$6 AND family_id=$7`,
		descricao, valor, data, tipo, recorrente, id, fid)
	return err
}

func DeleteReceita(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM receitas WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

// --- Histórico ---

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

		var despesasParc float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor_parcela), 0) FROM parcelamentos
			WHERE ativo = true AND family_id=$2
			  AND data_inicio <= $1::date
			  AND (EXTRACT(YEAR  FROM $1::date) - EXTRACT(YEAR  FROM data_inicio)) * 12
			    + (EXTRACT(MONTH FROM $1::date) - EXTRACT(MONTH FROM data_inicio)) + 1 <= total_parcelas`, mes, fid).Scan(&despesasParc)

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

// --- Investimentos ---

func GetInvestimentos(db *sql.DB, fid int) ([]models.Investimento, error) {
	rows, err := db.Query(`
		SELECT id, instituicao, tipo, valor, data, notas, criado_em
		FROM investimentos WHERE family_id=$1 ORDER BY data DESC, criado_em DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Investimento
	for rows.Next() {
		var i models.Investimento
		rows.Scan(&i.ID, &i.Instituicao, &i.Tipo, &i.Valor, &i.Data, &i.Notas, &i.CriadoEm)
		list = append(list, i)
	}
	return list, nil
}

func GetInvestimentosTotais(db *sql.DB, fid int) (map[string]float64, float64, error) {
	rows, err := db.Query(`
		SELECT instituicao, SUM(valor) FROM investimentos WHERE family_id=$1 GROUP BY instituicao ORDER BY instituicao`, fid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	totais := map[string]float64{}
	var total float64
	for rows.Next() {
		var inst string
		var v float64
		rows.Scan(&inst, &v)
		totais[inst] = v
		total += v
	}
	return totais, total, nil
}

func CreateInvestimento(db *sql.DB, fid int, instituicao, tipo string, valor float64, data time.Time, notas string) error {
	_, err := db.Exec(`INSERT INTO investimentos (family_id, instituicao, tipo, valor, data, notas) VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, instituicao, tipo, valor, data, notas)
	return err
}

func UpdateInvestimento(db *sql.DB, fid, id int, instituicao, tipo string, valor float64, data time.Time, notas string) error {
	_, err := db.Exec(`UPDATE investimentos SET instituicao=$1, tipo=$2, valor=$3, data=$4, notas=$5 WHERE id=$6 AND family_id=$7`,
		instituicao, tipo, valor, data, notas, id, fid)
	return err
}

func DeleteInvestimento(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM investimentos WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

// --- Reserva de Emergência ---

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

func GetInvestidoNoMes(db *sql.DB, fid int, mes time.Time) float64 {
	var totalInv, totalReserva float64
	db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM investimentos WHERE family_id=$1 AND date_trunc('month', data) = date_trunc('month', $2::date)`, fid, mes).Scan(&totalInv)
	db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM reserva_emergencia WHERE family_id=$1 AND tipo='deposito' AND date_trunc('month', data) = date_trunc('month', $2::date)`, fid, mes).Scan(&totalReserva)
	return totalInv + totalReserva
}

// --- Empréstimos ---

func GetEmprestimos(db *sql.DB, fid int) ([]models.Emprestimo, error) {
	rows, err := db.Query(`
		SELECT e.id, e.pessoa, e.valor, e.direcao, e.data, e.pago, e.notas, e.criado_em,
		       COALESCE(SUM(ep.valor), 0) AS valor_pago
		FROM emprestimos e
		LEFT JOIN emprestimo_pagamentos ep ON ep.emprestimo_id = e.id
		WHERE e.family_id = $1
		GROUP BY e.id, e.pessoa, e.valor, e.direcao, e.data, e.pago, e.notas, e.criado_em
		ORDER BY e.pago ASC, e.data DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Emprestimo
	for rows.Next() {
		var e models.Emprestimo
		rows.Scan(&e.ID, &e.Pessoa, &e.Valor, &e.Direcao, &e.Data, &e.Pago, &e.Notas, &e.CriadoEm, &e.ValorPago)
		e.ValorRestante = e.Valor - e.ValorPago
		if e.ValorRestante < 0 {
			e.ValorRestante = 0
		}
		list = append(list, e)
	}
	return list, nil
}

func AddPagamentoEmprestimo(db *sql.DB, fid, empID int, valor float64, data time.Time, notas string) error {
	var ok bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM emprestimos WHERE id=$1 AND family_id=$2)`, empID, fid).Scan(&ok)
	if !ok {
		return fmt.Errorf("empréstimo não encontrado")
	}
	_, err := db.Exec(`INSERT INTO emprestimo_pagamentos (emprestimo_id, family_id, valor, data, notas) VALUES ($1,$2,$3,$4,$5)`,
		empID, fid, valor, data, notas)
	if err != nil {
		return err
	}
	var valorOrig, totalPago float64
	db.QueryRow(`SELECT valor FROM emprestimos WHERE id=$1`, empID).Scan(&valorOrig)
	db.QueryRow(`SELECT COALESCE(SUM(valor),0) FROM emprestimo_pagamentos WHERE emprestimo_id=$1`, empID).Scan(&totalPago)
	if totalPago >= valorOrig {
		db.Exec(`UPDATE emprestimos SET pago = true, pago_em = NOW() WHERE id=$1 AND family_id=$2 AND NOT pago`, empID, fid)
	}
	return nil
}

func CreateEmprestimo(db *sql.DB, fid int, pessoa string, valor float64, direcao string, data time.Time, notas string) error {
	_, err := db.Exec(`INSERT INTO emprestimos (family_id, pessoa, valor, direcao, data, notas) VALUES ($1,$2,$3,$4,$5,$6)`,
		fid, pessoa, valor, direcao, data, notas)
	return err
}

func DeleteEmprestimo(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`DELETE FROM emprestimos WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func ToggleEmprestimoPago(db *sql.DB, fid, id int) (bool, error) {
	var pago bool
	err := db.QueryRow(`
		UPDATE emprestimos SET pago = NOT pago, pago_em = CASE WHEN NOT pago THEN NOW() ELSE NULL END
		WHERE id=$1 AND family_id=$2 RETURNING pago`, id, fid).Scan(&pago)
	return pago, err
}

// --- Autenticação ---

func HashSenha(senha string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	return string(b), err
}

func GetUserByEmail(db *sql.DB, email string) (*models.User, string, error) {
	var u models.User
	var hash string
	err := db.QueryRow(`
		SELECT u.id, u.nome, u.email, u.senha_hash, u.admin, u.ativo, u.family_id, COALESCE(f.nome,''), u.family_admin, u.criado_em
		FROM users u LEFT JOIN families f ON f.id = u.family_id
		WHERE u.email = $1`, email).
		Scan(&u.ID, &u.Nome, &u.Email, &hash, &u.Admin, &u.Ativo, &u.FamilyID, &u.FamilyNome, &u.FamilyAdmin, &u.CriadoEm)
	return &u, hash, err
}

func GetUserBySession(db *sql.DB, token string) (*models.User, error) {
	var u models.User
	err := db.QueryRow(`
		SELECT u.id, u.nome, u.email, u.admin, u.ativo, u.family_id, COALESCE(f.nome,''), u.family_admin, u.criado_em
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		LEFT JOIN families f ON f.id = u.family_id
		WHERE s.token = $1 AND s.expira_em > NOW() AND u.ativo = true`,
		token).Scan(&u.ID, &u.Nome, &u.Email, &u.Admin, &u.Ativo, &u.FamilyID, &u.FamilyNome, &u.FamilyAdmin, &u.CriadoEm)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateSession(db *sql.DB, userID int, token string, expiry time.Time) error {
	_, err := db.Exec(`INSERT INTO sessions (token, user_id, expira_em) VALUES ($1, $2, $3)`,
		token, userID, expiry)
	return err
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// --- Gestão de Usuários ---

func GetUsers(db *sql.DB) ([]models.User, error) {
	rows, err := db.Query(`
		SELECT u.id, u.nome, u.email, u.admin, u.ativo, u.family_id, COALESCE(f.nome,''), u.family_admin, u.criado_em
		FROM users u LEFT JOIN families f ON f.id = u.family_id
		ORDER BY u.criado_em`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Nome, &u.Email, &u.Admin, &u.Ativo, &u.FamilyID, &u.FamilyNome, &u.FamilyAdmin, &u.CriadoEm)
		list = append(list, u)
	}
	return list, nil
}

func GetFamilyMembers(db *sql.DB, familyID int) ([]models.User, error) {
	rows, err := db.Query(`
		SELECT u.id, u.nome, u.email, u.admin, u.ativo, u.family_id, COALESCE(f.nome,''), u.family_admin, u.criado_em
		FROM users u LEFT JOIN families f ON f.id = u.family_id
		WHERE u.family_id = $1
		ORDER BY u.criado_em`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Nome, &u.Email, &u.Admin, &u.Ativo, &u.FamilyID, &u.FamilyNome, &u.FamilyAdmin, &u.CriadoEm)
		list = append(list, u)
	}
	return list, nil
}

func CreateUser(db *sql.DB, nome, email, senhaHash string, admin bool, familyID int) error {
	_, err := db.Exec(`INSERT INTO users (nome, email, senha_hash, admin, family_id) VALUES ($1,$2,$3,$4,$5)`,
		nome, email, senhaHash, admin, familyID)
	return err
}

func SetUserFamily(db *sql.DB, userID, familyID int) error {
	_, err := db.Exec(`UPDATE users SET family_id=$1 WHERE id=$2`, familyID, userID)
	return err
}

func ToggleUserAdmin(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE users SET admin = NOT admin WHERE id = $1`, id)
	return err
}

func ToggleFamilyAdmin(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE users SET family_admin = NOT family_admin WHERE id = $1`, id)
	return err
}

func ToggleUserAtivo(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE users SET ativo = NOT ativo WHERE id = $1`, id)
	return err
}

func ResetSenha(db *sql.DB, id int, hash string) error {
	_, err := db.Exec(`UPDATE users SET senha_hash=$1 WHERE id=$2`, hash, id)
	return err
}

func DeleteUser(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM users WHERE id=$1`, id)
	return err
}

// --- Famílias ---

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

// --- Permissões de tela ---

func GetBlockedScreens(db *sql.DB, userID int) ([]string, error) {
	rows, err := db.Query(`SELECT tela FROM user_permissions WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var telas []string
	for rows.Next() {
		var t string
		rows.Scan(&t)
		telas = append(telas, t)
	}
	return telas, nil
}

func ToggleScreenPermission(db *sql.DB, userID int, tela string) (bool, error) {
	var exists bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_permissions WHERE user_id=$1 AND tela=$2)`, userID, tela).Scan(&exists)
	if exists {
		_, err := db.Exec(`DELETE FROM user_permissions WHERE user_id=$1 AND tela=$2`, userID, tela)
		return false, err
	}
	_, err := db.Exec(`INSERT INTO user_permissions (user_id, tela) VALUES ($1,$2) ON CONFLICT DO NOTHING`, userID, tela)
	return true, err
}

// --- Transações Bancárias ---

func CreateTransacaoBancoBatch(db *sql.DB, fid int, txns []models.TransacaoBanco) (novos int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, t := range txns {
		if t.FitID != "" {
			var exists bool
			tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM transacoes_banco WHERE family_id=$1 AND fit_id=$2)`,
				fid, t.FitID).Scan(&exists)
			if exists {
				continue
			}
		}
		_, err := tx.Exec(`INSERT INTO transacoes_banco
			(family_id, data, descricao, valor, tipo, categoria, origem, banco, fit_id, status, pluggy_item_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			fid, t.Data, t.Descricao, t.Valor, t.Tipo, t.Categoria, t.Origem, t.Banco, t.FitID, "pendente", t.PluggyItemID)
		if err != nil {
			return novos, fmt.Errorf("insert transacao: %w", err)
		}
		novos++
	}
	return novos, tx.Commit()
}

func GetTransacoesBanco(db *sql.DB, fid int, mes, origem, status, banco string) ([]models.TransacaoBanco, error) {
	q := `SELECT id, data, descricao, valor, tipo, categoria, origem, banco, fit_id, status,
	             despesa_id, receita_id, pluggy_item_id, criado_em
	      FROM transacoes_banco WHERE family_id=$1`
	args := []any{fid}
	n := 2
	if mes != "" {
		q += fmt.Sprintf(` AND TO_CHAR(data, 'YYYY-MM') = $%d`, n)
		args = append(args, mes)
		n++
	}
	if origem != "" {
		q += fmt.Sprintf(` AND origem = $%d`, n)
		args = append(args, origem)
		n++
	}
	if status != "" {
		q += fmt.Sprintf(` AND status = $%d`, n)
		args = append(args, status)
		n++
	}
	if banco != "" {
		q += fmt.Sprintf(` AND banco = $%d`, n)
		args = append(args, banco)
		n++
	}
	q += ` ORDER BY data DESC, id DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.TransacaoBanco
	for rows.Next() {
		var t models.TransacaoBanco
		rows.Scan(&t.ID, &t.Data, &t.Descricao, &t.Valor, &t.Tipo, &t.Categoria,
			&t.Origem, &t.Banco, &t.FitID, &t.Status,
			&t.DespesaID, &t.ReceitaID, &t.PluggyItemID, &t.CriadoEm)
		list = append(list, t)
	}
	return list, nil
}

func ConverterTransacaoEmDespesa(db *sql.DB, fid, id int, categoria string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var t models.TransacaoBanco
	err = tx.QueryRow(`SELECT descricao, valor FROM transacoes_banco WHERE id=$1 AND family_id=$2`, id, fid).
		Scan(&t.Descricao, &t.Valor)
	if err != nil {
		return fmt.Errorf("transacao não encontrada: %w", err)
	}

	var despesaID int
	err = tx.QueryRow(`INSERT INTO despesas_fixas (family_id, nome, valor, categoria)
		VALUES ($1,$2,$3,$4) RETURNING id`, fid, t.Descricao, abs(t.Valor), categoria).Scan(&despesaID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE transacoes_banco SET status='convertida', despesa_id=$1, categoria=$2
		WHERE id=$3 AND family_id=$4`, despesaID, categoria, id, fid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func ConverterTransacaoEmReceita(db *sql.DB, fid, id int, tipo string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var descricao string
	var valor float64
	var data time.Time
	err = tx.QueryRow(`SELECT descricao, valor, data FROM transacoes_banco WHERE id=$1 AND family_id=$2`, id, fid).
		Scan(&descricao, &valor, &data)
	if err != nil {
		return fmt.Errorf("transacao não encontrada: %w", err)
	}

	var receitaID int
	err = tx.QueryRow(`INSERT INTO receitas (family_id, descricao, valor, data, tipo, recorrente)
		VALUES ($1,$2,$3,$4,$5,false) RETURNING id`, fid, descricao, abs(valor), data, tipo).Scan(&receitaID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE transacoes_banco SET status='convertida', receita_id=$1
		WHERE id=$2 AND family_id=$3`, receitaID, id, fid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func IgnorarTransacao(db *sql.DB, fid, id int) error {
	_, err := db.Exec(`UPDATE transacoes_banco SET status='ignorada' WHERE id=$1 AND family_id=$2`, id, fid)
	return err
}

func CategorizarTransacao(db *sql.DB, fid, id int, categoria string) error {
	_, err := db.Exec(`UPDATE transacoes_banco SET categoria=$1, status='categorizada' WHERE id=$2 AND family_id=$3`,
		categoria, id, fid)
	return err
}

func GetPluggyItems(db *sql.DB, fid int) ([]models.PluggyItem, error) {
	rows, err := db.Query(`SELECT id, family_id, item_id, connector_name, status, last_sync, criado_em
		FROM pluggy_items WHERE family_id=$1 ORDER BY criado_em DESC`, fid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.PluggyItem
	for rows.Next() {
		var p models.PluggyItem
		rows.Scan(&p.ID, &p.FamilyID, &p.ItemID, &p.ConnectorName, &p.Status, &p.LastSync, &p.CriadoEm)
		list = append(list, p)
	}
	return list, nil
}

func CreatePluggyItem(db *sql.DB, fid int, itemID, connectorName string) error {
	_, err := db.Exec(`INSERT INTO pluggy_items (family_id, item_id, connector_name)
		VALUES ($1,$2,$3) ON CONFLICT (item_id) DO NOTHING`, fid, itemID, connectorName)
	return err
}

func UpdatePluggyItemSync(db *sql.DB, itemID string) error {
	_, err := db.Exec(`UPDATE pluggy_items SET last_sync=NOW() WHERE item_id=$1`, itemID)
	return err
}

func DeletePluggyItem(db *sql.DB, fid int, itemID string) error {
	_, err := db.Exec(`DELETE FROM pluggy_items WHERE family_id=$1 AND item_id=$2`, fid, itemID)
	return err
}

// --- Integracoes Config ---

func GetIntegracaoConfig(db *sql.DB, integracao string) (*models.IntegracaoConfig, error) {
	var c models.IntegracaoConfig
	err := db.QueryRow(`SELECT id, integracao, client_id, client_secret, service_url, ativo
		FROM integracoes_config WHERE integracao=$1`, integracao).
		Scan(&c.ID, &c.Integracao, &c.ClientID, &c.ClientSecret, &c.ServiceURL, &c.Ativo)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func SaveIntegracaoConfig(db *sql.DB, c models.IntegracaoConfig) error {
	_, err := db.Exec(`INSERT INTO integracoes_config (integracao, client_id, client_secret, service_url, ativo, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (integracao) DO UPDATE SET
			client_id=EXCLUDED.client_id,
			client_secret=EXCLUDED.client_secret,
			service_url=EXCLUDED.service_url,
			ativo=EXCLUDED.ativo,
			atualizado_em=NOW()`,
		c.Integracao, c.ClientID, c.ClientSecret, c.ServiceURL, c.Ativo)
	return err
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// --- Migrations ---

type migration struct {
	version int
	name    string
	stmts   []string
}

var migrations = []migration{
	{
		version: 1,
		name:    "core_tables",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS families (
				id        SERIAL PRIMARY KEY,
				nome      VARCHAR(120) UNIQUE NOT NULL,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS users (
				id         SERIAL PRIMARY KEY,
				nome       VARCHAR(100) NOT NULL,
				email      VARCHAR(150) UNIQUE NOT NULL,
				senha_hash TEXT NOT NULL,
				admin      BOOLEAN NOT NULL DEFAULT false,
				ativo      BOOLEAN NOT NULL DEFAULT true,
				family_id  INTEGER REFERENCES families(id),
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS sessions (
				token     TEXT PRIMARY KEY,
				user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				expira_em TIMESTAMPTZ NOT NULL,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
		},
	},
	{
		version: 2,
		name:    "aux_tables",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS categorias (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				nome      VARCHAR(60) NOT NULL,
				grupo     VARCHAR(20) NOT NULL DEFAULT 'basica',
				cor       VARCHAR(7)  NOT NULL DEFAULT '#607d8b',
				ativo     BOOLEAN NOT NULL DEFAULT true,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (family_id, nome)
			)`,
			`CREATE TABLE IF NOT EXISTS cartoes (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				nome      VARCHAR(60) NOT NULL,
				bandeira  VARCHAR(30) NOT NULL DEFAULT '',
				limite    NUMERIC(14,2) NOT NULL DEFAULT 0,
				cor       VARCHAR(7)  NOT NULL DEFAULT '#607d8b',
				ativo     BOOLEAN NOT NULL DEFAULT true,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (family_id, nome)
			)`,
		},
	},
	{
		version: 3,
		name:    "finance_tables",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS despesas_fixas (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				nome      VARCHAR(120) NOT NULL,
				valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
				categoria VARCHAR(60) NOT NULL DEFAULT '',
				ativa     BOOLEAN NOT NULL DEFAULT true,
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS despesas_fixas_mes (
				id         SERIAL PRIMARY KEY,
				despesa_id INTEGER NOT NULL REFERENCES despesas_fixas(id) ON DELETE CASCADE,
				mes        DATE NOT NULL,
				pago       BOOLEAN NOT NULL DEFAULT false,
				pago_em    TIMESTAMPTZ,
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (despesa_id, mes)
			)`,
			`CREATE TABLE IF NOT EXISTS parcelamentos (
				id             SERIAL PRIMARY KEY,
				family_id      INTEGER NOT NULL REFERENCES families(id),
				descricao      VARCHAR(120) NOT NULL,
				cartao         VARCHAR(60) NOT NULL DEFAULT '',
				valor_parcela  NUMERIC(14,2) NOT NULL DEFAULT 0,
				parcela_atual  INTEGER NOT NULL DEFAULT 1,
				total_parcelas INTEGER NOT NULL DEFAULT 1,
				data_inicio    DATE NOT NULL,
				ativo          BOOLEAN NOT NULL DEFAULT true,
				criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS receitas (
				id         SERIAL PRIMARY KEY,
				family_id  INTEGER NOT NULL REFERENCES families(id),
				descricao  VARCHAR(120) NOT NULL,
				valor      NUMERIC(14,2) NOT NULL DEFAULT 0,
				data       DATE NOT NULL,
				tipo       VARCHAR(60) NOT NULL DEFAULT '',
				recorrente BOOLEAN NOT NULL DEFAULT false,
				criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS investimentos (
				id          SERIAL PRIMARY KEY,
				family_id   INTEGER NOT NULL REFERENCES families(id),
				instituicao VARCHAR(100) NOT NULL,
				tipo        VARCHAR(60) NOT NULL DEFAULT '',
				valor       NUMERIC(14,2) NOT NULL DEFAULT 0,
				data        DATE NOT NULL,
				notas       TEXT NOT NULL DEFAULT '',
				criado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS reserva_emergencia (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
				data      DATE NOT NULL DEFAULT CURRENT_DATE,
				notas     TEXT NOT NULL DEFAULT '',
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS emprestimos (
				id        SERIAL PRIMARY KEY,
				family_id INTEGER NOT NULL REFERENCES families(id),
				pessoa    VARCHAR(100) NOT NULL,
				valor     NUMERIC(14,2) NOT NULL DEFAULT 0,
				direcao   VARCHAR(20) NOT NULL DEFAULT 'devo',
				data      DATE NOT NULL DEFAULT CURRENT_DATE,
				pago      BOOLEAN NOT NULL DEFAULT false,
				pago_em   TIMESTAMPTZ,
				notas     TEXT NOT NULL DEFAULT '',
				criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
		},
	},
	{
		version: 4,
		name:    "loans_and_permissions",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS emprestimo_pagamentos (
				id            SERIAL PRIMARY KEY,
				emprestimo_id INTEGER NOT NULL REFERENCES emprestimos(id) ON DELETE CASCADE,
				family_id     INTEGER NOT NULL REFERENCES families(id),
				valor         NUMERIC(14,2) NOT NULL DEFAULT 0,
				data          DATE NOT NULL DEFAULT CURRENT_DATE,
				notas         TEXT NOT NULL DEFAULT '',
				criado_em     TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE TABLE IF NOT EXISTS user_permissions (
				user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				tela      VARCHAR(50) NOT NULL,
				PRIMARY KEY (user_id, tela)
			)`,
		},
	},
	{
		version: 5,
		name:    "indexes",
		stmts: []string{
			`CREATE INDEX IF NOT EXISTS idx_categorias_family          ON categorias(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_cartoes_family             ON cartoes(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_despesas_fixas_family      ON despesas_fixas(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_parcelamentos_family       ON parcelamentos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_receitas_family            ON receitas(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_investimentos_family       ON investimentos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_reserva_emergencia_family  ON reserva_emergencia(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_emprestimos_family         ON emprestimos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_emp_pagamentos_emprestimo  ON emprestimo_pagamentos(emprestimo_id)`,
			`CREATE INDEX IF NOT EXISTS idx_emp_pagamentos_family      ON emprestimo_pagamentos(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_user_permissions_user      ON user_permissions(user_id)`,
		},
	},
	{
		version: 6,
		name:    "backcompat_columns",
		stmts: []string{
			`ALTER TABLE users              ADD COLUMN IF NOT EXISTS family_id    INTEGER REFERENCES families(id)`,
			`ALTER TABLE users              ADD COLUMN IF NOT EXISTS family_admin BOOLEAN NOT NULL DEFAULT false`,
			`ALTER TABLE despesas_fixas     ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE parcelamentos      ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE receitas           ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE investimentos      ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE reserva_emergencia ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE emprestimos        ADD COLUMN IF NOT EXISTS family_id   INTEGER REFERENCES families(id)`,
			`ALTER TABLE emprestimos        ADD COLUMN IF NOT EXISTS pago_em     TIMESTAMPTZ`,
		},
	},
	{
		version: 7,
		name:    "default_family_and_backfill",
		stmts: []string{
			`INSERT INTO families (nome) VALUES ('Família Padrão') ON CONFLICT (nome) DO NOTHING`,
			`UPDATE users              SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE despesas_fixas     SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE parcelamentos      SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE receitas           SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE investimentos      SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE reserva_emergencia SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`UPDATE emprestimos        SET family_id = (SELECT id FROM families WHERE nome='Família Padrão') WHERE family_id IS NULL`,
			`ALTER TABLE users              ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE despesas_fixas     ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE parcelamentos      ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE receitas           ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE investimentos      ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE reserva_emergencia ALTER COLUMN family_id SET NOT NULL`,
			`ALTER TABLE emprestimos        ALTER COLUMN family_id SET NOT NULL`,
		},
	},
	{
		version: 8,
		name:    "seed_default_categories",
		stmts: []string{
			`INSERT INTO categorias (family_id, nome, grupo, cor)
			 SELECT f.id, cat.nome, cat.grupo, cat.cor
			 FROM families f
			 CROSS JOIN (VALUES
			     ('Básicas',     'basica', '#3498db'),
			     ('Cartão',      'cartao', '#f39c12'),
			     ('VR',          'vr',     '#27ae60'),
			     ('Alimentação', 'basica', '#e74c3c'),
			     ('Transporte',  'basica', '#9b59b6'),
			     ('Saúde',       'basica', '#e91e63'),
			     ('Moradia',     'basica', '#795548'),
			     ('Lazer',       'basica', '#00bcd4'),
			     ('Educação',    'basica', '#3f51b5'),
			     ('Serviços',    'basica', '#607d8b')
			 ) AS cat(nome, grupo, cor)
			 ON CONFLICT (family_id, nome) DO NOTHING`,
		},
	},
	{
		version: 9,
		name:    "create_parcelamentos_mes",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS parcelamentos_mes (
				parcelamento_id INTEGER NOT NULL REFERENCES parcelamentos(id) ON DELETE CASCADE,
				mes             DATE    NOT NULL,
				pago            BOOLEAN NOT NULL DEFAULT false,
				pago_em         TIMESTAMPTZ,
				PRIMARY KEY (parcelamento_id, mes)
			)`,
		},
	},
	{
		version: 10,
		name:    "reserva_em_tipo",
		stmts: []string{
			`ALTER TABLE reserva_emergencia ADD COLUMN IF NOT EXISTS tipo VARCHAR(10) NOT NULL DEFAULT 'deposito'`,
		},
	},
	{
		version: 11,
		name:    "transacoes_banco_e_pluggy_items",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS transacoes_banco (
				id              SERIAL PRIMARY KEY,
				family_id       INTEGER NOT NULL REFERENCES families(id),
				data            DATE NOT NULL,
				descricao       TEXT NOT NULL,
				valor           NUMERIC(14,2) NOT NULL,
				tipo            VARCHAR(10) NOT NULL,
				categoria       VARCHAR(100) NOT NULL DEFAULT '',
				origem          VARCHAR(20) NOT NULL,
				banco           VARCHAR(100) NOT NULL DEFAULT '',
				fit_id          VARCHAR(255) NOT NULL DEFAULT '',
				status          VARCHAR(20) NOT NULL DEFAULT 'pendente',
				despesa_id      INTEGER REFERENCES despesas_fixas(id),
				receita_id      INTEGER REFERENCES receitas(id),
				pluggy_item_id  VARCHAR(255) NOT NULL DEFAULT '',
				criado_em       TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE INDEX IF NOT EXISTS idx_transacoes_banco_family ON transacoes_banco(family_id)`,
			`CREATE INDEX IF NOT EXISTS idx_transacoes_banco_fit_id ON transacoes_banco(family_id, fit_id)`,
			`CREATE TABLE IF NOT EXISTS pluggy_items (
				id              SERIAL PRIMARY KEY,
				family_id       INTEGER NOT NULL REFERENCES families(id),
				item_id         VARCHAR(255) NOT NULL UNIQUE,
				connector_name  VARCHAR(255) NOT NULL DEFAULT '',
				status          VARCHAR(50) NOT NULL DEFAULT 'active',
				last_sync       TIMESTAMPTZ,
				criado_em       TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`CREATE INDEX IF NOT EXISTS idx_pluggy_items_family ON pluggy_items(family_id)`,
		},
	},
	{
		version: 12,
		name:    "integracoes_config",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS integracoes_config (
				id              SERIAL PRIMARY KEY,
				integracao      VARCHAR(50) NOT NULL UNIQUE,
				client_id       TEXT NOT NULL DEFAULT '',
				client_secret   TEXT NOT NULL DEFAULT '',
				service_url     TEXT NOT NULL DEFAULT '',
				ativo           BOOLEAN NOT NULL DEFAULT false,
				atualizado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`,
			`INSERT INTO integracoes_config (integracao, service_url)
			 VALUES ('pluggy', 'http://pluggy-service:8081')
			 ON CONFLICT (integracao) DO NOTHING`,
		},
	},
}

// RunMigrations executa as migrations pendentes registrando cada uma em schema_migrations.
func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       VARCHAR(200) NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("criar schema_migrations: %w", err)
	}

	for _, m := range migrations {
		var applied bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, m.version).Scan(&applied)
		if applied {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d: begin: %w", m.version, err)
		}
		for _, stmt := range m.stmts {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.version, m.name); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: registrar versão: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d: commit: %w", m.version, err)
		}
		log.Printf("[migrate] v%d (%s) aplicada", m.version, m.name)
	}
	return nil
}

// EnsureAdmin cria o usuário admin padrão se não houver nenhum usuário.
func EnsureAdmin(db *sql.DB) error {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var familyID int
	if err := db.QueryRow(`SELECT id FROM families WHERE nome='Família Padrão'`).Scan(&familyID); err != nil {
		return fmt.Errorf("família padrão não encontrada: %w", err)
	}
	_, err = db.Exec(
		`INSERT INTO users (nome, email, senha_hash, admin, family_id) VALUES ('Admin', 'admin@finbertoldi.com', $1, true, $2)`,
		string(hash), familyID)
	if err == nil {
		log.Println("Usuário admin criado: admin@finbertoldi.com / admin123")
	}
	return err
}
