package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fincontrol/internal/models"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

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

func CreateUserReturningID(db *sql.DB, nome, email, senhaHash string, familyAdmin bool, familyID int) (int, error) {
	var id int
	err := db.QueryRow(`INSERT INTO users (nome, email, senha_hash, admin, family_admin, family_id)
		VALUES ($1,$2,$3,false,$4,$5) RETURNING id`,
		nome, email, senhaHash, familyAdmin, familyID).Scan(&id)
	return id, err
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
	return TrocarSenha(db, id, hash, "")
}

// TrocarSenha grava a nova senha e encerra as sessões do usuário em outros aparelhos.
// manterToken é a sessão atual (quem trocou a própria senha continua logado); vazio encerra todas.
func TrocarSenha(db *sql.DB, id int, hash, manterToken string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE users SET senha_hash=$1 WHERE id=$2`, hash, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=$1 AND token<>$2`, id, manterToken); err != nil {
		return err
	}
	return tx.Commit()
}

func DeleteUser(db *sql.DB, id int) error {
	db.Exec(`DELETE FROM password_resets WHERE user_id=$1`, id)
	db.Exec(`DELETE FROM screen_permissions WHERE user_id=$1`, id)
	db.Exec(`DELETE FROM dashboard_widgets WHERE user_id=$1`, id)
	_, err := db.Exec(`DELETE FROM users WHERE id=$1`, id)
	return err
}

// EnsureAdmin cria o usuário admin padrão se não houver nenhum usuário.
func EnsureAdmin(db *sql.DB) error {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count > 0 {
		return nil
	}
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@finbertoldi.com"
	}
	adminPass := os.Getenv("ADMIN_PASSWORD")
	generated := false
	if adminPass == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("gerar senha: %w", err)
		}
		adminPass = hex.EncodeToString(b)
		generated = true
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var familyID int
	if err := db.QueryRow(`SELECT id FROM families WHERE nome='Família Padrão'`).Scan(&familyID); err != nil {
		return fmt.Errorf("família padrão não encontrada: %w", err)
	}
	_, err = db.Exec(
		`INSERT INTO users (nome, email, senha_hash, admin, family_id) VALUES ('Admin', $1, $2, true, $3)`,
		adminEmail, string(hash), familyID)
	if err == nil {
		if generated {
			log.Printf("Usuário admin criado: %s / %s (ANOTE ESTA SENHA — não será exibida novamente)", adminEmail, adminPass)
		} else {
			log.Printf("Usuário admin criado: %s", adminEmail)
		}
	}
	return err
}
