package storage

import (
	"database/sql"
	"fmt"
)

// RunMigrations creates all necessary database tables
func RunMigrations(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS usage_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			billing_cycle_start TEXT NOT NULL,
			premium_requests_used INTEGER,
			premium_requests_limit INTEGER,
			usage_percentage REAL,
			is_on_demand BOOLEAN DEFAULT FALSE,
			on_demand_spend_cents INTEGER DEFAULT 0,
			raw_response TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS alerts_sent (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			alert_type TEXT NOT NULL,
			threshold_value REAL,
			billing_cycle TEXT NOT NULL,
			UNIQUE(alert_type, threshold_value, billing_cycle)
		)`,
		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS invoice_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			billing_cycle TEXT NOT NULL,
			model_name TEXT,
			request_count INTEGER,
			cost_cents INTEGER,
			is_discounted BOOLEAN DEFAULT FALSE,
			fetched_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS usage_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_date TEXT NOT NULL,
			billing_cycle TEXT NOT NULL,
			kind TEXT NOT NULL,
			model TEXT NOT NULL,
			max_mode TEXT,
			input_with_cache_write INTEGER,
			input_without_cache_write INTEGER,
			cache_read INTEGER,
			output_tokens INTEGER,
			total_tokens INTEGER,
			cost REAL,
			fetched_at TEXT NOT NULL,
			UNIQUE(event_date, model, kind, total_tokens)
		)`,
		// Create indexes for better query performance
		`CREATE INDEX IF NOT EXISTS idx_usage_snapshots_timestamp ON usage_snapshots(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_snapshots_billing_cycle ON usage_snapshots(billing_cycle_start)`,
		`CREATE INDEX IF NOT EXISTS idx_invoice_items_billing_cycle ON invoice_items(billing_cycle)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_events_billing_cycle ON usage_events(billing_cycle)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_events_kind ON usage_events(kind)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_events_date ON usage_events(event_date)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("running migration %d: %w", i+1, err)
		}
	}

	return nil
}
