package costs

import (
	"testing"
)

func TestCalculateBYOKCost(t *testing.T) {
	tests := []struct {
		name         string
		modelName    string
		inputTokens  int64
		outputTokens int64
		wantCost     float64
		wantError    bool
	}{
		{
			name:         "claude-4-sonnet",
			modelName:    "claude-4-sonnet",
			inputTokens:  1_000_000,  // 1M tokens
			outputTokens: 500_000,    // 0.5M tokens
			wantCost:     3.00 + 7.50, // $3 input + $7.50 output
			wantError:    false,
		},
		{
			name:         "gpt-4o",
			modelName:    "gpt-4o",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantCost:     2.50 + 10.00, // $2.50 input + $10 output
			wantError:    false,
		},
		{
			name:         "unknown model",
			modelName:    "unknown-model",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := CalculateBYOKCost(tt.modelName, tt.inputTokens, tt.outputTokens)
			if (err != nil) != tt.wantError {
				t.Errorf("CalculateBYOKCost() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Allow small floating point differences
				if cost < tt.wantCost-0.01 || cost > tt.wantCost+0.01 {
					t.Errorf("CalculateBYOKCost() = %.2f, want %.2f", cost, tt.wantCost)
				}
			}
		})
	}
}

func TestCalculateCursorCost(t *testing.T) {
	byokCost := 10.0
	expected := byokCost * CursorMarkup

	cursorCost := CalculateCursorCost(byokCost)
	if cursorCost != expected {
		t.Errorf("CalculateCursorCost() = %.2f, want %.2f", cursorCost, expected)
	}
}

func TestModelProviderDetection(t *testing.T) {
	tests := []struct {
		modelName string
		wantAnthropic bool
		wantOpenAI bool
		wantGoogle bool
	}{
		{"claude-4-sonnet", true, false, false},
		{"claude-3-opus", true, false, false},
		{"gpt-4o", false, true, false},
		{"gpt-4", false, true, false},
		{"o1", false, true, false},
		{"gemini-2.5-pro", false, false, true},
		{"gemini-1.5-flash", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			isAnthropic := isAnthropicModel(tt.modelName)
			isOpenAI := isOpenAIModel(tt.modelName)
			isGoogle := isGoogleModel(tt.modelName)

			if isAnthropic != tt.wantAnthropic {
				t.Errorf("isAnthropicModel(%s) = %v, want %v", tt.modelName, isAnthropic, tt.wantAnthropic)
			}
			if isOpenAI != tt.wantOpenAI {
				t.Errorf("isOpenAIModel(%s) = %v, want %v", tt.modelName, isOpenAI, tt.wantOpenAI)
			}
			if isGoogle != tt.wantGoogle {
				t.Errorf("isGoogleModel(%s) = %v, want %v", tt.modelName, isGoogle, tt.wantGoogle)
			}
		})
	}
}

func TestCompareCosts(t *testing.T) {
	billingCycle := "2026-01-01"
	cursorSpendCents := 2000 // $20.00

	items := []InvoiceItemForCost{
		{
			ModelName:   "claude-4-sonnet",
			InputTokens: 1_000_000,
			OutputTokens: 500_000,
			CostCents:   1000,
		},
		{
			ModelName:   "gpt-4o",
			InputTokens: 500_000,
			OutputTokens: 500_000,
			CostCents:   1000,
		},
	}

	comparison := CompareCosts(billingCycle, cursorSpendCents, items)

	if comparison.BillingCycle != billingCycle {
		t.Errorf("Expected billing cycle %s, got %s", billingCycle, comparison.BillingCycle)
	}

	if comparison.CursorSpendUSD != 20.0 {
		t.Errorf("Expected cursor spend $20.00, got $%.2f", comparison.CursorSpendUSD)
	}

	// Should have calculated BYOK costs
	if comparison.AnthropicDirect <= 0 {
		t.Error("Expected Anthropic direct cost > 0")
	}
	if comparison.OpenAIDirect <= 0 {
		t.Error("Expected OpenAI direct cost > 0")
	}

	// Savings should be positive (Cursor charges more)
	if comparison.AnthropicSavings <= 0 {
		t.Error("Expected positive savings with Anthropic BYOK")
	}
}
