package costs

import (
	"fmt"
	"strings"
)

// ModelPricing holds per-model pricing in dollars per 1M tokens
type ModelPricing struct {
	Input  float64
	Output float64
}

// BYOKPricing holds pricing data for BYOK models
var BYOKPricing = map[string]map[string]ModelPricing{
	"anthropic": {
		"claude-4-sonnet":          {Input: 3.00, Output: 15.00},
		"claude-4-sonnet-thinking": {Input: 3.00, Output: 15.00},
		"claude-3.5-sonnet":        {Input: 3.00, Output: 15.00},
		"claude-3-opus":            {Input: 15.00, Output: 75.00},
		"claude-3-sonnet":          {Input: 3.00, Output: 15.00},
		"claude-3-haiku":           {Input: 0.25, Output: 1.25},
	},
	"openai": {
		"gpt-4o":      {Input: 2.50, Output: 10.00},
		"gpt-4o-mini": {Input: 0.15, Output: 0.60},
		"gpt-4":       {Input: 30.00, Output: 60.00},
		"gpt-4-turbo": {Input: 10.00, Output: 30.00},
		"o1":          {Input: 15.00, Output: 60.00},
		"o1-mini":     {Input: 3.00, Output: 12.00},
	},
	"google": {
		"gemini-2.5-pro":   {Input: 1.25, Output: 10.00},
		"gemini-2.0-flash": {Input: 0.10, Output: 0.40},
		"gemini-1.5-pro":   {Input: 1.25, Output: 5.00},
		"gemini-1.5-flash": {Input: 0.075, Output: 0.30},
	},
}

// CursorMarkup is the estimated markup over direct API pricing (~18%)
const CursorMarkup = 1.18

// CalculateBYOKCost calculates what the cost would be with BYOK for a given model and usage
func CalculateBYOKCost(modelName string, inputTokens, outputTokens int64) (float64, error) {
	// Find the model in pricing data
	var pricing ModelPricing
	var found bool

	for _, provider := range BYOKPricing {
		if p, ok := provider[modelName]; ok {
			pricing = p
			found = true
			break
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
			// Skip models we don't have pricing for
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
	comparison.AnthropicSavings = cursorSpend - comparison.AnthropicDirect
	comparison.OpenAISavings = cursorSpend - comparison.OpenAIDirect
	comparison.GoogleSavings = cursorSpend - comparison.GoogleDirect

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
