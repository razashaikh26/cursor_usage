package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Storage handles all database operations
type Storage struct {
	db *sql.DB
}

// UsageSnapshot represents a single usage data point
type UsageSnapshot struct {
	ID                  int64
	Timestamp            time.Time
	BillingCycleStart   time.Time
	PremiumRequestsUsed int
	PremiumRequestsLimit int
	UsagePercentage     float64
	IsOnDemand          bool
	OnDemandSpendCents  int
	RawResponse         string
}

// InvoiceItem represents a billing item from Cursor
type InvoiceItem struct {
	ID            int64
	BillingCycle  string
	ModelName     string
	RequestCount  int
	CostCents     int
	IsDiscounted  bool
	FetchedAt     time.Time
}

// UsageEvent represents a single usage event
type UsageEvent struct {
	ID                    int64
	EventDate             time.Time
	BillingCycle          string
	Kind                  string // "Included", "On-Demand", "Errored, No Charge", etc.
	Model                 string
	MaxMode               string
	InputWithCacheWrite   int
	InputWithoutCacheWrite int
	CacheRead             int
	OutputTokens          int
	TotalTokens           int
	Cost                  float64
	FetchedAt             time.Time
}

// New creates a new Storage instance and opens the database
func New(dbPath string) (*Storage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable foreign keys and set connection pool settings
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	storage := &Storage{db: db}

	// Run migrations
	if err := RunMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveUsageSnapshot saves a usage snapshot to the database
func (s *Storage) SaveUsageSnapshot(snapshot *UsageSnapshot) error {
	query := `INSERT INTO usage_snapshots 
		(timestamp, billing_cycle_start, premium_requests_used, premium_requests_limit, 
		 usage_percentage, is_on_demand, on_demand_spend_cents, raw_response)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(
		query,
		snapshot.Timestamp.Format(time.RFC3339),
		snapshot.BillingCycleStart.Format(time.RFC3339),
		snapshot.PremiumRequestsUsed,
		snapshot.PremiumRequestsLimit,
		snapshot.UsagePercentage,
		snapshot.IsOnDemand,
		snapshot.OnDemandSpendCents,
		snapshot.RawResponse,
	)

	if err != nil {
		return fmt.Errorf("saving usage snapshot: %w", err)
	}

	return nil
}

// GetLatestSnapshot returns the most recent usage snapshot
func (s *Storage) GetLatestSnapshot() (*UsageSnapshot, error) {
	query := `SELECT id, timestamp, billing_cycle_start, premium_requests_used, 
		premium_requests_limit, usage_percentage, is_on_demand, on_demand_spend_cents, raw_response
		FROM usage_snapshots 
		ORDER BY timestamp DESC 
		LIMIT 1`

	var snapshot UsageSnapshot
	var timestampStr, cycleStr string

	err := s.db.QueryRow(query).Scan(
		&snapshot.ID,
		&timestampStr,
		&cycleStr,
		&snapshot.PremiumRequestsUsed,
		&snapshot.PremiumRequestsLimit,
		&snapshot.UsagePercentage,
		&snapshot.IsOnDemand,
		&snapshot.OnDemandSpendCents,
		&snapshot.RawResponse,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting latest snapshot: %w", err)
	}

	snapshot.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
	snapshot.BillingCycleStart, _ = time.Parse(time.RFC3339, cycleStr)

	return &snapshot, nil
}

// GetPreviousSnapshot returns the second most recent snapshot (for comparison)
func (s *Storage) GetPreviousSnapshot() (*UsageSnapshot, error) {
	query := `SELECT id, timestamp, billing_cycle_start, premium_requests_used, 
		premium_requests_limit, usage_percentage, is_on_demand, on_demand_spend_cents, raw_response
		FROM usage_snapshots 
		ORDER BY timestamp DESC 
		LIMIT 1 OFFSET 1`

	var snapshot UsageSnapshot
	var timestampStr, cycleStr string

	err := s.db.QueryRow(query).Scan(
		&snapshot.ID,
		&timestampStr,
		&cycleStr,
		&snapshot.PremiumRequestsUsed,
		&snapshot.PremiumRequestsLimit,
		&snapshot.UsagePercentage,
		&snapshot.IsOnDemand,
		&snapshot.OnDemandSpendCents,
		&snapshot.RawResponse,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting previous snapshot: %w", err)
	}

	snapshot.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
	snapshot.BillingCycleStart, _ = time.Parse(time.RFC3339, cycleStr)

	return &snapshot, nil
}

// AlertAlreadySent checks if an alert has already been sent for a given threshold and billing cycle
func (s *Storage) AlertAlreadySent(alertType string, thresholdValue float64, billingCycle string) (bool, error) {
	query := `SELECT COUNT(*) FROM alerts_sent 
		WHERE alert_type = ? AND threshold_value = ? AND billing_cycle = ?`

	var count int
	err := s.db.QueryRow(query, alertType, thresholdValue, billingCycle).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking alert history: %w", err)
	}

	return count > 0, nil
}

// RecordAlert records that an alert was sent
func (s *Storage) RecordAlert(alertType string, thresholdValue float64, billingCycle string) error {
	query := `INSERT OR IGNORE INTO alerts_sent (timestamp, alert_type, threshold_value, billing_cycle)
		VALUES (?, ?, ?, ?)`

	_, err := s.db.Exec(
		query,
		time.Now().Format(time.RFC3339),
		alertType,
		thresholdValue,
		billingCycle,
	)

	if err != nil {
		return fmt.Errorf("recording alert: %w", err)
	}

	return nil
}

// SaveInvoiceItems saves invoice items for a billing cycle
func (s *Storage) SaveInvoiceItems(billingCycle string, items []InvoiceItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	
	// Use a flag to track if we should rollback
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// Delete existing items for this billing cycle
	_, err = tx.Exec("DELETE FROM invoice_items WHERE billing_cycle = ?", billingCycle)
	if err != nil {
		return fmt.Errorf("deleting existing invoice items: %w", err)
	}

	// Insert new items
	stmt, err := tx.Prepare(`INSERT INTO invoice_items 
		(billing_cycle, model_name, request_count, cost_cents, is_discounted, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing insert statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		_, err = stmt.Exec(
			item.BillingCycle,
			item.ModelName,
			item.RequestCount,
			item.CostCents,
			item.IsDiscounted,
			item.FetchedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("inserting invoice item: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	committed = true

	return nil
}

// GetInvoiceItemsForCycle returns all invoice items for a billing cycle
func (s *Storage) GetInvoiceItemsForCycle(billingCycle string) ([]InvoiceItem, error) {
	query := `SELECT id, billing_cycle, model_name, request_count, cost_cents, is_discounted, fetched_at
		FROM invoice_items 
		WHERE billing_cycle = ?
		ORDER BY fetched_at DESC`

	rows, err := s.db.Query(query, billingCycle)
	if err != nil {
		return nil, fmt.Errorf("querying invoice items: %w", err)
	}
	defer rows.Close()

	var items []InvoiceItem
	for rows.Next() {
		var item InvoiceItem
		var fetchedAtStr string

		err := rows.Scan(
			&item.ID,
			&item.BillingCycle,
			&item.ModelName,
			&item.RequestCount,
			&item.CostCents,
			&item.IsDiscounted,
			&fetchedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning invoice item: %w", err)
		}

		item.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAtStr)
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating invoice items: %w", err)
	}

	return items, nil
}

// SaveUsageEvents saves usage events for a billing cycle
func (s *Storage) SaveUsageEvents(billingCycle string, events []UsageEvent) error {
	if len(events) == 0 {
		return nil // Nothing to save
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	
	// Use a flag to track if we should rollback
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// Insert events (using INSERT OR IGNORE to handle duplicates)
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO usage_events 
		(event_date, billing_cycle, kind, model, max_mode, 
		 input_with_cache_write, input_without_cache_write, cache_read,
		 output_tokens, total_tokens, cost, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing insert statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		_, err = stmt.Exec(
			event.EventDate.Format(time.RFC3339),
			billingCycle,
			event.Kind,
			event.Model,
			event.MaxMode,
			event.InputWithCacheWrite,
			event.InputWithoutCacheWrite,
			event.CacheRead,
			event.OutputTokens,
			event.TotalTokens,
			event.Cost,
			event.FetchedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("inserting usage event: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	committed = true

	return nil
}

// GetUsageEventsForCycle returns all usage events for a billing cycle
func (s *Storage) GetUsageEventsForCycle(billingCycle string) ([]UsageEvent, error) {
	query := `SELECT id, event_date, billing_cycle, kind, model, max_mode,
		input_with_cache_write, input_without_cache_write, cache_read,
		output_tokens, total_tokens, cost, fetched_at
		FROM usage_events 
		WHERE billing_cycle = ?
		ORDER BY event_date DESC`
	
	return s.queryUsageEvents(query, billingCycle)
}

// GetUsageEventsForDateRange returns all usage events within a date range
func (s *Storage) GetUsageEventsForDateRange(startDate, endDate time.Time) ([]UsageEvent, error) {
	query := `SELECT id, event_date, billing_cycle, kind, model, max_mode,
		input_with_cache_write, input_without_cache_write, cache_read,
		output_tokens, total_tokens, cost, fetched_at
		FROM usage_events 
		WHERE event_date >= ? AND event_date < ?
		ORDER BY event_date DESC`
	
	startStr := startDate.Format(time.RFC3339)
	endStr := endDate.Format(time.RFC3339)
	
	return s.queryUsageEvents(query, startStr, endStr)
}

// queryUsageEvents is a helper to query usage events with variable parameters
func (s *Storage) queryUsageEvents(query string, args ...interface{}) ([]UsageEvent, error) {

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying usage events: %w", err)
	}
	defer rows.Close()

	var events []UsageEvent
	for rows.Next() {
		var event UsageEvent
		var eventDateStr, fetchedAtStr string

		err := rows.Scan(
			&event.ID,
			&eventDateStr,
			&event.BillingCycle,
			&event.Kind,
			&event.Model,
			&event.MaxMode,
			&event.InputWithCacheWrite,
			&event.InputWithoutCacheWrite,
			&event.CacheRead,
			&event.OutputTokens,
			&event.TotalTokens,
			&event.Cost,
			&fetchedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning usage event: %w", err)
		}

		event.EventDate, _ = time.Parse(time.RFC3339, eventDateStr)
		event.FetchedAt, _ = time.Parse(time.RFC3339, fetchedAtStr)
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating usage events: %w", err)
	}

	return events, nil
}

// CleanupOldData removes data older than the retention period
func (s *Storage) CleanupOldData(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	cutoffStr := cutoff.Format(time.RFC3339)

	// Delete old snapshots
	_, err := s.db.Exec("DELETE FROM usage_snapshots WHERE timestamp < ?", cutoffStr)
	if err != nil {
		return fmt.Errorf("deleting old snapshots: %w", err)
	}

	// Delete old alerts
	_, err = s.db.Exec("DELETE FROM alerts_sent WHERE timestamp < ?", cutoffStr)
	if err != nil {
		return fmt.Errorf("deleting old alerts: %w", err)
	}

	// Delete old invoice items
	_, err = s.db.Exec("DELETE FROM invoice_items WHERE fetched_at < ?", cutoffStr)
	if err != nil {
		return fmt.Errorf("deleting old invoice items: %w", err)
	}

	// Delete old usage events
	_, err = s.db.Exec("DELETE FROM usage_events WHERE fetched_at < ?", cutoffStr)
	if err != nil {
		return fmt.Errorf("deleting old usage events: %w", err)
	}

	return nil
}

// GetUsageSummary returns a summary of usage for the current billing cycle
func (s *Storage) GetUsageSummary(billingCycleStart time.Time) (map[string]interface{}, error) {
	cycleStr := billingCycleStart.Format(time.RFC3339)
	billingCycleDateStr := billingCycleStart.Format("2006-01-02")

	// Get max usage from snapshots
	query := `SELECT 
		MAX(premium_requests_used) as max_used,
		MAX(premium_requests_limit) as max_limit,
		MAX(usage_percentage) as max_percentage
		FROM usage_snapshots 
		WHERE billing_cycle_start = ?`

	var maxUsed, maxLimit sql.NullInt64
	var maxPercentage sql.NullFloat64

	err := s.db.QueryRow(query, cycleStr).Scan(&maxUsed, &maxLimit, &maxPercentage)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("getting usage summary: %w", err)
	}

	// Calculate running total from invoice items (more accurate than snapshots)
	// This gives us the true cumulative on-demand spend for the billing cycle
	invoiceItems, err := s.GetInvoiceItemsForCycle(billingCycleDateStr)
	if err != nil {
		// If we can't get invoice items, fall back to snapshot data
		invoiceItems = []InvoiceItem{}
	}

	totalOnDemandCents := 0
	for _, item := range invoiceItems {
		totalOnDemandCents += item.CostCents
	}

	summary := map[string]interface{}{
		"max_used":           0,
		"limit":              0,
		"max_percentage":     0.0,
		"total_on_demand_usd": 0.0,
	}

	if maxUsed.Valid {
		summary["max_used"] = int(maxUsed.Int64)
	}
	if maxLimit.Valid {
		summary["limit"] = int(maxLimit.Int64)
	}
	if maxPercentage.Valid {
		summary["max_percentage"] = maxPercentage.Float64
	}
	
	// Use invoice items total (running total) instead of snapshot sum
	summary["total_on_demand_usd"] = float64(totalOnDemandCents) / 100.0

	return summary, nil
}
