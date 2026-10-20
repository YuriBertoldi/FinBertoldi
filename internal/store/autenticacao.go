package store

import (
	"database/sql"
	"fincontrol/internal/models"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

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

var googlePending = struct {
	sync.Mutex
	data map[string][2]string // token -> [email, nome]
}{data: map[string][2]string{}}

func SaveGooglePending(_ *sql.DB, token, email, nome string) {
	googlePending.Lock()
	defer googlePending.Unlock()
	googlePending.data[token] = [2]string{email, nome}
}

func GetGooglePending(_ *sql.DB, token string) (email, nome string, ok bool) {
	googlePending.Lock()
	defer googlePending.Unlock()
	d, exists := googlePending.data[token]
	if exists {
		delete(googlePending.data, token)
	}
	return d[0], d[1], exists
}
