package store

import (
	"database/sql"
	"time"
)

func CreatePasswordReset(db *sql.DB, userID int, token string, expiry time.Time) error {
	_, err := db.Exec(`INSERT INTO password_resets (user_id, token, expira_em) VALUES ($1, $2, $3)`,
		userID, token, expiry)
	return err
}

func GetPasswordReset(db *sql.DB, token string) (int, error) {
	var userID int
	err := db.QueryRow(`SELECT user_id FROM password_resets
		WHERE token=$1 AND usado=false AND expira_em > NOW()`, token).Scan(&userID)
	return userID, err
}

func UsePasswordReset(db *sql.DB, token string) error {
	_, err := db.Exec(`UPDATE password_resets SET usado=true WHERE token=$1`, token)
	return err
}

func UpdateUserPassword(db *sql.DB, userID int, hashedPassword string) error {
	return TrocarSenha(db, userID, hashedPassword, "")
}

func GetUserIDByEmail(db *sql.DB, email string) (int, error) {
	var id int
	err := db.QueryRow(`SELECT id FROM users WHERE email=$1 AND ativo=true`, email).Scan(&id)
	return id, err
}
