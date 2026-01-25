package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UsageResponse represents the response from /api/usage
type UsageResponse struct {
	GPT4 struct {
		NumRequests      int  `json:"numRequests"`
		NumRequestsTotal int  `json:"numRequestsTotal"`
		MaxRequestUsage  *int `json:"maxRequestUsage"` // Can be null
		MaxTokenUsage    *int `json:"maxTokenUsage"`   // Can be null
		NumTokens        int  `json:"numTokens"`
	} `json:"gpt-4"`
	GPT35Turbo struct {
		NumRequests    int  `json:"numRequests"`
		MaxRequestUsage *int `json:"maxRequestUsage"`
		NumTokens      int  `json:"numTokens"`
	} `json:"gpt-3.5-turbo"`
	StartOfMonth string `json:"startOfMonth"`
}

// UsageData represents processed usage information
type UsageData struct {
	PremiumRequestsUsed int
	PremiumRequestsLimit int
	UsagePercentage     float64
	BillingCycleStart   time.Time
	IsOnDemand          bool
	OnDemandSpendCents  int
	RawResponse         string
}

// GetUsage fetches current usage data from the API
func (c *Client) GetUsage(userID string) (*UsageData, error) {
	endpoint := fmt.Sprintf("/api/usage?user=%s", userID)
	resp, err := c.makeRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Debug: Log raw response
	bodyStr := string(body)
	if len(bodyStr) > 500 {
		fmt.Printf("DEBUG: Usage API response (first 500 chars): %s...\n", bodyStr[:500])
	} else {
		fmt.Printf("DEBUG: Usage API response: %s\n", bodyStr)
	}

	var usageResp UsageResponse
	if err := json.Unmarshal(body, &usageResp); err != nil {
		return nil, fmt.Errorf("parsing usage response: %w (body: %s)", err, bodyStr)
	}

	// Parse billing cycle start
	billingStart, err := time.Parse(time.RFC3339, usageResp.StartOfMonth)
	if err != nil {
		return nil, fmt.Errorf("parsing startOfMonth: %w", err)
	}

	// Calculate usage percentage
	// Handle null maxRequestUsage (Pro+ plans may have different limits or unlimited)
	var limit int
	if usageResp.GPT4.MaxRequestUsage != nil {
		limit = *usageResp.GPT4.MaxRequestUsage
	}
	
	// If limit is 0 or null, try to infer from Pro+ plan structure
	// Pro+ typically has $20/month included usage, which translates to ~500 requests
	// But we should check invoice data for actual limits
	if limit == 0 {
		// Default to 500 for Pro plan, but this might need adjustment based on actual plan
		limit = 500
	}
	
	// Use numRequestsTotal if available (includes all request types), otherwise use numRequests
	requestsUsed := usageResp.GPT4.NumRequestsTotal
	if requestsUsed == 0 {
		requestsUsed = usageResp.GPT4.NumRequests
	}
	
	var usagePct float64
	if limit > 0 {
		usagePct = float64(requestsUsed) / float64(limit) * 100
	} else {
		// Unlimited or unknown limit - show usage count only
		usagePct = 0.0
	}

	// Determine if on-demand (when usage exceeds limit or when there's on-demand spending)
	// For now, we'll check invoice data separately to determine on-demand status
	isOnDemand := requestsUsed >= limit

	data := &UsageData{
		PremiumRequestsUsed:  requestsUsed,
		PremiumRequestsLimit:  limit,
		UsagePercentage:      usagePct,
		BillingCycleStart:    billingStart,
		IsOnDemand:           isOnDemand,
		OnDemandSpendCents:   0, // Will be updated from invoice data
		RawResponse:          string(body),
	}

	return data, nil
}
