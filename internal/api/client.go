package api

import (
	"fmt"
	"net/http"
	"time"
)

// Client handles HTTP requests to Cursor's API
type Client struct {
	httpClient *http.Client
	baseURL    string
	sessionToken string
}

// NewClient creates a new API client
func NewClient(sessionToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:      "https://cursor.com",
		sessionToken: sessionToken,
	}
}

// makeRequest performs an authenticated HTTP request
func (c *Client) makeRequest(method, endpoint string, body []byte) (*http.Response, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set authentication cookie
	req.Header.Set("Cookie", fmt.Sprintf("WorkosCursorSessionToken=%s", c.sessionToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://cursor.com")
	req.Header.Set("Referer", "https://cursor.com/dashboard")

	if body != nil {
		req.Body = http.NoBody // We'll handle body separately if needed
		req.ContentLength = int64(len(body))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	return resp, nil
}
