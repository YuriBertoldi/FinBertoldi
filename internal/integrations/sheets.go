package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"fincontrol/internal/store"
)

// ExportToSheet exporta dados para Google Sheets via API
// Requer: spreadsheet_id e service_account_json configurados em integracoes_kv
func ExportToSheet(db *sql.DB, fid int) error {
	spreadsheetID := store.GetIntegracaoKV(db, fid, "sheets", "spreadsheet_id")
	saJSON := store.GetIntegracaoKV(db, fid, "sheets", "service_account_json")
	if spreadsheetID == "" || saJSON == "" {
		return fmt.Errorf("Google Sheets não configurado")
	}

	// Obter access token via service account
	token, err := getGoogleAccessToken(saJSON)
	if err != nil {
		return fmt.Errorf("erro ao obter token: %w", err)
	}

	// Exportar transações do mês
	mes := time.Now().Format("2006-01")
	txns, _ := store.GetTransacoesBanco(db, fid, mes, "", "", "")

	var rows [][]string
	rows = append(rows, []string{"Data", "Descricao", "Valor", "Tipo", "Categoria", "Status"})
	for _, t := range txns {
		rows = append(rows, []string{
			t.Data.Format("02/01/2006"),
			t.Descricao,
			fmt.Sprintf("%.2f", t.Valor),
			t.Tipo,
			t.Categoria,
			t.Status,
		})
	}

	return writeToSheet(token, spreadsheetID, "Transacoes!A1", rows)
}

func getGoogleAccessToken(saJSON string) (string, error) {
	// Parse service account JSON
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		TokenURI    string `json:"token_uri"`
	}
	if err := json.Unmarshal([]byte(saJSON), &sa); err != nil {
		return "", err
	}

	// Para simplificar, usamos o flow de API key / OAuth simples
	// Em produção, seria JWT bearer token
	// Por enquanto, retornamos erro informativo
	return "", fmt.Errorf("Google Sheets requer configuracao de Service Account (JWT). Configure via console.cloud.google.com")
}

func writeToSheet(token, spreadsheetID, rangeStr string, values [][]string) error {
	url := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?valueInputOption=USER_ENTERED",
		spreadsheetID, rangeStr)

	body := map[string]any{
		"range":  rangeStr,
		"values": values,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", url, strings.NewReader(string(jsonBody)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Google Sheets API retornou status %d", resp.StatusCode)
	}
	log.Printf("[sheets] dados exportados para %s", rangeStr)
	return nil
}
