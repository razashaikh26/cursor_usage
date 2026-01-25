package monitor

import (
	"context"
	"fmt"
	"time"

	"cursor-monitor/internal/api"
	"cursor-monitor/internal/storage"
)

// FetchHistoricalInvoiceData fetches invoice data for previous billing cycles
// This is useful for backfilling data when the service first starts
func (m *Monitor) FetchHistoricalInvoiceData(ctx context.Context, billingCycleStart time.Time, monthsBack int) error {
	m.logger.Printf("Fetching historical invoice data for last %d months...", monthsBack)

	// Ensure apiClient is initialized before use
	if m.apiClient == nil {
		// Get session token to initialize API client
		sessionToken, err := m.auth.GetToken()
		if err != nil {
			return fmt.Errorf("getting session token for historical fetch: %w", err)
		}
		m.apiClient = api.NewClient(sessionToken)
	}

	for i := 0; i < monthsBack; i++ {
		// Calculate the date for this month
		targetDate := billingCycleStart.AddDate(0, -i, 0)
		month := int(targetDate.Month())
		year := targetDate.Year()

		// Fetch invoice for this month
		invoiceData, err := m.apiClient.GetMonthlyInvoice(month, year)
		if err != nil {
			m.logger.Printf("Warning: Could not fetch historical invoice for %d/%d: %v", month, year, err)
			continue
		}

		// Save invoice items
		billingCycleStr := targetDate.Format("2006-01-02")
		if len(invoiceData.Items) > 0 {
			var items []storage.InvoiceItem
			for _, item := range invoiceData.Items {
				modelName, requestCount, err := api.ParseInvoiceItem(item)
				if err != nil {
					continue
				}

				items = append(items, storage.InvoiceItem{
					BillingCycle: billingCycleStr,
					ModelName:    modelName,
					RequestCount: requestCount,
					CostCents:    item.Cents,
					IsDiscounted: false,
					FetchedAt:    time.Now(),
				})
			}

			if len(items) > 0 {
				if err := m.storage.SaveInvoiceItems(billingCycleStr, items); err != nil {
					m.logger.Printf("Warning: Could not save historical invoice items: %v", err)
				} else {
					m.logger.Printf("Saved %d invoice items for billing cycle %s", len(items), billingCycleStr)
				}
			}
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}
