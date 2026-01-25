package api

import (
	"testing"
)

func TestParseInvoiceItem(t *testing.T) {
	tests := []struct {
		name        string
		item        InvoiceItem
		wantModel   string
		wantCount   int
		wantError   bool
	}{
		{
			name: "token-based format",
			item: InvoiceItem{
				Description: "150 token-based usage calls to claude-4-sonnet, totalling: $12.50",
				Cents:       1250,
			},
			wantModel: "claude-4-sonnet",
			wantCount: 150,
			wantError: false,
		},
		{
			name: "simple format",
			item: InvoiceItem{
				Description: "200 gpt-4o requests",
				Cents:       2000,
			},
			wantModel: "gpt-4o",
			wantCount: 200,
			wantError: false,
		},
		{
			name: "unparseable format",
			item: InvoiceItem{
				Description: "Some random text",
				Cents:       1000,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, count, err := ParseInvoiceItem(tt.item)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseInvoiceItem() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if model != tt.wantModel {
					t.Errorf("ParseInvoiceItem() model = %v, want %v", model, tt.wantModel)
				}
				if count != tt.wantCount {
					t.Errorf("ParseInvoiceItem() count = %v, want %v", count, tt.wantCount)
				}
			}
		})
	}
}

func TestInvoiceDataOnDemand(t *testing.T) {
	tests := []struct {
		name           string
		items          []InvoiceItem
		wantOnDemand   bool
		wantTotalCents int
	}{
		{
			name: "on-demand items",
			items: []InvoiceItem{
				{Description: "100 calls to claude-4-sonnet", Cents: 1000},
				{Description: "50 calls to gpt-4o", Cents: 500},
			},
			wantOnDemand:   true,
			wantTotalCents: 1500,
		},
		{
			name: "mid-month payment excluded",
			items: []InvoiceItem{
				{Description: "Mid-month usage paid", Cents: 1000},
				{Description: "100 calls to claude-4-sonnet", Cents: 500},
			},
			wantOnDemand:   true,
			wantTotalCents: 500,
		},
		{
			name: "no on-demand items",
			items: []InvoiceItem{
				{Description: "Mid-month usage paid", Cents: 1000},
			},
			wantOnDemand:   false,
			wantTotalCents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &InvoiceData{
				Items: tt.items,
			}

			totalCents := 0
			for _, item := range data.Items {
				if contains(item.Description, "Mid-month usage paid") {
					continue
				}
				if item.Cents > 0 {
					totalCents += item.Cents
				}
			}

			isOnDemand := totalCents > 0

			if isOnDemand != tt.wantOnDemand {
				t.Errorf("IsOnDemand = %v, want %v", isOnDemand, tt.wantOnDemand)
			}
			if totalCents != tt.wantTotalCents {
				t.Errorf("TotalOnDemandCents = %v, want %v", totalCents, tt.wantTotalCents)
			}
		})
	}
}
