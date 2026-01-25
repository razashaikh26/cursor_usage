package monitor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cursor-monitor/internal/alerts"
	"cursor-monitor/internal/api"
	"cursor-monitor/internal/auth"
	"cursor-monitor/internal/config"
	"cursor-monitor/internal/storage"
)

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Monitor handles the polling loop and coordinates all components
type Monitor struct {
	config      *config.Config
	auth        *auth.Auth
	apiClient   *api.Client
	storage     *storage.Storage
	alertEngine *alerts.AlertEngine
	logger      *log.Logger
}

// New creates a new Monitor instance
func New(cfg *config.Config, logger *log.Logger) (*Monitor, error) {
	// Initialize auth
	authInstance, err := auth.New()
	if err != nil {
		return nil, fmt.Errorf("initializing auth: %w", err)
	}

	// Initialize storage
	store, err := storage.New(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("initializing storage: %w", err)
	}

	// Initialize alert engine
	alertEngine := alerts.New(cfg.Alerts.Sound)

	return &Monitor{
		config:      cfg,
		auth:        authInstance,
		storage:     store,
		alertEngine: alertEngine,
		logger:      logger,
	}, nil
}

// Close closes all resources
func (m *Monitor) Close() error {
	if m.storage != nil {
		return m.storage.Close()
	}
	return nil
}

// Run starts the monitoring loop
func (m *Monitor) Run(ctx context.Context) error {
	ticker := time.NewTicker(m.config.PollInterval())
	defer ticker.Stop()

	// Check if this is first run (no snapshots in database)
	latestSnapshot, _ := m.storage.GetLatestSnapshot()
	if latestSnapshot == nil {
		m.logger.Println("First run detected - fetching historical invoice data...")
		// Get current billing cycle from initial poll
		if err := m.poll(ctx); err != nil {
			m.logger.Printf("Initial poll failed: %v", err)
		} else {
			// Fetch historical data for last 3 billing cycles
			latestSnapshot, _ = m.storage.GetLatestSnapshot()
			if latestSnapshot != nil {
				if err := m.FetchHistoricalInvoiceData(ctx, latestSnapshot.BillingCycleStart, 3); err != nil {
					m.logger.Printf("Warning: Could not fetch historical data: %v", err)
				}
			}
		}
	} else {
		// Initial poll
		m.logger.Println("Starting initial poll...")
		if err := m.poll(ctx); err != nil {
			m.logger.Printf("Initial poll failed: %v", err)
			// Continue anyway - don't fail on first poll
		}
	}

	for {
		select {
		case <-ctx.Done():
			m.logger.Println("Shutting down monitor")
			return ctx.Err()
		case <-ticker.C:
			if err := m.poll(ctx); err != nil {
				m.logger.Printf("Poll failed: %v", err)
				// Don't crash, continue polling
			}
		}
	}
}

// Poll performs a single polling cycle (exposed for manual refresh)
func (m *Monitor) Poll(ctx context.Context) error {
	return m.poll(ctx)
}

// poll performs a single polling cycle
func (m *Monitor) poll(ctx context.Context) error {
	// Get session token
	sessionToken, err := m.auth.GetToken()
	if err != nil {
		// Try refreshing token
		m.auth.RefreshToken()
		sessionToken, err = m.auth.GetToken()
		if err != nil {
			return fmt.Errorf("getting session token: %w", err)
		}
	}

	// Create/update API client
	m.apiClient = api.NewClient(sessionToken)

	// Extract user ID from token (format: userId%3A%3AjwtToken)
	parts := strings.Split(sessionToken, "%3A%3A")
	if len(parts) < 2 {
		return fmt.Errorf("invalid session token format")
	}
	userID := parts[0]

	// Fetch usage data
	usageData, err := m.apiClient.GetUsage(userID)
	if err != nil {
		// If 401, refresh token and retry once
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "Unauthorized") {
			m.auth.RefreshToken()
			sessionToken, err = m.auth.GetToken()
			if err != nil {
				return fmt.Errorf("getting token after refresh failed: %w", err)
			}
			m.apiClient = api.NewClient(sessionToken)
			parts = strings.Split(sessionToken, "%3A%3A")
			if len(parts) >= 2 {
				userID = parts[0]
			}
			usageData, err = m.apiClient.GetUsage(userID)
			if err != nil {
				return fmt.Errorf("fetching usage after token refresh: %w", err)
			}
		} else {
			return fmt.Errorf("fetching usage: %w", err)
		}
	}

	// Fetch invoice data for the billing cycle
	// The billing cycle might not align with calendar months (e.g., Jan 18 - Feb 18)
	// Try fetching for the month containing the billing cycle start, and also current month
	billingMonth := int(usageData.BillingCycleStart.Month())
	billingYear := usageData.BillingCycleStart.Year()
	now := time.Now()
	currentMonth := int(now.Month())
	currentYear := now.Year()
	
	m.logger.Printf("Billing cycle starts: %s (fetching invoice for %d/%d)", 
		usageData.BillingCycleStart.Format("2006-01-02"), billingMonth, billingYear)
	
	// Try billing cycle month first, using the actual billing cycle start time
	invoiceData, err := m.apiClient.GetMonthlyInvoiceWithCycle(billingMonth, billingYear, usageData.BillingCycleStart)
	if err != nil {
		m.logger.Printf("Warning: Could not fetch invoice for billing cycle month %d/%d: %v", 
			billingMonth, billingYear, err)
		// Try current month as fallback
		if billingMonth != currentMonth || billingYear != currentYear {
			m.logger.Printf("Trying current month %d/%d as fallback...", currentMonth, currentYear)
			invoiceData, err = m.apiClient.GetMonthlyInvoice(currentMonth, currentYear)
		}
		if err != nil {
			m.logger.Printf("Warning: Could not fetch invoice data: %v", err)
			invoiceData = &api.InvoiceData{
				IsOnDemand:         false,
				TotalOnDemandCents: 0,
			}
		}
	}
	
	// Log what we received
	if invoiceData != nil {
		m.logger.Printf("Invoice data: %d items, %d usage events, on-demand: $%.2f", 
			len(invoiceData.Items), len(invoiceData.UsageEvents), 
			float64(invoiceData.TotalOnDemandCents)/100.0)
	}

	// Calculate running total from invoice items (more accurate than snapshot data)
	// This gives us the true cumulative on-demand spend for the billing cycle
	runningOnDemandCents := invoiceData.TotalOnDemandCents
	
	// Also check if we have stored invoice items and sum them for running total
	billingCycleStr := usageData.BillingCycleStart.Format("2006-01-02")
	storedItems, err := m.storage.GetInvoiceItemsForCycle(billingCycleStr)
	if err == nil && len(storedItems) > 0 {
		// Sum all invoice items for this billing cycle
		storedTotal := 0
		for _, item := range storedItems {
			storedTotal += item.CostCents
		}
		// Use the maximum (invoice API total or stored total) to ensure we have the latest
		if storedTotal > runningOnDemandCents {
			runningOnDemandCents = storedTotal
		}
	}

	// Update on-demand status from invoice data
	usageData.IsOnDemand = invoiceData.IsOnDemand
	usageData.OnDemandSpendCents = runningOnDemandCents

	// billingCycleStr already declared above, so we'll use it when saving events
	// For now, handle the case where API shows 0 but we have events
	if usageData.IsOnDemand && usageData.PremiumRequestsUsed == 0 {
		// Try to get usage events from storage if not in invoiceData
		var usageEvents []api.UsageEvent
		if len(invoiceData.UsageEvents) > 0 {
			usageEvents = invoiceData.UsageEvents
		} else {
			// Try to get from storage
			storedEvents, err := m.storage.GetUsageEventsForCycle(billingCycleStr)
			if err == nil {
				// Convert storage events to API events for counting
				for _, se := range storedEvents {
					usageEvents = append(usageEvents, api.UsageEvent{
						Kind: se.Kind,
					})
				}
			}
		}
		
		// Count included events to determine actual usage
		includedCount := 0
		for _, event := range usageEvents {
			if event.Kind == "Included" {
				includedCount++
			}
		}
		
		// If we have included events, use that as the actual usage
		if includedCount > 0 {
			usageData.PremiumRequestsUsed = includedCount
			// Recalculate percentage
			if usageData.PremiumRequestsLimit > 0 {
				usageData.UsagePercentage = float64(includedCount) / float64(usageData.PremiumRequestsLimit) * 100
			} else {
				usageData.UsagePercentage = 100.0 // At least 100% if we're on-demand
			}
			m.logger.Printf("Calculated usage from events: %d included requests (%.1f%%)", includedCount, usageData.UsagePercentage)
		} else {
			// No included events found, but we're on-demand - must have exceeded limit
			// Set to at least 100% to reflect that we've used all included credits
			usageData.UsagePercentage = 100.0
			// Estimate usage as limit (we've at least used all included credits)
			usageData.PremiumRequestsUsed = usageData.PremiumRequestsLimit
			m.logger.Printf("On-demand with no included events - setting usage to 100%% (limit: %d)", usageData.PremiumRequestsLimit)
		}
	} else if usageData.IsOnDemand && usageData.PremiumRequestsUsed < usageData.PremiumRequestsLimit {
		// On-demand but usage shows less than limit - this is inconsistent
		// Set to at least 100% since we've exceeded the limit
		usageData.UsagePercentage = 100.0
		usageData.PremiumRequestsUsed = usageData.PremiumRequestsLimit
		m.logger.Printf("On-demand with inconsistent usage (%d < %d) - setting to 100%%", usageData.PremiumRequestsUsed, usageData.PremiumRequestsLimit)
	}

	// Create snapshot
	snapshot := &storage.UsageSnapshot{
		Timestamp:            time.Now(),
		BillingCycleStart:   usageData.BillingCycleStart,
		PremiumRequestsUsed:  usageData.PremiumRequestsUsed,
		PremiumRequestsLimit: usageData.PremiumRequestsLimit,
		UsagePercentage:     usageData.UsagePercentage,
		IsOnDemand:          usageData.IsOnDemand,
		OnDemandSpendCents:  usageData.OnDemandSpendCents,
		RawResponse:         usageData.RawResponse,
	}

	// Save snapshot
	if err := m.storage.SaveUsageSnapshot(snapshot); err != nil {
		return fmt.Errorf("saving snapshot: %w", err)
	}

	// Get previous snapshot for comparison
	previousSnapshot, err := m.storage.GetPreviousSnapshot()
	if err != nil {
		m.logger.Printf("Warning: Could not get previous snapshot: %v", err)
	}

	// Check alerts (billingCycleStr already declared above)

	// Threshold alerts
	if err := m.alertEngine.CheckThresholdAlert(
		usageData.UsagePercentage,
		m.config.Alerts.Thresholds,
		billingCycleStr,
		m.storage.AlertAlreadySent,
		m.storage.RecordAlert,
	); err != nil {
		m.logger.Printf("Error checking threshold alerts: %v", err)
	}

	// On-demand switch alert (critical)
	if m.config.Alerts.OnDemandCritical && previousSnapshot != nil {
		if err := m.alertEngine.CheckOnDemandSwitch(
			usageData.IsOnDemand,
			previousSnapshot.IsOnDemand,
			usageData.OnDemandSpendCents,
		); err != nil {
			m.logger.Printf("Error checking on-demand switch: %v", err)
		}
	}

	// Save invoice items if available
	if invoiceData != nil && len(invoiceData.Items) > 0 {
		var items []storage.InvoiceItem
		for _, item := range invoiceData.Items {
			modelName, requestCount, err := api.ParseInvoiceItem(item)
			if err != nil {
				// Skip items we can't parse
				continue
			}

			items = append(items, storage.InvoiceItem{
				BillingCycle: billingCycleStr,
				ModelName:    modelName,
				RequestCount: requestCount,
				CostCents:    item.Cents,
				IsDiscounted: false, // Would need to parse from description
				FetchedAt:    time.Now(),
			})
		}

		if len(items) > 0 {
			if err := m.storage.SaveInvoiceItems(billingCycleStr, items); err != nil {
				m.logger.Printf("Warning: Could not save invoice items: %v", err)
			}
		}
	}

	// Save usage events if available
	if invoiceData != nil && len(invoiceData.UsageEvents) > 0 {
		var events []storage.UsageEvent
		for _, event := range invoiceData.UsageEvents {
			// Parse event date
			eventDate, err := time.Parse(time.RFC3339, event.Date)
			if err != nil {
				// Try alternative format if RFC3339 fails
				eventDate, err = time.Parse("2006-01-02T15:04:05.000Z", event.Date)
				if err != nil {
					m.logger.Printf("Warning: Could not parse event date %s: %v", event.Date, err)
					continue
				}
			}

			events = append(events, storage.UsageEvent{
				EventDate:             eventDate,
				BillingCycle:          billingCycleStr,
				Kind:                  event.Kind,
				Model:                 event.Model,
				MaxMode:               event.MaxMode,
				InputWithCacheWrite:   event.InputWithCacheWrite,
				InputWithoutCacheWrite: event.InputWithoutCacheWrite,
				CacheRead:             event.CacheRead,
				OutputTokens:          event.OutputTokens,
				TotalTokens:           event.TotalTokens,
				Cost:                  event.Cost,
				FetchedAt:             time.Now(),
			})
		}

		if len(events) > 0 {
			if err := m.storage.SaveUsageEvents(billingCycleStr, events); err != nil {
				m.logger.Printf("Warning: Could not save usage events: %v", err)
			} else {
				m.logger.Printf("Saved %d usage events for billing cycle %s", len(events), billingCycleStr)
				
				// Recalculate usage from events (more accurate than API)
				stats, err := m.storage.CalculateUsageStats(billingCycleStr)
				if err == nil {
					// Update usage data with accurate counts from events
					usageData.PremiumRequestsUsed = stats.IncludedRequests
					if usageData.PremiumRequestsLimit > 0 {
						usageData.UsagePercentage = float64(stats.IncludedRequests) / float64(usageData.PremiumRequestsLimit) * 100
					}
					
					// If we're on-demand, we've used at least the included amount
					if usageData.IsOnDemand && m.config.Plan.IncludedUsageUSD > 0 {
						// Calculate percentage based on dollar usage
						// If on-demand, we've used at least the included amount
						usageData.UsagePercentage = 100.0
						m.logger.Printf("Recalculated from events: %d included requests, %.1f%% usage", 
							stats.IncludedRequests, usageData.UsagePercentage)
					}
				}
			}
		}
	}

	// Cleanup old data
	if err := m.storage.CleanupOldData(m.config.Database.RetentionDays); err != nil {
		m.logger.Printf("Warning: Could not cleanup old data: %v", err)
	}

	m.logger.Printf("Poll completed: Usage %d/%d (%.1f%%)", 
		usageData.PremiumRequestsUsed, 
		usageData.PremiumRequestsLimit, 
		usageData.UsagePercentage)

	return nil
}
