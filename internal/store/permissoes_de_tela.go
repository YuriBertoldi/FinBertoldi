package store

import "database/sql"

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
