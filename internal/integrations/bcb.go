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

// Códigos das séries do BCB
// 432 = Selic Meta, 4389 = CDI, 433 = IPCA
var bcbSeries = map[string]int{
	"selic": 432,
	"cdi":   4389,
	"ipca":  433,
}

type bcbResponse struct {
	Data  string `json:"data"`
	Valor string `json:"valor"`
}

func FetchBCB(db *sql.DB) error {
	client := &http.Client{Timeout: 15 * time.Second}

	for nome, codigo := range bcbSeries {
		url := fmt.Sprintf("https://api.bcb.gov.br/dados/serie/bcdata.sgs.%d/dados/ultimos/1?formato=json", codigo)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[bcb] erro ao buscar %s: %v", nome, err)
			continue
		}

		var dados []bcbResponse
		if err := json.NewDecoder(resp.Body).Decode(&dados); err != nil {
			resp.Body.Close()
			log.Printf("[bcb] erro ao decodar %s: %v", nome, err)
			continue
		}
		resp.Body.Close()

		if len(dados) == 0 {
			continue
		}

		var valor float64
		fmt.Sscanf(dados[0].Valor, "%f", &valor)

		data, err := time.Parse("02/01/2006", dados[0].Data)
		if err != nil {
			data = time.Now()
		}

		if err := store.UpsertDadoEconomico(db, nome, valor, data, "bcb"); err != nil {
			log.Printf("[bcb] erro ao salvar %s: %v", nome, err)
		} else {
			log.Printf("[bcb] %s = %.4f (%s)", nome, valor, dados[0].Data)
		}
	}
	return nil
}
