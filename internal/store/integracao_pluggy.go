package store

import (
	"database/sql"
	"fincontrol/internal/models"
)

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
