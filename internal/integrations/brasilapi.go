package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"fincontrol/internal/store"
)

type feriadoAPI struct {
	Date string `json:"date"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func FetchFeriados(db *sql.DB, ano int) error {
	client := &http.Client{Timeout: 15 * time.Second}
	url := fmt.Sprintf("https://brasilapi.com.br/api/feriados/v1/%d", ano)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var feriados []feriadoAPI
	if err := json.NewDecoder(resp.Body).Decode(&feriados); err != nil {
		return err
	}

	for _, f := range feriados {
		data, err := time.Parse("2006-01-02", f.Date)
		if err != nil {
			continue
		}
		if err := store.UpsertFeriado(db, data, f.Name, f.Type); err != nil {
			log.Printf("[brasilapi] erro ao salvar feriado %s: %v", f.Name, err)
		}
	}
	log.Printf("[brasilapi] %d feriados de %d importados", len(feriados), ano)
	return nil
}
