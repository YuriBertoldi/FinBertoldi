package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// GoogleClaims contém os dados extraídos do ID token do Google.
type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
}

// VerifyGoogleIDToken verifica o ID token usando o endpoint tokeninfo do Google.
func VerifyGoogleIDToken(idToken string) (*GoogleClaims, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID não configurado")
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(
		"https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken))
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token invalido: %s", string(body))
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("erro parse token: %w", err)
	}

	// Verificar audience
	aud, _ := raw["aud"].(string)
	if aud != clientID {
		return nil, fmt.Errorf("audience invalido: %s", aud)
	}

	var claims GoogleClaims
	claims.Email = strings.ToLower(fmt.Sprint(raw["email"]))
	claims.EmailVerified = fmt.Sprint(raw["email_verified"]) == "true"
	claims.Name = fmt.Sprint(raw["name"])
	claims.Picture = fmt.Sprint(raw["picture"])
	claims.Sub = fmt.Sprint(raw["sub"])

	if !claims.EmailVerified {
		return nil, fmt.Errorf("email não verificado pelo Google")
	}

	return &claims, nil
}
