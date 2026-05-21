//go:build integration
// +build integration

package store

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"fincontrol/internal/models"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Testes de integração — requerem PostgreSQL rodando.
// Execute com: go test -tags=integration ./...
//
// Variáveis de ambiente:
//   DATABASE_URL (opcional) — connection string completa
//   Ou: DB_HOST, DB_PORT, DB_USER, DB_PASS, DB_NAME (defaults: localhost:5432/fincontrol)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "fincontrol")
		pass := getEnv("DB_PASS", "fincontrol")
		name := getEnv("DB_NAME", "fincontrol")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, pass, name)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("postgres não disponível: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("postgres não responde: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := db.Exec(fmt.Sprintf(`CREATE SCHEMA %s`, schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(fmt.Sprintf(`SET search_path TO %s`, schema)); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if _, err := db.Exec(`SET TIME ZONE 'UTC'`); err != nil {
		t.Fatalf("set tz: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`SET search_path TO public`)
		db.Exec(fmt.Sprintf(`DROP SCHEMA %s CASCADE`, schema))
		db.Close()
	})

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	return db
}

func criarDuasFamilias(t *testing.T, db *sql.DB) (int, int) {
	t.Helper()
	fA, err := CreateFamily(db, "Família A")
	if err != nil {
		t.Fatalf("criar família A: %v", err)
	}
	fB, err := CreateFamily(db, "Família B")
	if err != nil {
		t.Fatalf("criar família B: %v", err)
	}
	return fA, fB
}

// ====================== ISOLAMENTO MULTI-TENANT ======================

func TestIsolamento_Despesas(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)

	CreateDespesa(db, fA, "Netflix A", 50, "Cartão")
	CreateDespesa(db, fA, "Água A", 80, "Básicas")
	CreateDespesa(db, fB, "Netflix B", 60, "Cartão")

	despA, _ := GetDespesasFixas(db, fA)
	despB, _ := GetDespesasFixas(db, fB)

	if len(despA) != 2 {
		t.Errorf("Família A: esperado 2 despesas, got %d", len(despA))
	}
	if len(despB) != 1 {
		t.Errorf("Família B: esperado 1 despesa, got %d", len(despB))
	}
	for _, d := range despA {
		if d.Nome == "Netflix B" {
			t.Error("Família A está vendo despesa da família B! VAZAMENTO")
		}
	}
	for _, d := range despB {
		if d.Nome == "Netflix A" || d.Nome == "Água A" {
			t.Error("Família B está vendo despesa da família A! VAZAMENTO")
		}
	}
}

func TestIsolamento_Receitas(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	hoje := time.Now()

	CreateReceita(db, fA, "Salário A", 5000, hoje, "Salário", true)
	CreateReceita(db, fB, "Salário B", 8000, hoje, "Salário", true)

	mes := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, time.UTC)
	totalA, _ := GetReceitasMes(db, fA, mes)
	totalB, _ := GetReceitasMes(db, fB, mes)

	if totalA != 5000 {
		t.Errorf("receita A = %.2f; want 5000", totalA)
	}
	if totalB != 8000 {
		t.Errorf("receita B = %.2f; want 8000", totalB)
	}
}

func TestIsolamento_Investimentos(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	hoje := time.Now()

	CreateInvestimento(db, fA, "Rico", "Renda Fixa", 10000, hoje, "")
	CreateInvestimento(db, fA, "INCO", "FII", 5000, hoje, "")
	CreateInvestimento(db, fB, "XP", "Renda Variável", 30000, hoje, "")

	_, totalA, _ := GetInvestimentosTotais(db, fA)
	_, totalB, _ := GetInvestimentosTotais(db, fB)

	if totalA != 15000 {
		t.Errorf("invest A = %.2f; want 15000", totalA)
	}
	if totalB != 30000 {
		t.Errorf("invest B = %.2f; want 30000", totalB)
	}
}

func TestIsolamento_ReservaEM(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)

	AddReservaEM(db, fA, 20000, "Inicial A")
	AddReservaEM(db, fB, 50000, "Inicial B")

	vA, _ := GetReservaEM(db, fA)
	vB, _ := GetReservaEM(db, fB)

	if vA != 20000 {
		t.Errorf("reserva A = %.2f; want 20000", vA)
	}
	if vB != 50000 {
		t.Errorf("reserva B = %.2f; want 50000", vB)
	}
}

func TestIsolamento_Parcelamentos(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	inicio := time.Now().AddDate(0, -3, 0)

	CreateParcelamento(db, fA, "Geladeira A", "Nubank", 200, 1, 10, inicio)
	CreateParcelamento(db, fB, "TV B", "Itaú", 500, 1, 12, inicio)

	listA, _ := GetParcelamentos(db, fA, false)
	listB, _ := GetParcelamentos(db, fB, false)

	if len(listA) != 1 || listA[0].Descricao != "Geladeira A" {
		t.Errorf("parc A inesperado: %+v", listA)
	}
	if len(listB) != 1 || listB[0].Descricao != "TV B" {
		t.Errorf("parc B inesperado: %+v", listB)
	}
}

func TestIsolamento_Emprestimos(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	hoje := time.Now()

	CreateEmprestimo(db, fA, "Pai", 1000, "devo", hoje, "")
	CreateEmprestimo(db, fB, "Amigo", 500, "emprestei", hoje, "")

	listA, _ := GetEmprestimos(db, fA)
	listB, _ := GetEmprestimos(db, fB)

	if len(listA) != 1 || listA[0].Pessoa != "Pai" {
		t.Errorf("empr A: %+v", listA)
	}
	if len(listB) != 1 || listB[0].Pessoa != "Amigo" {
		t.Errorf("empr B: %+v", listB)
	}
}

// ====================== TENTATIVA DE ESCAPE (segurança) ======================

func TestSeguranca_UpdateForaDaFamilia(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	CreateDespesa(db, fA, "Netflix A", 50, "Cartão")

	var idA int
	db.QueryRow(`SELECT id FROM despesas_fixas WHERE family_id=$1`, fA).Scan(&idA)
	if idA == 0 {
		t.Fatal("despesa A não foi criada")
	}

	err := UpdateDespesa(db, fB, idA, "HACKED", 99999, "Cartão")
	if err != nil {
		t.Errorf("update inesperado falhou: %v", err)
	}

	var nome string
	var valor float64
	db.QueryRow(`SELECT nome, valor FROM despesas_fixas WHERE id=$1`, idA).Scan(&nome, &valor)
	if nome == "HACKED" || valor == 99999 {
		t.Errorf("VAZAMENTO: família B alterou despesa da A! nome=%q valor=%.2f", nome, valor)
	}
	if nome != "Netflix A" || valor != 50 {
		t.Errorf("despesa A alterada inesperadamente: %q %.2f", nome, valor)
	}
}

func TestSeguranca_DeleteForaDaFamilia(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	CreateDespesa(db, fA, "Netflix A", 50, "Cartão")

	var idA int
	db.QueryRow(`SELECT id FROM despesas_fixas WHERE family_id=$1`, fA).Scan(&idA)

	DeleteDespesa(db, fB, idA)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM despesas_fixas WHERE id=$1`, idA).Scan(&count)
	if count != 1 {
		t.Error("VAZAMENTO: família B conseguiu excluir despesa da A")
	}
}

func TestSeguranca_ToggleDespesaPagoForaDaFamilia(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)
	CreateDespesa(db, fA, "Netflix A", 50, "Cartão")
	var idA int
	db.QueryRow(`SELECT id FROM despesas_fixas WHERE family_id=$1`, fA).Scan(&idA)

	mes := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err := ToggleDespesaMesPago(db, fB, idA, mes)
	if err == nil {
		t.Error("toggle pago de despesa de OUTRA família deveria falhar, mas não falhou")
	}
}

// ====================== LÓGICA DE PARCELAMENTOS ======================

func TestParcelamento_AutoFinalizar(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	inicio := time.Now().AddDate(0, -5, 0)
	CreateParcelamento(db, fA, "Antigo", "X", 100, 1, 3, inicio)
	CreateParcelamento(db, fA, "Ativo", "X", 200, 1, 12, time.Now().AddDate(0, -2, 0))

	if err := AutoFinalizarParcelamentos(db, fA); err != nil {
		t.Fatalf("auto finalizar: %v", err)
	}

	var ativos int
	db.QueryRow(`SELECT COUNT(*) FROM parcelamentos WHERE family_id=$1 AND ativo=true`, fA).Scan(&ativos)
	if ativos != 1 {
		t.Errorf("após auto-finalizar, esperado 1 ativo, got %d", ativos)
	}

	var antigoAtivo bool
	db.QueryRow(`SELECT ativo FROM parcelamentos WHERE family_id=$1 AND descricao='Antigo'`, fA).Scan(&antigoAtivo)
	if antigoAtivo {
		t.Error("parcelamento 'Antigo' (5 meses passados, 3 parcelas) deveria estar inativo")
	}
}

func TestParcelamento_VigenteNoMes(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := CreateParcelamento(db, fA, "Notebook", "Visa", 500, 1, 12, inicio); err != nil {
		t.Fatalf("create: %v", err)
	}

	mesMeio := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	listMeio, err := GetParcelamentosVigentesNoMes(db, fA, mesMeio)
	if err != nil {
		t.Fatalf("get vigentes: %v", err)
	}
	if len(listMeio) != 1 {
		t.Errorf("06/2026 deveria ter 1 parcelamento vigente, got %d", len(listMeio))
	} else if listMeio[0].ParcelaAtual != 6 {
		t.Errorf("parcela atual = %d; want 6", listMeio[0].ParcelaAtual)
	}

	mesDepois := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	listDepois, _ := GetParcelamentosVigentesNoMes(db, fA, mesDepois)
	if len(listDepois) != 0 {
		t.Errorf("02/2027 não deveria ter parcelamentos vigentes, got %d", len(listDepois))
	}

	mesAntes := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	listAntes, _ := GetParcelamentosVigentesNoMes(db, fA, mesAntes)
	if len(listAntes) != 0 {
		t.Errorf("12/2025 (antes do início) não deveria ter parcelamentos, got %d", len(listAntes))
	}
}

// ====================== USUÁRIOS E AUTENTICAÇÃO ======================

func TestUsuario_CriarELogarComFamilia(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	hash, _ := HashSenha("senha123")
	if err := CreateUser(db, "João", "joao@teste.com", hash, false, fA); err != nil {
		t.Fatalf("criar usuário: %v", err)
	}

	u, gotHash, err := GetUserByEmail(db, "joao@teste.com")
	if err != nil {
		t.Fatalf("buscar usuário: %v", err)
	}
	if u.Nome != "João" || u.Email != "joao@teste.com" {
		t.Errorf("usuário errado: %+v", u)
	}
	if u.FamilyID != fA {
		t.Errorf("FamilyID = %d; want %d", u.FamilyID, fA)
	}
	if u.FamilyNome != "Família A" {
		t.Errorf("FamilyNome = %q; want 'Família A'", u.FamilyNome)
	}
	if gotHash == "senha123" {
		t.Error("hash não pode ser igual à senha em texto puro")
	}
}

func TestUsuario_MoverEntreFamilias(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)

	hash, _ := HashSenha("senha123")
	CreateUser(db, "Maria", "maria@t.com", hash, false, fA)

	var userID int
	db.QueryRow(`SELECT id FROM users WHERE email='maria@t.com'`).Scan(&userID)

	if err := SetUserFamily(db, userID, fB); err != nil {
		t.Fatalf("mover família: %v", err)
	}

	u, _, _ := GetUserByEmail(db, "maria@t.com")
	if u.FamilyID != fB {
		t.Errorf("após mover, FamilyID = %d; want %d", u.FamilyID, fB)
	}
}

func TestFamilia_NaoExcluiComMembros(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	hash, _ := HashSenha("senha123")
	CreateUser(db, "Pedro", "pedro@t.com", hash, false, fA)

	err := DeleteFamily(db, fA)
	if err == nil {
		t.Error("DeleteFamily deveria falhar quando família tem membros")
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM families WHERE id=$1`, fA).Scan(&count)
	if count != 1 {
		t.Error("família foi excluída mesmo tendo membros")
	}
}

func TestFamilia_ExcluiVazia(t *testing.T) {
	db := setupTestDB(t)

	fid, err := CreateFamily(db, "Família Temporária")
	if err != nil {
		t.Fatalf("criar família: %v", err)
	}

	if err := DeleteFamily(db, fid); err != nil {
		t.Errorf("excluir família vazia falhou: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM families WHERE id=$1`, fid).Scan(&count)
	if count != 0 {
		t.Error("família vazia não foi excluída")
	}
}

func TestFamilia_RenomearListarContagem(t *testing.T) {
	db := setupTestDB(t)
	fA, fB := criarDuasFamilias(t, db)

	hash, _ := HashSenha("x")
	CreateUser(db, "A1", "a1@t.com", hash, false, fA)
	CreateUser(db, "A2", "a2@t.com", hash, false, fA)
	CreateUser(db, "B1", "b1@t.com", hash, false, fB)

	if err := RenameFamily(db, fA, "Família A Renomeada"); err != nil {
		t.Fatalf("renomear: %v", err)
	}

	familias, err := GetFamilies(db)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}

	var encA, encB *models.Family
	for i := range familias {
		switch familias[i].ID {
		case fA:
			encA = &familias[i]
		case fB:
			encB = &familias[i]
		}
	}
	if encA == nil || encB == nil {
		t.Fatal("famílias não encontradas na listagem")
	}
	if encA.Nome != "Família A Renomeada" {
		t.Errorf("nome A = %q; want 'Família A Renomeada'", encA.Nome)
	}
	if encA.NumMembros != 2 {
		t.Errorf("membros A = %d; want 2", encA.NumMembros)
	}
	if encB.NumMembros != 1 {
		t.Errorf("membros B = %d; want 1", encB.NumMembros)
	}
}

// ====================== CRUD COMPLETO ======================

func TestDespesa_CrudCompleto(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	if err := CreateDespesa(db, fA, "Spotify", 21.90, "Cartão"); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, _ := GetDespesasFixas(db, fA)
	if len(list) != 1 {
		t.Fatalf("listar: esperado 1, got %d", len(list))
	}
	id := list[0].ID

	if err := UpdateDespesa(db, fA, id, "Spotify Family", 34.90, "Cartão"); err != nil {
		t.Fatalf("update: %v", err)
	}
	list, _ = GetDespesasFixas(db, fA)
	if list[0].Nome != "Spotify Family" || list[0].Valor != 34.90 {
		t.Errorf("após update: %+v", list[0])
	}

	ToggleDespesaAtiva(db, fA, id)
	list, _ = GetDespesasFixas(db, fA)
	if list[0].Ativa {
		t.Error("após toggle, ativa deveria ser false")
	}

	if err := DeleteDespesa(db, fA, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = GetDespesasFixas(db, fA)
	if len(list) != 0 {
		t.Errorf("após delete: esperado 0, got %d", len(list))
	}
}

func TestDespesa_TogglePagoNoMes(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	CreateDespesa(db, fA, "Água", 80, "Básicas")
	var id int
	db.QueryRow(`SELECT id FROM despesas_fixas WHERE family_id=$1`, fA).Scan(&id)

	mes := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	EnsureDespesasMes(db, fA, mes)

	pago, err := ToggleDespesaMesPago(db, fA, id, mes)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !pago {
		t.Error("primeiro toggle deveria retornar pago=true")
	}

	pago2, _ := ToggleDespesaMesPago(db, fA, id, mes)
	if pago2 {
		t.Error("segundo toggle deveria retornar pago=false")
	}
}

func TestEmprestimo_TogglePago(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	CreateEmprestimo(db, fA, "Tio", 300, "devo", time.Now(), "")
	var id int
	db.QueryRow(`SELECT id FROM emprestimos WHERE family_id=$1`, fA).Scan(&id)

	pago, err := ToggleEmprestimoPago(db, fA, id)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !pago {
		t.Error("toggle deveria marcar como pago")
	}

	pago2, _ := ToggleEmprestimoPago(db, fA, id)
	if pago2 {
		t.Error("segundo toggle deveria desmarcar")
	}
}

// ====================== HISTÓRICO E AGREGAÇÕES ======================

func TestHistoricoMeses(t *testing.T) {
	db := setupTestDB(t)
	fA, _ := criarDuasFamilias(t, db)

	hoje := time.Now()
	mesAtual := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, time.UTC)

	CreateReceita(db, fA, "Salário", 6000, hoje, "Salário", true)
	CreateDespesa(db, fA, "Aluguel", 1500, "Básicas")

	hist, err := GetHistoricoMeses(db, fA, mesAtual, 6)
	if err != nil {
		t.Fatalf("histórico: %v", err)
	}
	if len(hist) != 6 {
		t.Errorf("esperado 6 meses, got %d", len(hist))
	}

	for _, m := range hist {
		if m.Receitas != 6000 {
			t.Errorf("mês %s: receitas = %.2f; want 6000", m.Mes, m.Receitas)
		}
		if m.Despesas != 1500 {
			t.Errorf("mês %s: despesas = %.2f; want 1500", m.Mes, m.Despesas)
		}
		if m.Sobra != 4500 {
			t.Errorf("mês %s: sobra = %.2f; want 4500", m.Mes, m.Sobra)
		}
	}
}

// ====================== MIGRAÇÃO E ADMIN PADRÃO ======================

func TestMigrate_Idempotente(t *testing.T) {
	db := setupTestDB(t)

	if err := RunMigrations(db); err != nil {
		t.Errorf("RunMigrations segunda vez: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Errorf("RunMigrations terceira vez: %v", err)
	}
}

func TestMigrate_CriaFamiliaPadrao(t *testing.T) {
	db := setupTestDB(t)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM families WHERE nome='Família Padrão'`).Scan(&count)
	if count != 1 {
		t.Errorf("Família Padrão não foi criada (count=%d)", count)
	}
}

func TestEnsureAdmin_CriaQuandoVazio(t *testing.T) {
	db := setupTestDB(t)

	if err := EnsureAdmin(db); err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}

	u, hash, err := GetUserByEmail(db, "admin@finbertoldi.com")
	if err != nil {
		t.Fatalf("admin não criado: %v", err)
	}
	if !u.Admin {
		t.Error("admin criado sem flag admin")
	}
	if !u.Ativo {
		t.Error("admin criado inativo")
	}
	if u.FamilyID == 0 {
		t.Error("admin sem family_id")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("admin123")); err != nil {
		t.Error("senha padrão não valida")
	}
}

func TestEnsureAdmin_NaoCriaQuandoJaTem(t *testing.T) {
	db := setupTestDB(t)
	EnsureAdmin(db)

	var countAntes int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&countAntes)

	EnsureAdmin(db)

	var countDepois int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&countDepois)

	if countAntes != countDepois {
		t.Errorf("EnsureAdmin chamado 2x: count antes=%d depois=%d", countAntes, countDepois)
	}
}
