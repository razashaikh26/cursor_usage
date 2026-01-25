package costs

import (
	"fmt"
	"regexp"
	"strings"
)

// ModelPricing holds per-model pricing in dollars per 1M tokens
type ModelPricing struct {
	Input  float64
	Output float64
}

// BYOKPricing holds pricing data for BYOK models
// Pricing is per million tokens (Input/Output)
// Sources: Official API documentation as of 2025
// Note: Extended context pricing (>200K tokens) not included - use standard pricing as approximation
var BYOKPricing = map[string]map[string]ModelPricing{
	"anthropic": {
		// Claude 4.5 Series (Latest - November 2025)
		"claude-4.5-opus":               {Input: 5.00, Output: 25.00},  // 67% reduction from 4.1
		"claude-4.5-opus-high-thinking": {Input: 5.00, Output: 25.00},
		"claude-4.5-sonnet":             {Input: 3.00, Output: 15.00},  // Standard: $3/$15, Extended (>200K): $6/$22.50
		"claude-4.5-sonnet-thinking":    {Input: 3.00, Output: 15.00},
		"claude-4.5-haiku":             {Input: 1.00, Output: 5.00},
		// Claude 4 Series
		"claude-4-opus":                 {Input: 15.00, Output: 75.00},
		"claude-4-opus-high-thinking":   {Input: 15.00, Output: 75.00},
		"claude-4-sonnet":              {Input: 3.00, Output: 15.00},
		"claude-4-sonnet-thinking":      {Input: 3.00, Output: 15.00},
		// Claude 3.5 Series
		"claude-3.5-sonnet":             {Input: 3.00, Output: 15.00},
		"claude-3.5-opus":               {Input: 15.00, Output: 75.00},
		// Claude 3 Series
		"claude-3-opus":                 {Input: 15.00, Output: 75.00},
		"claude-3-sonnet":               {Input: 3.00, Output: 15.00},
		"claude-3-haiku":                {Input: 0.25, Output: 1.25},
	},
	"openai": {
		// GPT-5 Series (Latest 2025)
		"gpt-5.2":      {Input: 1.75, Output: 14.00},
		"gpt-5.2-pro":  {Input: 21.00, Output: 168.00},
		"gpt-5-mini":   {Input: 0.25, Output: 2.00},
		// GPT-4 Series
		"gpt-4o":       {Input: 2.50, Output: 10.00},
		"gpt-4o-mini":    {Input: 0.15, Output: 0.60},
		"gpt-4":          {Input: 30.00, Output: 60.00},
		"gpt-4-turbo":    {Input: 10.00, Output: 30.00},
		// O1 Series (Reasoning models)
		"o1":            {Input: 15.00, Output: 60.00},
		"o1-mini":       {Input: 3.00, Output: 12.00},
		"o1-preview":    {Input: 15.00, Output: 60.00},
	},
	"google": {
		// Gemini 3 Series (Latest)
		"gemini-3-pro":         {Input: 2.00, Output: 12.00},  // ≤200K tokens, >200K: $4/$18
		"gemini-3-pro-preview": {Input: 2.00, Output: 12.00},
		// Gemini 2.5 Series
		"gemini-2.5-pro":       {Input: 1.25, Output: 10.00},  // Estimated, competitive pricing
		// Gemini 2.0 Series
		"gemini-2.0-flash":     {Input: 0.10, Output: 0.40},
		"gemini-2.0-pro":       {Input: 1.25, Output: 10.00},  // Estimated
		// Gemini 1.5 Series
		"gemini-1.5-pro":       {Input: 1.25, Output: 5.00},
		"gemini-1.5-flash":     {Input: 0.075, Output: 0.30},
		"gemini-1.5-flash-8b":  {Input: 0.075, Output: 0.30},  // Same as flash
	},
}

// CursorMarkup is the estimated markup over direct API pricing (~18%)
const CursorMarkup = 1.18

// CalculateBYOKCost calculates what the cost would be with BYOK for a given model and usage
func CalculateBYOKCost(modelName string, inputTokens, outputTokens int64) (float64, error) {
	// Find the model in pricing data (try exact match first, then fuzzy match)
	var pricing ModelPricing
	var found bool

	// First try exact match
	for _, provider := range BYOKPricing {
		if p, ok := provider[modelName]; ok {
			pricing = p
			found = true
			break
		}
	}

	// If not found, try fuzzy matching (handle version differences)
	if !found {
		modelLower := strings.ToLower(modelName)
		for _, provider := range BYOKPricing {
			for key, p := range provider {
				keyLower := strings.ToLower(key)
				// Match if model name contains key or vice versa (handles version differences)
				// e.g., "claude-4.5-opus-high-thinking" matches "claude-4-opus-high-thinking"
				if strings.Contains(modelLower, keyLower) || strings.Contains(keyLower, modelLower) {
					// More specific: check if it's the same model family
					if isSameModelFamily(modelLower, keyLower) {
						pricing = p
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}
	}

	if !found {
		return 0, fmt.Errorf("model %s not found in BYOK pricing data", modelName)
	}

	// Calculate cost: (inputTokens / 1M) * inputPrice + (outputTokens / 1M) * outputPrice
	inputCost := (float64(inputTokens) / 1_000_000) * pricing.Input
	outputCost := (float64(outputTokens) / 1_000_000) * pricing.Output

	return inputCost + outputCost, nil
}

// CalculateCursorCost estimates what Cursor charges (with markup)
func CalculateCursorCost(byokCost float64) float64 {
	return byokCost * CursorMarkup
}

// CostComparison holds comparison data for a billing cycle
type CostComparison struct {
	BillingCycle      string
	CursorSpendUSD    float64
	AnthropicDirect   float64
	OpenAIDirect      float64
	GoogleDirect      float64
	AnthropicSavings  float64
	OpenAISavings     float64
	GoogleSavings     float64
}

// CompareCosts compares Cursor costs with BYOK costs for a set of invoice items
func CompareCosts(billingCycle string, cursorSpendCents int, invoiceItems []InvoiceItemForCost) CostComparison {
	cursorSpend := float64(cursorSpendCents) / 100.0

	comparison := CostComparison{
		BillingCycle:   billingCycle,
		CursorSpendUSD: cursorSpend,
	}

	// Calculate BYOK costs by provider
	for _, item := range invoiceItems {
		byokCost, err := CalculateBYOKCost(item.ModelName, item.InputTokens, item.OutputTokens)
		if err != nil {
			// Skip models we don't have pricing for (e.g., "default", "agent_review")
			continue
		}

		// Determine provider
		if isAnthropicModel(item.ModelName) {
			comparison.AnthropicDirect += byokCost
		} else if isOpenAIModel(item.ModelName) {
			comparison.OpenAIDirect += byokCost
		} else if isGoogleModel(item.ModelName) {
			comparison.GoogleDirect += byokCost
		}
	}

	// Calculate savings
	// Only show savings if there's actual usage from that provider
	// If no usage, savings should be 0 (not cursorSpend)
	if comparison.AnthropicDirect > 0 {
		comparison.AnthropicSavings = cursorSpend - comparison.AnthropicDirect
	} else {
		comparison.AnthropicSavings = 0
	}
	if comparison.OpenAIDirect > 0 {
		comparison.OpenAISavings = cursorSpend - comparison.OpenAIDirect
	} else {
		comparison.OpenAISavings = 0
	}
	if comparison.GoogleDirect > 0 {
		comparison.GoogleSavings = cursorSpend - comparison.GoogleDirect
	} else {
		comparison.GoogleSavings = 0
	}

	return comparison
}

// InvoiceItemForCost represents an invoice item with token usage
type InvoiceItemForCost struct {
	ModelName   string
	InputTokens int64
	OutputTokens int64
	CostCents   int
}

// Helper functions to identify model providers
func isAnthropicModel(modelName string) bool {
	return contains(modelName, "claude")
}

func isOpenAIModel(modelName string) bool {
	return contains(modelName, "gpt") || contains(modelName, "o1")
}

func isGoogleModel(modelName string) bool {
	return contains(modelName, "gemini")
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// isSameModelFamily checks if two model names are from the same model family
// e.g., "claude-4.5-opus-high-thinking" and "claude-4-opus-high-thinking" are the same family
func isSameModelFamily(model1, model2 string) bool {
	// Extract key parts: model type (opus/sonnet/haiku) and variant (thinking, etc.)
	// Remove version numbers and compare core parts
	normalize := func(s string) string {
		// Remove version patterns like "4.5", "3.5", "4", "3"
		s = regexp.MustCompile(`-\d+\.?\d*-`).ReplaceAllString(s, "-")
		s = regexp.MustCompile(`^claude-\d+\.?\d*-`).ReplaceAllString(s, "claude-")
		// Keep only the core model identifier (opus, sonnet, haiku) and variant
		parts := strings.Split(s, "-")
		var keyParts []string
		for _, part := range parts {
			if part == "opus" || part == "sonnet" || part == "haiku" || 
			   part == "thinking" || part == "high" {
				keyParts = append(keyParts, part)
			}
		}
		return strings.Join(keyParts, "-")
	}
	norm1 := normalize(model1)
	norm2 := normalize(model2)
	// Match if normalized versions are similar (one contains the other)
	return norm1 != "" && norm2 != "" && (strings.Contains(norm1, norm2) || strings.Contains(norm2, norm1))
}
