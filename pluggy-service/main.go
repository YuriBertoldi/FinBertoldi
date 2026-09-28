package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"pluggy-service/pluggy"
	psync "pluggy-service/sync"

	_ "github.com/lib/pq"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("db open:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("db ping:", err)
	}

	clientID := os.Getenv("PLUGGY_CLIENT_ID")
	clientSecret := os.Getenv("PLUGGY_CLIENT_SECRET")

	// If env vars empty, try to read from integracoes_config table
	if clientID == "" || clientSecret == "" {
		var id, secret string
		err := db.QueryRow(`SELECT client_id, client_secret FROM integracoes_config WHERE integracao='pluggy' AND ativo=true`).
			Scan(&id, &secret)
		if err == nil && id != "" && secret != "" {
			clientID = id
			clientSecret = secret
			log.Println("[config] credenciais carregadas do banco de dados")
		} else {
			log.Println("[config] credenciais nao encontradas (env ou banco). Servico iniciara sem autenticacao Pluggy.")
		}
	}

	client := pluggy.NewClient(clientID, clientSecret)
	syncer := psync.NewSyncer(db, client)

	// refreshCredentials re-reads credentials from DB before sync
	refreshCredentials := func() {
		var id, secret string
		err := db.QueryRow(`SELECT client_id, client_secret FROM integracoes_config WHERE integracao='pluggy' AND ativo=true`).
			Scan(&id, &secret)
		if err == nil && id != "" && secret != "" {
			client.UpdateCredentials(id, secret)
		}
	}

	// Background sync scheduler
	interval := getEnv("SYNC_INTERVAL", "6h")
	dur, err := time.ParseDuration(interval)
	if err != nil {
		dur = 6 * time.Hour
	}
	go func() {
		log.Printf("[scheduler] sync a cada %s", dur)
		for {
			time.Sleep(dur)
			refreshCredentials()
			log.Println("[scheduler] iniciando sync...")
			syncer.SyncAll()
		}
	}()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /api/pluggy/status", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Generate connect token for Pluggy Connect Widget
	mux.HandleFunc("POST /api/pluggy/connect-token", func(w http.ResponseWriter, r *http.Request) {
		refreshCredentials()
		var req struct {
			ClientUserID string `json:"clientUserId"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		token, err := client.CreateConnectToken(req.ClientUserID)
		if err != nil {
			log.Printf("[connect-token] erro: %v", err)
			http.Error(w, "failed to create connect token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"accessToken": token})
	})

	// Webhook from Pluggy
	mux.HandleFunc("POST /api/pluggy/webhook", func(w http.ResponseWriter, r *http.Request) {
		var payload pluggy.WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		log.Printf("[webhook] event=%s id=%s", payload.Event, payload.ID)

		switch payload.Event {
		case "item/created", "item/updated":
			item, err := client.GetItem(payload.ID)
			if err != nil {
				log.Printf("[webhook] erro ao buscar item: %v", err)
				w.WriteHeader(http.StatusOK)
				return
			}

			// Try to find family_id for this item
			var familyID int
			err = db.QueryRow(`SELECT family_id FROM pluggy_items WHERE item_id=$1`, payload.ID).Scan(&familyID)
			if err != nil {
				log.Printf("[webhook] item %s not found in DB, skipping sync", payload.ID)
				w.WriteHeader(http.StatusOK)
				return
			}

			db.Exec(`UPDATE pluggy_items SET connector_name=$1, status=$2 WHERE item_id=$3`,
				item.Connector.Name, item.Status, payload.ID)

			if item.Status == "UPDATED" || item.Status == "LOGIN_SUCCESS" {
				go func() {
					count, err := syncer.SyncItem(payload.ID)
					if err != nil {
						log.Printf("[webhook] sync erro: %v", err)
						return
					}
					log.Printf("[webhook] synced %d transações para item %s", count, payload.ID)
				}()
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	// Register a new Pluggy item (called from main app after Connect Widget success)
	mux.HandleFunc("POST /api/pluggy/items", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FamilyID      int    `json:"familyId"`
			ItemID        string `json:"itemId"`
			ConnectorName string `json:"connectorName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(`INSERT INTO pluggy_items (family_id, item_id, connector_name)
			VALUES ($1,$2,$3) ON CONFLICT (item_id) DO UPDATE SET connector_name=$3`,
			req.FamilyID, req.ItemID, req.ConnectorName)
		if err != nil {
			log.Printf("[items] erro ao salvar: %v", err)
			http.Error(w, "failed to save item", http.StatusInternalServerError)
			return
		}

		// Trigger initial sync
		go func() {
			count, err := syncer.SyncItem(req.ItemID)
			if err != nil {
				log.Printf("[items] sync inicial erro: %v", err)
				return
			}
			log.Printf("[items] sync inicial: %d transações", count)
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Force sync for a specific item
	mux.HandleFunc("POST /api/pluggy/sync/{item_id}", func(w http.ResponseWriter, r *http.Request) {
		itemID := r.PathValue("item_id")
		count, err := syncer.SyncItem(itemID)
		if err != nil {
			http.Error(w, fmt.Sprintf("sync failed: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"synced": count})
	})

	port := getEnv("PORT", "8081")
	log.Printf("Pluggy service iniciado em :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
