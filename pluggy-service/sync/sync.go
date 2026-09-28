package sync

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"pluggy-service/pluggy"
)

type Syncer struct {
	db     *sql.DB
	client *pluggy.Client
}

func NewSyncer(db *sql.DB, client *pluggy.Client) *Syncer {
	return &Syncer{db: db, client: client}
}

// SyncAll fetches transactions for all active Pluggy items and inserts new ones.
func (s *Syncer) SyncAll() {
	rows, err := s.db.Query(`SELECT id, family_id, item_id, connector_name FROM pluggy_items WHERE status='active'`)
	if err != nil {
		log.Printf("[sync] erro ao listar items: %v", err)
		return
	}
	defer rows.Close()

	type item struct {
		id            int
		familyID      int
		itemID        string
		connectorName string
	}
	var items []item
	for rows.Next() {
		var it item
		rows.Scan(&it.id, &it.familyID, &it.itemID, &it.connectorName)
		items = append(items, it)
	}

	for _, it := range items {
		log.Printf("[sync] sincronizando item %s (%s)...", it.itemID, it.connectorName)
		count, err := s.syncItem(it.familyID, it.itemID, it.connectorName)
		if err != nil {
			log.Printf("[sync] erro item %s: %v", it.itemID, err)
			s.db.Exec(`UPDATE pluggy_items SET status='error' WHERE item_id=$1`, it.itemID)
			continue
		}
		s.db.Exec(`UPDATE pluggy_items SET last_sync=NOW(), status='active' WHERE item_id=$1`, it.itemID)
		log.Printf("[sync] item %s: %d novas transações", it.itemID, count)
	}
}

func (s *Syncer) syncItem(familyID int, itemID, connectorName string) (int, error) {
	accounts, err := s.client.ListAccounts(itemID)
	if err != nil {
		return 0, fmt.Errorf("listar contas: %w", err)
	}

	to := time.Now()
	from := to.AddDate(0, -1, 0) // last 30 days

	total := 0
	for _, acc := range accounts {
		txns, err := s.client.ListTransactions(acc.ID, from, to)
		if err != nil {
			log.Printf("[sync] erro transações conta %s: %v", acc.ID, err)
			continue
		}

		for _, t := range txns {
			fitID := t.ProviderCode
			if fitID == "" {
				fitID = t.ID
			}

			var exists bool
			s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM transacoes_banco WHERE family_id=$1 AND fit_id=$2)`,
				familyID, fitID).Scan(&exists)
			if exists {
				continue
			}

			tipo := "debito"
			if t.Amount > 0 {
				tipo = "credito"
			}

			cat := ""
			if t.Category != nil {
				cat = *t.Category
			}

			desc := t.Description
			if desc == "" {
				desc = t.DescriptionRaw
			}

			_, err := s.db.Exec(`INSERT INTO transacoes_banco
				(family_id, data, descricao, valor, tipo, categoria, origem, banco, fit_id, status, pluggy_item_id)
				VALUES ($1,$2,$3,$4,$5,$6,'pluggy',$7,$8,'pendente',$9)`,
				familyID, t.Date, desc, math.Abs(t.Amount), tipo, cat, connectorName, fitID, itemID)
			if err != nil {
				log.Printf("[sync] erro insert txn %s: %v", fitID, err)
				continue
			}
			total++
		}
	}
	return total, nil
}

// SyncItem forces a sync for a specific item.
func (s *Syncer) SyncItem(itemID string) (int, error) {
	var familyID int
	var connectorName string
	err := s.db.QueryRow(`SELECT family_id, connector_name FROM pluggy_items WHERE item_id=$1`, itemID).
		Scan(&familyID, &connectorName)
	if err != nil {
		return 0, fmt.Errorf("item não encontrado: %w", err)
	}
	return s.syncItem(familyID, itemID, connectorName)
}
