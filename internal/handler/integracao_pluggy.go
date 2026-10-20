package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fincontrol/internal/auth"
	"fincontrol/internal/models"
	"fincontrol/internal/store"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func pluggyHealthCheck(serviceURL string) string {
	if serviceURL == "" {
		return "nao_configurado"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(serviceURL + "/api/pluggy/status")
	if err != nil || resp.StatusCode != http.StatusOK {
		return "offline"
	}
	defer resp.Body.Close()
	return "online"
}

func pluggyProxy(db *sql.DB, method, path string, body any) ([]byte, int, error) {
	cfg, err := store.GetIntegracaoConfig(db, "pluggy")
	if err != nil || !cfg.Ativo {
		return nil, http.StatusServiceUnavailable, fmt.Errorf("pluggy nao configurado")
	}

	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, cfg.ServiceURL+path, reqBody)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("pluggy-service indisponivel: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

func HandleSavePluggyConfig(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cfg := models.IntegracaoConfig{
			Integracao:   "pluggy",
			ClientID:     strings.TrimSpace(r.FormValue("client_id")),
			ClientSecret: strings.TrimSpace(r.FormValue("client_secret")),
			ServiceURL:   strings.TrimSpace(r.FormValue("service_url")),
			Ativo:        r.FormValue("ativo") == "on",
		}
		if err := store.SaveIntegracaoConfig(db, cfg); err != nil {
			log.Printf("Erro ao salvar config pluggy: %v", err)
		}
		http.Redirect(w, r, "/integracoes", http.StatusSeeOther)
	}
}

func HandlePluggyConnectToken(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, status, err := pluggyProxy(db, "POST", "/api/pluggy/connect-token", nil)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
	}
}

func HandlePluggyRegisterItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		var payload struct {
			ItemID string `json:"item_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.ItemID == "" {
			http.Error(w, "item_id obrigatorio", http.StatusBadRequest)
			return
		}

		// Register item in pluggy-service
		body, status, err := pluggyProxy(db, "POST", "/api/pluggy/items", map[string]any{
			"item_id":   payload.ItemID,
			"family_id": u.FamilyID,
		})
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		// Also save locally
		store.CreatePluggyItem(db, u.FamilyID, payload.ItemID, "")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body)
	}
}

func HandlePluggyDisconnect(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := auth.CurrentUser(r)
		itemID := r.PathValue("item_id")
		store.DeletePluggyItem(db, u.FamilyID, itemID)
		http.Redirect(w, r, "/integracoes", http.StatusSeeOther)
	}
}

func HandlePluggySync(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemID := r.PathValue("item_id")
		_, status, err := pluggyProxy(db, "POST", "/api/pluggy/sync/"+itemID, nil)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}
		http.Redirect(w, r, "/integracoes", http.StatusSeeOther)
	}
}

// HandlePluggyWebhook faz proxy do webhook externo para o pluggy-service interno.
// Rota publica (sem auth) — a validacao e feita por token no pluggy-service.
func HandlePluggyWebhook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := store.GetIntegracaoConfig(db, "pluggy")
		if err != nil || cfg.ServiceURL == "" {
			cfg = &models.IntegracaoConfig{ServiceURL: "http://pluggy-service:8081"}
		}

		// Forward token from query string
		targetURL := cfg.ServiceURL + "/api/pluggy/webhook"
		if token := r.URL.Query().Get("token"); token != "" {
			targetURL += "?token=" + token
		}

		proxyReq, err := http.NewRequest("POST", targetURL, r.Body)
		if err != nil {
			http.Error(w, "proxy error", http.StatusBadGateway)
			return
		}
		proxyReq.Header.Set("Content-Type", "application/json")
		// Forward webhook token header if present
		if h := r.Header.Get("X-Webhook-Token"); h != "" {
			proxyReq.Header.Set("X-Webhook-Token", h)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Printf("[webhook-proxy] erro: %v", err)
			http.Error(w, "pluggy-service indisponivel", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
