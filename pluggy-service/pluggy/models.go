package pluggy

import "time"

type AuthResponse struct {
	APIKey string `json:"apiKey"`
}

type ConnectTokenRequest struct {
	ClientUserID string `json:"clientUserId,omitempty"`
	ItemID       string `json:"itemId,omitempty"`
}

type ConnectTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

type Connector struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	InstitutionURL  string `json:"institutionUrl"`
	ImageURL        string `json:"imageUrl"`
	Type            string `json:"type"`
	Country         string `json:"country"`
	IsOpenFinance   bool   `json:"isOpenFinance"`
}

type ConnectorsResponse struct {
	Results    []Connector `json:"results"`
	Total      int         `json:"total"`
	TotalPages int         `json:"totalPages"`
	Page       int         `json:"page"`
}

type Item struct {
	ID            string    `json:"id"`
	Connector     Connector `json:"connector"`
	Status        string    `json:"status"`
	Error         *Error    `json:"error"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Account struct {
	ID          string  `json:"id"`
	ItemID      string  `json:"itemId"`
	Type        string  `json:"type"`
	Subtype     string  `json:"subtype"`
	Name        string  `json:"name"`
	Balance     float64 `json:"balance"`
	CurrencyCode string `json:"currencyCode"`
	Number      string  `json:"number"`
}

type AccountsResponse struct {
	Results []Account `json:"results"`
	Total   int       `json:"total"`
}

type Transaction struct {
	ID              string    `json:"id"`
	AccountID       string    `json:"accountId"`
	Date            time.Time `json:"date"`
	Description     string    `json:"description"`
	DescriptionRaw  string    `json:"descriptionRaw"`
	Amount          float64   `json:"amount"`
	Balance         float64   `json:"balance"`
	CurrencyCode    string    `json:"currencyCode"`
	Category        *string   `json:"category"`
	ProviderCode    string    `json:"providerCode"`
	Status          string    `json:"status"`
	Type            string    `json:"type"`
}

type TransactionsResponse struct {
	Results []Transaction `json:"results"`
	Next    *string       `json:"next"`
}

type WebhookPayload struct {
	Event string `json:"event"`
	ID    string `json:"id"`
}
