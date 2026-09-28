package pluggy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const baseURL = "https://api.pluggy.ai"

type Client struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu       sync.Mutex
	apiKey   string
	expireAt time.Time
}

func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) UpdateCredentials(clientID, clientSecret string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clientID != clientID || c.clientSecret != clientSecret {
		c.clientID = clientID
		c.clientSecret = clientSecret
		c.apiKey = ""
		c.expireAt = time.Time{}
	}
}

func (c *Client) authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.apiKey != "" && time.Now().Before(c.expireAt) {
		return nil
	}

	body, _ := json.Marshal(map[string]string{
		"clientId":     c.clientID,
		"clientSecret": c.clientSecret,
	})

	resp, err := c.httpClient.Post(baseURL+"/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pluggy auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pluggy auth failed (%d): %s", resp.StatusCode, string(b))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("pluggy auth decode: %w", err)
	}

	c.apiKey = authResp.APIKey
	c.expireAt = time.Now().Add(110 * time.Minute) // API key valid 2h, refresh early
	return nil
}

func (c *Client) doRequest(method, path string, reqBody any) ([]byte, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if reqBody != nil {
		b, _ := json.Marshal(reqBody)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pluggy request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("pluggy %s %s (%d): %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

// CreateConnectToken generates a connect token for the Pluggy Connect Widget.
func (c *Client) CreateConnectToken(clientUserID string) (string, error) {
	reqBody := ConnectTokenRequest{ClientUserID: clientUserID}
	data, err := c.doRequest("POST", "/connect_token", reqBody)
	if err != nil {
		return "", err
	}
	var resp ConnectTokenResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

// GetItem retrieves a connected item by ID.
func (c *Client) GetItem(itemID string) (*Item, error) {
	data, err := c.doRequest("GET", "/items/"+itemID, nil)
	if err != nil {
		return nil, err
	}
	var item Item
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// ListAccounts retrieves all accounts for an item.
func (c *Client) ListAccounts(itemID string) ([]Account, error) {
	data, err := c.doRequest("GET", "/accounts?itemId="+itemID, nil)
	if err != nil {
		return nil, err
	}
	var resp AccountsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// ListTransactions retrieves transactions for an account within a date range.
func (c *Client) ListTransactions(accountID string, from, to time.Time) ([]Transaction, error) {
	path := fmt.Sprintf("/transactions?accountId=%s&from=%s&to=%s&pageSize=500",
		accountID, from.Format("2006-01-02"), to.Format("2006-01-02"))

	var all []Transaction
	page := 1
	for {
		data, err := c.doRequest("GET", fmt.Sprintf("%s&page=%d", path, page), nil)
		if err != nil {
			return nil, err
		}
		var resp TransactionsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if page >= resp.TotalPages {
			break
		}
		page++
	}
	return all, nil
}

// ListConnectors lists available financial institution connectors.
func (c *Client) ListConnectors() ([]Connector, error) {
	data, err := c.doRequest("GET", "/connectors?sandbox=true", nil)
	if err != nil {
		return nil, err
	}
	var resp ConnectorsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}
