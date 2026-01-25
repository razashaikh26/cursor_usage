package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InvoiceResponse represents the response from /api/dashboard/get-monthly-invoice
type InvoiceResponse struct {
	Items                []InvoiceItem `json:"items"`
	HasUnpaidMidMonthInvoice bool `json:"hasUnpaidMidMonthInvoice"`
	UsageEvents          []UsageEvent  `json:"usageEvents,omitempty"` // Only present when includeUsageEvents=true
}

// UsageEvent represents a single usage event from the invoice API
// The API may return field names in different formats (camelCase, with spaces, etc.)
type UsageEvent struct {
	Date                string
	Kind                string
	Model               string
	MaxMode             string
	InputWithCacheWrite int
	InputWithoutCacheWrite int
	CacheRead           int
	OutputTokens        int
	TotalTokens         int
	Cost                float64
}

// UnmarshalJSON implements custom JSON unmarshaling to handle different field name formats
func (e *UsageEvent) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Try different field name variations
	e.Date = getString(raw, "date", "Date")
	e.Kind = getString(raw, "kind", "Kind")
	e.Model = getString(raw, "model", "Model")
	e.MaxMode = getString(raw, "maxMode", "Max Mode", "max_mode")
	
	e.InputWithCacheWrite = getInt(raw, "inputWithCacheWrite", "Input (w/ Cache Write)", "InputWithCacheWrite", "input_with_cache_write")
	e.InputWithoutCacheWrite = getInt(raw, "inputWithoutCacheWrite", "Input (w/o Cache Write)", "InputWithoutCacheWrite", "input_without_cache_write")
	e.CacheRead = getInt(raw, "cacheRead", "Cache Read", "CacheRead", "cache_read")
	e.OutputTokens = getInt(raw, "outputTokens", "Output Tokens", "OutputTokens", "output_tokens")
	e.TotalTokens = getInt(raw, "totalTokens", "Total Tokens", "TotalTokens", "total_tokens")
	
	e.Cost = getFloat64(raw, "cost", "Cost")

	return nil
}

// Helper functions for flexible field access
func getString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

func getInt(m map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case string:
				if i, err := strconv.Atoi(v); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

// InvoiceItem represents a single invoice line item
type InvoiceItem struct {
	Description string `json:"description"`
	Cents       int    `json:"cents"`
}

// InvoiceData represents processed invoice information
type InvoiceData struct {
	Items                []InvoiceItem
	HasUnpaidMidMonthInvoice bool
	TotalOnDemandCents  int
	IsOnDemand          bool
	UsageEvents         []UsageEvent // Individual usage events when includeUsageEvents=true
}

// GetMonthlyInvoice fetches invoice data for a specific month
// GetMonthlyInvoice fetches invoice data for a given month/year
// It can also accept a billing cycle start time for more accurate results
func (c *Client) GetMonthlyInvoice(month, year int) (*InvoiceData, error) {
	return c.GetMonthlyInvoiceWithCycle(month, year, time.Time{})
}

// GetMonthlyInvoiceWithCycle fetches invoice data using billing cycle start time if provided
func (c *Client) GetMonthlyInvoiceWithCycle(month, year int, billingCycleStart time.Time) (*InvoiceData, error) {
	// Based on HAR file analysis, the API accepts two payload formats:
	// 1. Simple: {"month":0,"year":2026}
	// 2. With cycle filter: {"year":2026,"cycleFilterType":"CYCLE_TYPE_START_TIME","startTimeMs":"1768745328000"}
	// We'll try both formats, with the cycle filter format being more accurate
	
	// Use provided billing cycle start time, or estimate it
	var startTimeMs int64
	if !billingCycleStart.IsZero() {
		startTimeMs = billingCycleStart.UnixMilli()
	} else {
		// Estimate billing cycle start (typically around the 18th of the month)
		estimatedStart := time.Date(year, time.Month(month), 18, 0, 0, 0, 0, time.UTC)
		if month == 0 {
			// Month 0 means current month, adjust
			now := time.Now()
			estimatedStart = time.Date(now.Year(), now.Month(), 18, 0, 0, 0, 0, time.UTC)
			if now.Day() < 18 {
				estimatedStart = estimatedStart.AddDate(0, -1, 0)
			}
		}
		startTimeMs = estimatedStart.UnixMilli()
	}
	
	endpoints := []struct {
		path    string
		payload map[string]interface{}
	}{
		// Try the cycle filter format first (more accurate)
		{
			"/api/dashboard/get-monthly-invoice",
			map[string]interface{}{
				"year":            year,
				"cycleFilterType": "CYCLE_TYPE_START_TIME",
				"startTimeMs":     fmt.Sprintf("%d", startTimeMs),
			},
		},
		// Fallback to simple format
		{
			"/api/dashboard/get-monthly-invoice",
			map[string]interface{}{
				"month": month,
				"year":   year,
			},
		},
		// Try with includeUsageEvents (if supported)
		{
			"/api/dashboard/get-monthly-invoice",
			map[string]interface{}{
				"month":              month,
				"year":               year,
				"includeUsageEvents": true,
			},
		},
	}
	
	// Try to get invoice data first
	var invoiceData *InvoiceData
	for _, ep := range endpoints {
		data, err := c.tryGetInvoice(ep.path, ep.payload)
		if err == nil {
			// If we got any data (items or events), use it
			if len(data.Items) > 0 || len(data.UsageEvents) > 0 {
				fmt.Printf("DEBUG: Successfully retrieved data from %s\n", ep.path)
				invoiceData = data
				break
			}
		}
	}
	
	// If we didn't get usage events from invoice endpoint, try the dedicated usage events endpoint
	if invoiceData == nil || len(invoiceData.UsageEvents) == 0 {
		// Calculate date range for billing cycle
		billingCycleEnd := time.Date(year, time.Month(month), 18, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
		if !billingCycleStart.IsZero() {
			billingCycleEnd = billingCycleStart.AddDate(0, 1, 0)
		}
		
		startDateMs := startTimeMs
		endDateMs := billingCycleEnd.UnixMilli()
		
		// Try get-filtered-usage-events endpoint (this is the one that actually returns events!)
		usageEventsData, err := c.getFilteredUsageEvents(startDateMs, endDateMs)
		if err == nil && len(usageEventsData.UsageEvents) > 0 {
			fmt.Printf("DEBUG: Successfully retrieved %d usage events from get-filtered-usage-events\n", len(usageEventsData.UsageEvents))
			if invoiceData == nil {
				invoiceData = usageEventsData
			} else {
				// Merge usage events into invoice data
				invoiceData.UsageEvents = usageEventsData.UsageEvents
			}
		}
	}
	
	if invoiceData == nil {
		// If all attempts returned no data, return empty data structure
		fmt.Printf("DEBUG: All endpoints tried, none returned invoice items or usage events\n")
		return &InvoiceData{
			Items:                []InvoiceItem{},
			HasUnpaidMidMonthInvoice: false,
			TotalOnDemandCents:  0,
			IsOnDemand:          false,
			UsageEvents:         []UsageEvent{},
		}, nil
	}
	
	return invoiceData, nil
}

// tryGetInvoice attempts to fetch invoice data and handles the response
func (c *Client) tryGetInvoice(endpoint string, payload map[string]interface{}) (*InvoiceData, error) {

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Create request with body
	url := c.baseURL + endpoint
	fmt.Printf("DEBUG: Trying endpoint %s with payload: %s\n", url, string(jsonData))
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Mask token in debug output
	tokenPreview := c.sessionToken
	if len(tokenPreview) > 20 {
		tokenPreview = tokenPreview[:20] + "..."
	}
	fmt.Printf("DEBUG: Using session token (preview): %s\n", tokenPreview)

	req.Header.Set("Cookie", fmt.Sprintf("WorkosCursorSessionToken=%s", c.sessionToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://cursor.com")
	req.Header.Set("Referer", "https://cursor.com/dashboard")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: Invoice API response status: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Debug: Log raw response (first 500 chars to avoid huge logs)
	bodyStr := string(body)
	if len(bodyStr) > 500 {
		fmt.Printf("DEBUG: Invoice API response (first 500 chars): %s...\n", bodyStr[:500])
	} else {
		fmt.Printf("DEBUG: Invoice API response: %s\n", bodyStr)
	}

	// Try to parse as flexible JSON first to see structure
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err == nil {
		fmt.Printf("DEBUG: Response keys: %v\n", getKeys(rawResponse))
		
		// Check for items in various locations
		if items, ok := rawResponse["items"].([]interface{}); ok {
			fmt.Printf("DEBUG: Found %d items in raw response\n", len(items))
		}
		if items, ok := rawResponse["invoiceItems"].([]interface{}); ok {
			fmt.Printf("DEBUG: Found %d invoiceItems in raw response\n", len(items))
		}
		if items, ok := rawResponse["lineItems"].([]interface{}); ok {
			fmt.Printf("DEBUG: Found %d lineItems in raw response\n", len(items))
		}
		
		// Check for usage events
		if events, ok := rawResponse["usageEvents"].([]interface{}); ok {
			fmt.Printf("DEBUG: Found %d usageEvents in raw response\n", len(events))
		}
		if events, ok := rawResponse["usage_events"].([]interface{}); ok {
			fmt.Printf("DEBUG: Found %d usage_events (snake_case) in raw response\n", len(events))
		}
		if events, ok := rawResponse["events"].([]interface{}); ok {
			fmt.Printf("DEBUG: Found %d events in raw response\n", len(events))
		}
		
		// Check if we got period info - might need to use these timestamps
		var periodStartMs, periodEndMs string
		if ps, ok := rawResponse["periodStartMs"].(string); ok {
			periodStartMs = ps
			fmt.Printf("DEBUG: Got periodStartMs: %s\n", periodStartMs)
		}
		if pe, ok := rawResponse["periodEndMs"].(string); ok {
			periodEndMs = pe
			fmt.Printf("DEBUG: Got periodEndMs: %s\n", periodEndMs)
		}
		
		// If we have period timestamps but no items, try alternative endpoints
		if periodStartMs != "" && periodEndMs != "" {
			fmt.Printf("DEBUG: Have period timestamps but no items - trying alternative endpoints...\n")
			// Try endpoints that might use period timestamps
			altEndpoints := []string{
				"/api/dashboard/get-usage-events",
				"/api/dashboard/usage-events",
				"/api/dashboard/invoice-items",
			}
			for _, altEndpoint := range altEndpoints {
				altPayload := map[string]interface{}{
					"periodStartMs": periodStartMs,
					"periodEndMs":   periodEndMs,
				}
				altData, err := c.tryGetInvoice(altEndpoint, altPayload)
				if err == nil && (len(altData.Items) > 0 || len(altData.UsageEvents) > 0) {
					fmt.Printf("DEBUG: Successfully got data from %s\n", altEndpoint)
					return altData, nil
				}
			}
		}
		
		// Check for nested data structures
		if data, ok := rawResponse["data"].(map[string]interface{}); ok {
			fmt.Printf("DEBUG: Found nested 'data' object with keys: %v\n", getKeys(data))
			if items, ok := data["items"].([]interface{}); ok {
				fmt.Printf("DEBUG: Found %d items in nested data\n", len(items))
			}
		}
		if invoice, ok := rawResponse["invoice"].(map[string]interface{}); ok {
			fmt.Printf("DEBUG: Found nested 'invoice' object with keys: %v\n", getKeys(invoice))
			if items, ok := invoice["items"].([]interface{}); ok {
				fmt.Printf("DEBUG: Found %d items in nested invoice\n", len(items))
			}
		}
	}

	var invoiceResp InvoiceResponse
	if err := json.Unmarshal(body, &invoiceResp); err != nil {
		return nil, fmt.Errorf("parsing invoice response: %w (body: %s)", err, bodyStr)
	}

	// Debug: Log parsed response
	fmt.Printf("DEBUG: Parsed invoice - %d items, %d usage events\n", 
		len(invoiceResp.Items), len(invoiceResp.UsageEvents))
	
	// If usageEvents is empty but we have items, try to manually parse
	if len(invoiceResp.UsageEvents) == 0 && len(invoiceResp.Items) > 0 {
		fmt.Printf("DEBUG: No usage events parsed, but have %d items. Checking item structure...\n", len(invoiceResp.Items))
		for i, item := range invoiceResp.Items {
			if i < 2 { // Show first 2 items
				fmt.Printf("DEBUG: Item %d: description='%s', cents=%d\n", i, item.Description, item.Cents)
			}
		}
	}

	// Calculate total on-demand spending
	// On-demand items are those that are NOT included in the subscription
	totalOnDemandCents := 0
	for _, item := range invoiceResp.Items {
		// Skip mid-month payment items
		if contains(item.Description, "Mid-month usage paid") {
			continue
		}
		// All invoice items with cents > 0 are on-demand charges
		if item.Cents > 0 {
			totalOnDemandCents += item.Cents
		}
	}

	isOnDemand := totalOnDemandCents > 0

	data := &InvoiceData{
		Items:                invoiceResp.Items,
		HasUnpaidMidMonthInvoice: invoiceResp.HasUnpaidMidMonthInvoice,
		TotalOnDemandCents:  totalOnDemandCents,
		IsOnDemand:          isOnDemand,
		UsageEvents:         invoiceResp.UsageEvents,
	}

	return data, nil
}

// getFilteredUsageEvents fetches usage events from the dedicated endpoint
// This is the endpoint that actually returns detailed usage events!
// Implements pagination to fetch all events, not just the first page
func (c *Client) getFilteredUsageEvents(startDateMs, endDateMs int64) (*InvoiceData, error) {
	endpoint := "/api/dashboard/get-filtered-usage-events"
	pageSize := 100
	page := 1
	var allEvents []UsageEvent
	var totalCount int

	// Loop through pages until all events are fetched
	for {
		payload := map[string]interface{}{
			"teamId":    0,
			"startDate": fmt.Sprintf("%d", startDateMs),
			"endDate":   fmt.Sprintf("%d", endDateMs),
			"page":      page,
			"pageSize":  pageSize,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}

		url := c.baseURL + endpoint
		if page == 1 {
			fmt.Printf("DEBUG: Fetching usage events from %s (page %d)\n", url, page)
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonData))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Cookie", fmt.Sprintf("WorkosCursorSessionToken=%s", c.sessionToken))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Origin", "https://cursor.com")
		req.Header.Set("Referer", "https://cursor.com/dashboard")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("making request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		// Parse response - structure: {"totalUsageEventsCount":808,"usageEventsDisplay":[...]}
		var rawResponse map[string]interface{}
		if err := json.Unmarshal(body, &rawResponse); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		// Get total count on first page
		if page == 1 {
			if totalCountRaw, ok := rawResponse["totalUsageEventsCount"]; ok {
				switch v := totalCountRaw.(type) {
				case int:
					totalCount = v
				case int64:
					totalCount = int(v)
				case float64:
					totalCount = int(v)
				}
				fmt.Printf("DEBUG: Total usage events available: %d\n", totalCount)
			}
		}

		// Extract usageEventsDisplay array
		usageEventsDisplay, ok := rawResponse["usageEventsDisplay"].([]interface{})
		if !ok {
			if page == 1 {
				fmt.Printf("DEBUG: No usageEventsDisplay array found in response\n")
			}
			break
		}

		pageEventCount := len(usageEventsDisplay)
		if page == 1 {
			fmt.Printf("DEBUG: Found %d usage events in page %d\n", pageEventCount, page)
		}

		// Convert to UsageEvent structs
		for _, eventRaw := range usageEventsDisplay {
			eventMap, ok := eventRaw.(map[string]interface{})
			if !ok {
				continue
			}

			// Parse timestamp (milliseconds)
			timestampMs := getInt64(eventMap, "timestamp")
			eventDate := time.Unix(timestampMs/1000, (timestampMs%1000)*1000000).UTC()

			// Parse kind - convert from API format to our format
			kindStr := getString(eventMap, "kind")
			kind := "Included"
			if kindStr == "USAGE_EVENT_KIND_USAGE_BASED" {
				kind = "On-Demand"
			} else if kindStr == "USAGE_EVENT_KIND_ERRORED_NOT_CHARGED" {
				kind = "Errored, No Charge"
			}

			// Parse model
			model := getString(eventMap, "model")

			// Parse token usage
			tokenUsage, _ := eventMap["tokenUsage"].(map[string]interface{})
			inputTokens := getInt(tokenUsage, "inputTokens")
			outputTokens := getInt(tokenUsage, "outputTokens")
			cacheWriteTokens := getInt(tokenUsage, "cacheWriteTokens")
			cacheReadTokens := getInt(tokenUsage, "cacheReadTokens")

			// Calculate total tokens
			totalTokens := inputTokens + outputTokens + cacheWriteTokens + cacheReadTokens

			// Parse cost - Always get the actual cost value, even for included events
			// The dashboard shows the actual price even if it's included in the plan
			var cost float64
			if kind == "On-Demand" {
				// For on-demand events, use usageBasedCosts field (e.g., "$0.43")
				usageBasedCostsStr := getString(eventMap, "usageBasedCosts")
				if usageBasedCostsStr != "" && usageBasedCostsStr != "-" {
					// Parse "$0.43" format
					costStr := strings.TrimPrefix(usageBasedCostsStr, "$")
					if parsedCost, err := strconv.ParseFloat(costStr, 64); err == nil {
						cost = parsedCost
					}
				}
				// If usageBasedCosts is not available, fall back to tokenUsage.totalCents
				if cost == 0 {
					totalCents := getFloat64(tokenUsage, "totalCents")
					cost = totalCents / 100.0
				}
			} else {
				// For included events, try multiple sources for cost
				// The API should provide the actual cost value even for included events
				// Based on dashboard: costs are typically $0.04-$0.47 for included events
				
				// 1. First try usageBasedCosts (this is the most reliable - already in dollars)
				usageBasedCostsStr := getString(eventMap, "usageBasedCosts")
				if usageBasedCostsStr != "" && usageBasedCostsStr != "-" {
					costStr := strings.TrimPrefix(usageBasedCostsStr, "$")
					if parsedCost, err := strconv.ParseFloat(costStr, 64); err == nil {
						cost = parsedCost
					}
				}
				
				// 2. Try tokenUsage.totalCents
				// Based on debug output: tokenUsage.totalCents=19.09 for dashboard cost of $0.19
				// 19.09 / 100 = 0.1909 ≈ $0.19 ✓ CORRECT
				// The field "totalCents" contains values in "hundredths of dollars" (19.09 = $0.1909)
				// Always divide by 100 to convert to dollars
				if cost == 0 && tokenUsage != nil {
					totalCents := getFloat64(tokenUsage, "totalCents")
					if totalCents > 0 {
						// Convert from hundredths to dollars
						cost = totalCents / 100.0
					}
				}
				
				// 3. Try direct cost field in eventMap (might be in dollars already)
				if cost == 0 {
					cost = getFloat64(eventMap, "cost", "Cost", "totalCost", "total_cost", "costUSD")
				}
				
				// Debug output removed - calculation is working correctly
				// tokenUsage.totalCents is in hundredths (19.09 = $0.19), divide by 100
			}

			allEvents = append(allEvents, UsageEvent{
				Date:                eventDate.Format(time.RFC3339),
				Kind:                kind,
				Model:               model,
				MaxMode:             "", // Not in this response
				InputWithCacheWrite: inputTokens + cacheWriteTokens, // Approximate
				InputWithoutCacheWrite: 0, // Not directly available
				CacheRead:           cacheReadTokens,
				OutputTokens:        outputTokens,
				TotalTokens:         totalTokens,
				Cost:                cost,
			})
		}

		// Check if we've fetched all events
		if pageEventCount < pageSize {
			// Last page
			break
		}
		if totalCount > 0 && len(allEvents) >= totalCount {
			// We've fetched all events
			break
		}

		// Move to next page
		page++
	}

	fmt.Printf("DEBUG: Fetched %d total usage events across %d pages\n", len(allEvents), page)

	return &InvoiceData{
		UsageEvents: allEvents,
		Items:       []InvoiceItem{},
	}, nil
}

// getInt64 extracts an int64 value from a map, trying multiple key variations
func getInt64(m map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case int64:
				return v
			case int:
				return int64(v)
			case float64:
				return int64(v)
			case string:
				if i, err := strconv.ParseInt(v, 10, 64); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

// ParseInvoiceItem extracts model name and request count from invoice item description
func ParseInvoiceItem(item InvoiceItem) (modelName string, requestCount int, err error) {
	// Pattern: "32 token-based usage calls to non-max-claude-4.5-opus-high-thinking, totalling: $2.28..."
	// The model name can contain hyphens, dots, underscores, and numbers
	// Match up to the comma before "totalling"
	tokenBasedPattern := regexp.MustCompile(`^(\d+)\s+token-based usage calls to\s+([^,]+?)(?:,\s+totalling|$)`)
	matches := tokenBasedPattern.FindStringSubmatch(item.Description)
	
	if len(matches) >= 3 {
		count, _ := strconv.Atoi(matches[1])
		model := strings.TrimSpace(matches[2])
		return model, count, nil
	}

	// Fallback pattern: "150 claude-4-sonnet requests"
	fallbackPattern := regexp.MustCompile(`^(\d+)\s+([\w.-]+)`)
	matches = fallbackPattern.FindStringSubmatch(item.Description)
	if len(matches) >= 3 {
		count, _ := strconv.Atoi(matches[1])
		return strings.TrimSpace(matches[2]), count, nil
	}

	return "unknown", 0, fmt.Errorf("could not parse invoice item: %s", item.Description)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// getKeys extracts all keys from a map for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
