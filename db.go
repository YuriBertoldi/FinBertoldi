package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func newDB() *sql.DB {
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

// --- Despesas Fixas ---

func dbGetDespesasFixas(db *sql.DB) ([]DespesaFixa, error) {
	rows, err := db.Query(`
		SELECT id, nome, valor, categoria, ativa, criado_em
		FROM despesas_fixas ORDER BY categoria, nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DespesaFixa
	for rows.Next() {
		var d DespesaFixa
		rows.Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria, &d.Ativa, &d.CriadoEm)
		list = append(list, d)
	}
	return list, nil
}

func dbCreateDespesa(db *sql.DB, nome string, valor float64, categoria string) error {
	_, err := db.Exec(`INSERT INTO despesas_fixas (nome, valor, categoria) VALUES ($1,$2,$3)`,
		nome, valor, categoria)
	return err
}

func dbUpdateDespesa(db *sql.DB, id int, nome string, valor float64, categoria string) error {
	_, err := db.Exec(`UPDATE despesas_fixas SET nome=$1, valor=$2, categoria=$3 WHERE id=$4`,
		nome, valor, categoria, id)
	return err
}

func dbDeleteDespesa(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM despesas_fixas WHERE id=$1`, id)
	return err
}

func dbToggleDespesaAtiva(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE despesas_fixas SET ativa = NOT ativa WHERE id=$1`, id)
	return err
}

// EnsureDespesasMes garante registros mensais para todas as despesas ativas
func dbEnsureDespesasMes(db *sql.DB, mes time.Time) error {
	_, err := db.Exec(`
		INSERT INTO despesas_fixas_mes (despesa_id, mes)
		SELECT id, $1 FROM despesas_fixas WHERE ativa = true
		ON CONFLICT (despesa_id, mes) DO NOTHING`, mes)
	return err
}

// GetDespesasMes retorna despesas com status do mês
func dbGetDespesasMes(db *sql.DB, mes time.Time, mesStr string) ([]DespesaMes, []DespesaMes, []DespesaMes, error) {
	rows, err := db.Query(`
		SELECT df.id, df.nome, df.valor, df.categoria,
		       COALESCE(dfm.pago, false) AS pago
		FROM despesas_fixas df
		LEFT JOIN despesas_fixas_mes dfm
		       ON dfm.despesa_id = df.id AND dfm.mes = $1
		WHERE df.ativa = true
		ORDER BY df.categoria, df.nome`, mes)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	var basicas, cartao, vr []DespesaMes
	for rows.Next() {
		var d DespesaMes
		rows.Scan(&d.ID, &d.Nome, &d.Valor, &d.Categoria, &d.Pago)
		d.Mes = mesStr
		switch d.Categoria {
		case "Cartão":
			cartao = append(cartao, d)
		case "VR":
			vr = append(vr, d)
		default:
			basicas = append(basicas, d)
		}
	}
	return basicas, cartao, vr, nil
}

func dbToggleDespesaMesPago(db *sql.DB, despesaID int, mes time.Time) (bool, error) {
	// Upsert then toggle
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

// dbGetParcelamentosVigentesNoMes retorna parcelamentos cuja parcela calculada
// (data_inicio + meses decorridos) ainda está dentro do total_parcelas no mês.
// Atualiza ParcelaAtual com a parcela vigente naquele mês.
func dbGetParcelamentosVigentesNoMes(db *sql.DB, mes time.Time) ([]Parcelamento, error) {
	rows, err := db.Query(`
		SELECT id, descricao, cartao, valor_parcela, total_parcelas, data_inicio, ativo,
		       (EXTRACT(YEAR  FROM $1) - EXTRACT(YEAR  FROM data_inicio)) * 12
		     + (EXTRACT(MONTH FROM $1) - EXTRACT(MONTH FROM data_inicio)) + 1 AS parcela_mes
		FROM parcelamentos
		WHERE ativo = true
		  AND data_inicio <= $1
		  AND (EXTRACT(YEAR  FROM $1) - EXTRACT(YEAR  FROM data_inicio)) * 12
		    + (EXTRACT(MONTH FROM $1) - EXTRACT(MONTH FROM data_inicio)) + 1 <= total_parcelas
		ORDER BY descricao`, mes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Parcelamento
	for rows.Next() {
		var p Parcelamento
		var parcelaMes int
		rows.Scan(&p.ID, &p.Descricao, &p.Cartao, &p.ValorParcela,
			&p.TotalParcelas, &p.DataInicio, &p.Ativo, &parcelaMes)
		p.ParcelaAtual = parcelaMes
		p.Restantes = p.TotalParcelas - parcelaMes
		if p.Restantes < 0 {
			p.Restantes = 0
		}
		p.ValorRestante = float64(p.Restantes+1) * p.ValorParcela
		list = append(list, p)
	}
	return list, nil
}

// dbAutoFinalizarParcelamentos marca como ativo=false todos os parcelamentos
// que já passaram do total de parcelas (baseado na data corrente).
func dbAutoFinalizarParcelamentos(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE parcelamentos SET ativo = false
		WHERE ativo = true
		  AND (EXTRACT(YEAR  FROM NOW()) - EXTRACT(YEAR  FROM data_inicio)) * 12
		    + (EXTRACT(MONTH FROM NOW()) - EXTRACT(MONTH FROM data_inicio)) + 1 > total_parcelas`)
	return err
}

func dbGetParcelamentos(db *sql.DB, apenasAtivos bool) ([]Parcelamento, error) {
	q := `SELECT id, descricao, cartao, valor_parcela, parcela_atual, total_parcelas, data_inicio, ativo
	      FROM parcelamentos`
	if apenasAtivos {
		q += ` WHERE ativo = true`
	}
	q += ` ORDER BY ativo DESC, criado_em DESC`

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Parcelamento
	for rows.Next() {
		var p Parcelamento
		rows.Scan(&p.ID, &p.Descricao, &p.Cartao, &p.ValorParcela,
			&p.ParcelaAtual, &p.TotalParcelas, &p.DataInicio, &p.Ativo)
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

func dbCreateParcelamento(db *sql.DB, descricao, cartao string, valorParcela float64, parcelaAtual, totalParcelas int, dataInicio time.Time) error {
	_, err := db.Exec(`
		INSERT INTO parcelamentos (descricao, cartao, valor_parcela, parcela_atual, total_parcelas, data_inicio)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		descricao, cartao, valorParcela, parcelaAtual, totalParcelas, dataInicio)
	return err
}

func dbUpdateParcelamento(db *sql.DB, id, parcelaAtual int, ativo bool) error {
	_, err := db.Exec(`UPDATE parcelamentos SET parcela_atual=$1, ativo=$2 WHERE id=$3`,
		parcelaAtual, ativo, id)
	return err
}

func dbDeleteParcelamento(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM parcelamentos WHERE id=$1`, id)
	return err
}

// --- Receitas ---

func dbGetReceitas(db *sql.DB) ([]Receita, error) {
	rows, err := db.Query(`
		SELECT id, descricao, valor, data, tipo, recorrente, criado_em
		FROM receitas ORDER BY data DESC, criado_em DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Receita
	for rows.Next() {
		var r Receita
		rows.Scan(&r.ID, &r.Descricao, &r.Valor, &r.Data, &r.Tipo, &r.Recorrente, &r.CriadoEm)
		list = append(list, r)
	}
	return list, nil
}

// GetReceitasMes soma receitas recorrentes + receitas do mês
func dbGetReceitasMes(db *sql.DB, mes time.Time) (float64, error) {
	var total float64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(valor), 0)
		FROM receitas
		WHERE recorrente = true
		   OR (date_trunc('month', data) = $1)`, mes).Scan(&total)
	return total, err
}

func dbCreateReceita(db *sql.DB, descricao string, valor float64, data time.Time, tipo string, recorrente bool) error {
	_, err := db.Exec(`INSERT INTO receitas (descricao, valor, data, tipo, recorrente) VALUES ($1,$2,$3,$4,$5)`,
		descricao, valor, data, tipo, recorrente)
	return err
}

func dbDeleteReceita(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM receitas WHERE id=$1`, id)
	return err
}

// --- Histórico para gráficos ---

// dbGetHistoricoMeses retorna receitas, despesas (fixas + parcelamentos vigentes),
// e investimentos para cada um dos últimos N meses (incluindo mesRef).
func dbGetHistoricoMeses(db *sql.DB, mesRef time.Time, numMeses int) ([]MesResumo, error) {
	var resumos []MesResumo
	mesesPt := []string{"", "Jan", "Fev", "Mar", "Abr", "Mai", "Jun",
		"Jul", "Ago", "Set", "Out", "Nov", "Dez"}

	for i := numMeses - 1; i >= 0; i-- {
		mes := mesRef.AddDate(0, -i, 0)
		mesStr := mes.Format("2006-01")
		label := fmt.Sprintf("%s/%02d", mesesPt[int(mes.Month())], mes.Year()%100)

		// Receitas (recorrentes + as do mês)
		var receitas float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor), 0) FROM receitas
			WHERE recorrente = true OR date_trunc('month', data) = $1`, mes).Scan(&receitas)

		// Despesas fixas ativas (todas as ativas, pois sempre contam)
		var despesasFixas float64
		db.QueryRow(`SELECT COALESCE(SUM(valor), 0) FROM despesas_fixas WHERE ativa = true`).Scan(&despesasFixas)

		// Parcelamentos vigentes no mês
		var despesasParc float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor_parcela), 0) FROM parcelamentos
			WHERE ativo = true
			  AND data_inicio <= $1
			  AND (EXTRACT(YEAR  FROM $1) - EXTRACT(YEAR  FROM data_inicio)) * 12
			    + (EXTRACT(MONTH FROM $1) - EXTRACT(MONTH FROM data_inicio)) + 1 <= total_parcelas`, mes).Scan(&despesasParc)

		// Investimentos do mês
		var invest float64
		db.QueryRow(`
			SELECT COALESCE(SUM(valor), 0) FROM investimentos
			WHERE date_trunc('month', data) = $1`, mes).Scan(&invest)

		despesas := despesasFixas + despesasParc
		resumos = append(resumos, MesResumo{
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

func dbGetInvestimentos(db *sql.DB) ([]Investimento, error) {
	rows, err := db.Query(`
		SELECT id, instituicao, tipo, valor, data, notas, criado_em
		FROM investimentos ORDER BY data DESC, criado_em DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Investimento
	for rows.Next() {
		var i Investimento
		rows.Scan(&i.ID, &i.Instituicao, &i.Tipo, &i.Valor, &i.Data, &i.Notas, &i.CriadoEm)
		list = append(list, i)
	}
	return list, nil
}

func dbGetInvestimentosTotais(db *sql.DB) (map[string]float64, float64, error) {
	rows, err := db.Query(`
		SELECT instituicao, SUM(valor) FROM investimentos GROUP BY instituicao ORDER BY instituicao`)
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

func dbCreateInvestimento(db *sql.DB, instituicao, tipo string, valor float64, data time.Time, notas string) error {
	_, err := db.Exec(`INSERT INTO investimentos (instituicao, tipo, valor, data, notas) VALUES ($1,$2,$3,$4,$5)`,
		instituicao, tipo, valor, data, notas)
	return err
}

func dbDeleteInvestimento(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM investimentos WHERE id=$1`, id)
	return err
}

// --- Reserva de Emergência ---

func dbGetReservaEM(db *sql.DB) (float64, error) {
	var v float64
	err := db.QueryRow(`SELECT COALESCE(valor,0) FROM reserva_emergencia ORDER BY data DESC LIMIT 1`).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return v, err
}

func dbGetHistoricoReserva(db *sql.DB) ([]ReservaEM, error) {
	rows, err := db.Query(`SELECT id, valor, data, notas, criado_em FROM reserva_emergencia ORDER BY data DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ReservaEM
	for rows.Next() {
		var r ReservaEM
		rows.Scan(&r.ID, &r.Valor, &r.Data, &r.Notas, &r.CriadoEm)
		list = append(list, r)
	}
	return list, nil
}

func dbAddReservaEM(db *sql.DB, valor float64, notas string) error {
	_, err := db.Exec(`INSERT INTO reserva_emergencia (valor, notas) VALUES ($1,$2)`, valor, notas)
	return err
}

func dbDeleteReservaEM(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM reserva_emergencia WHERE id=$1`, id)
	return err
}

// --- Empréstimos ---

func dbGetEmprestimos(db *sql.DB) ([]Emprestimo, error) {
	rows, err := db.Query(`
		SELECT id, pessoa, valor, direcao, data, pago, notas, criado_em
		FROM emprestimos ORDER BY pago ASC, data DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Emprestimo
	for rows.Next() {
		var e Emprestimo
		rows.Scan(&e.ID, &e.Pessoa, &e.Valor, &e.Direcao, &e.Data, &e.Pago, &e.Notas, &e.CriadoEm)
		list = append(list, e)
	}
	return list, nil
}

func dbCreateEmprestimo(db *sql.DB, pessoa string, valor float64, direcao string, data time.Time, notas string) error {
	_, err := db.Exec(`INSERT INTO emprestimos (pessoa, valor, direcao, data, notas) VALUES ($1,$2,$3,$4,$5)`,
		pessoa, valor, direcao, data, notas)
	return err
}

func dbDeleteEmprestimo(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM emprestimos WHERE id=$1`, id)
	return err
}

func dbToggleEmprestimoPago(db *sql.DB, id int) (bool, error) {
	var pago bool
	err := db.QueryRow(`
		UPDATE emprestimos SET pago = NOT pago, pago_em = CASE WHEN NOT pago THEN NOW() ELSE NULL END
		WHERE id=$1 RETURNING pago`, id).Scan(&pago)
	return pago, err
}

// --- Migração automática ---

func dbAutoMigrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			nome       VARCHAR(100) NOT NULL,
			email      VARCHAR(150) UNIQUE NOT NULL,
			senha_hash TEXT NOT NULL,
			admin      BOOLEAN NOT NULL DEFAULT false,
			ativo      BOOLEAN NOT NULL DEFAULT true,
			criado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS sessions (
			token     TEXT PRIMARY KEY,
			user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expira_em TIMESTAMPTZ NOT NULL,
			criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

// dbEnsureAdmin cria o usuário admin padrão se não houver nenhum usuário
func dbEnsureAdmin(db *sql.DB) error {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO users (nome, email, senha_hash, admin) VALUES ('Admin', 'admin@finbertoldi.com', $1, true)`,
		string(hash))
	if err == nil {
		log.Println("Usuário admin criado: admin@finbertoldi.com / admin123")
	}
	return err
}

// --- Autenticação ---

func dbGetUserByEmail(db *sql.DB, email string) (*User, string, error) {
	var u User
	var hash string
	err := db.QueryRow(
		`SELECT id, nome, email, senha_hash, admin, ativo, criado_em FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Nome, &u.Email, &hash, &u.Admin, &u.Ativo, &u.CriadoEm)
	return &u, hash, err
}

func dbGetUserBySession(db *sql.DB, token string) (*User, error) {
	var u User
	err := db.QueryRow(`
		SELECT u.id, u.nome, u.email, u.admin, u.ativo, u.criado_em
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		WHERE s.token = $1 AND s.expira_em > NOW() AND u.ativo = true`,
		token).Scan(&u.ID, &u.Nome, &u.Email, &u.Admin, &u.Ativo, &u.CriadoEm)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func dbCreateSession(db *sql.DB, userID int, token string, expiry time.Time) error {
	_, err := db.Exec(`INSERT INTO sessions (token, user_id, expira_em) VALUES ($1, $2, $3)`,
		token, userID, expiry)
	return err
}

func dbDeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// --- Gestão de Usuários ---

func dbGetUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(
		`SELECT id, nome, email, admin, ativo, criado_em FROM users ORDER BY criado_em`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Nome, &u.Email, &u.Admin, &u.Ativo, &u.CriadoEm)
		list = append(list, u)
	}
	return list, nil
}

func dbCreateUser(db *sql.DB, nome, email, senhaHash string, admin bool) error {
	_, err := db.Exec(
		`INSERT INTO users (nome, email, senha_hash, admin) VALUES ($1,$2,$3,$4)`,
		nome, email, senhaHash, admin)
	return err
}

func dbToggleUserAdmin(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE users SET admin = NOT admin WHERE id = $1`, id)
	return err
}

func dbToggleUserAtivo(db *sql.DB, id int) error {
	_, err := db.Exec(`UPDATE users SET ativo = NOT ativo WHERE id = $1`, id)
	return err
}

func dbResetSenha(db *sql.DB, id int, hash string) error {
	_, err := db.Exec(`UPDATE users SET senha_hash = $1 WHERE id = $2`, hash, id)
	return err
}

func dbDeleteUser(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}

func hashSenha(senha string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	return string(b), err
}
