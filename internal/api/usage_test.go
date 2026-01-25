package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUsageResponseParsing(t *testing.T) {
	jsonData := `{
		"gpt-4": {
			"numRequests": 46,
			"maxRequestUsage": 500,
			"numTokens": 125000
		},
		"gpt-3.5-turbo": {
			"numRequests": 10,
			"maxRequestUsage": null
		},
		"startOfMonth": "2026-01-01T00:00:00.000Z"
	}`

	var resp UsageResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to parse usage response: %v", err)
	}

	if resp.GPT4.NumRequests != 46 {
		t.Errorf("Expected 46 requests, got %d", resp.GPT4.NumRequests)
	}

	if resp.GPT4.MaxRequestUsage != 500 {
		t.Errorf("Expected limit 500, got %d", resp.GPT4.MaxRequestUsage)
	}

	if resp.StartOfMonth != "2026-01-01T00:00:00.000Z" {
		t.Errorf("Expected startOfMonth '2026-01-01T00:00:00.000Z', got %s", resp.StartOfMonth)
	}
}

func TestUsageDataCalculation(t *testing.T) {
	resp := UsageResponse{
		GPT4: struct {
			NumRequests    int `json:"numRequests"`
			MaxRequestUsage int `json:"maxRequestUsage"`
			NumTokens      int `json:"numTokens"`
		}{
			NumRequests:    375,
			MaxRequestUsage: 500,
			NumTokens:      125000,
		},
		StartOfMonth: "2026-01-01T00:00:00.000Z",
	}

	billingStart, _ := time.Parse(time.RFC3339, resp.StartOfMonth)
	limit := resp.GPT4.MaxRequestUsage
	if limit == 0 {
		limit = 500
	}
	usagePct := float64(resp.GPT4.NumRequests) / float64(limit) * 100

	expectedPct := 75.0
	if usagePct != expectedPct {
		t.Errorf("Expected usage percentage %.1f%%, got %.1f%%", expectedPct, usagePct)
	}

	if billingStart.Year() != 2026 || billingStart.Month() != time.January {
		t.Errorf("Billing cycle start parsed incorrectly")
	}
}
