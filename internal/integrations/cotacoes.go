package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"fincontrol/internal/store"
)

type cotacaoItem struct {
	Bid string `json:"bid"`
}

func FetchCotacoes(db *sql.DB) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://economia.awesomeapi.com.br/last/USD-BRL,EUR-BRL,BTC-BRL")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("awesomeapi retornou status %d", resp.StatusCode)
	}

	var dados map[string]cotacaoItem
	if err := json.NewDecoder(resp.Body).Decode(&dados); err != nil {
		return err
	}

	moedas := map[string]string{
		"USDBRL": "usd",
		"EURBRL": "eur",
		"BTCBRL": "btc",
	}

	now := time.Now()
	for key, nome := range moedas {
		if d, ok := dados[key]; ok {
			valor, err := strconv.ParseFloat(d.Bid, 64)
			if err != nil {
				continue
			}
			if err := store.UpsertDadoEconomico(db, nome, valor, now, "awesomeapi"); err != nil {
				log.Printf("[cotacoes] erro ao salvar %s: %v", nome, err)
			} else {
				log.Printf("[cotacoes] %s = R$ %.2f", nome, valor)
			}
		}
	}
	return nil
}
