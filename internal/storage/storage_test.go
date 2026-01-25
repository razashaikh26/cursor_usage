package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*Storage, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	return store, dbPath
}

func TestStorage_SaveAndGetSnapshot(t *testing.T) {
	store, _ := setupTestDB(t)
	defer store.Close()

	snapshot := &UsageSnapshot{
		Timestamp:            time.Now(),
		BillingCycleStart:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PremiumRequestsUsed: 100,
		PremiumRequestsLimit: 500,
		UsagePercentage:     20.0,
		IsOnDemand:          false,
		OnDemandSpendCents:  0,
		RawResponse:         `{"test": "data"}`,
	}

	if err := store.SaveUsageSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	retrieved, err := store.GetLatestSnapshot()
	if err != nil {
		t.Fatalf("Failed to get latest snapshot: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected snapshot, got nil")
	}

	if retrieved.PremiumRequestsUsed != snapshot.PremiumRequestsUsed {
		t.Errorf("Expected %d requests, got %d", snapshot.PremiumRequestsUsed, retrieved.PremiumRequestsUsed)
	}

	if retrieved.UsagePercentage != snapshot.UsagePercentage {
		t.Errorf("Expected %.1f%%, got %.1f%%", snapshot.UsagePercentage, retrieved.UsagePercentage)
	}
}

func TestStorage_AlertTracking(t *testing.T) {
	store, _ := setupTestDB(t)
	defer store.Close()

	billingCycle := "2026-01-01"
	alertType := "threshold"
	threshold := 75.0

	// Check that alert hasn't been sent
	sent, err := store.AlertAlreadySent(alertType, threshold, billingCycle)
	if err != nil {
		t.Fatalf("Failed to check alert: %v", err)
	}
	if sent {
		t.Error("Expected alert not to be sent yet")
	}

	// Record alert
	if err := store.RecordAlert(alertType, threshold, billingCycle); err != nil {
		t.Fatalf("Failed to record alert: %v", err)
	}

	// Check that alert has been sent
	sent, err = store.AlertAlreadySent(alertType, threshold, billingCycle)
	if err != nil {
		t.Fatalf("Failed to check alert: %v", err)
	}
	if !sent {
		t.Error("Expected alert to be sent")
	}

	// Try to record again (should be idempotent)
	if err := store.RecordAlert(alertType, threshold, billingCycle); err != nil {
		t.Fatalf("Failed to record alert again: %v", err)
	}
}

func TestStorage_InvoiceItems(t *testing.T) {
	store, _ := setupTestDB(t)
	defer store.Close()

	billingCycle := "2026-01-01"
	items := []InvoiceItem{
		{
			BillingCycle: billingCycle,
			ModelName:    "claude-4-sonnet",
			RequestCount: 100,
			CostCents:    1000,
			IsDiscounted: false,
			FetchedAt:    time.Now(),
		},
		{
			BillingCycle: billingCycle,
			ModelName:    "gpt-4o",
			RequestCount: 50,
			CostCents:    500,
			IsDiscounted: false,
			FetchedAt:    time.Now(),
		},
	}

	if err := store.SaveInvoiceItems(billingCycle, items); err != nil {
		t.Fatalf("Failed to save invoice items: %v", err)
	}

	retrieved, err := store.GetInvoiceItemsForCycle(billingCycle)
	if err != nil {
		t.Fatalf("Failed to get invoice items: %v", err)
	}

	if len(retrieved) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(retrieved))
	}

	if retrieved[0].ModelName != items[0].ModelName {
		t.Errorf("Expected model %s, got %s", items[0].ModelName, retrieved[0].ModelName)
	}
}

func TestStorage_GetPreviousSnapshot(t *testing.T) {
	store, _ := setupTestDB(t)
	defer store.Close()

	// Save two snapshots
	snapshot1 := &UsageSnapshot{
		Timestamp:            time.Now().Add(-2 * time.Hour),
		BillingCycleStart:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PremiumRequestsUsed: 100,
		PremiumRequestsLimit: 500,
		UsagePercentage:     20.0,
	}

	snapshot2 := &UsageSnapshot{
		Timestamp:            time.Now(),
		BillingCycleStart:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PremiumRequestsUsed: 200,
		PremiumRequestsLimit: 500,
		UsagePercentage:     40.0,
	}

	store.SaveUsageSnapshot(snapshot1)
	store.SaveUsageSnapshot(snapshot2)

	previous, err := store.GetPreviousSnapshot()
	if err != nil {
		t.Fatalf("Failed to get previous snapshot: %v", err)
	}

	if previous == nil {
		t.Fatal("Expected previous snapshot, got nil")
	}

	if previous.PremiumRequestsUsed != snapshot1.PremiumRequestsUsed {
		t.Errorf("Expected %d requests in previous snapshot, got %d", 
			snapshot1.PremiumRequestsUsed, previous.PremiumRequestsUsed)
	}
}

func TestStorage_CleanupOldData(t *testing.T) {
	store, _ := setupTestDB(t)
	defer store.Close()

	// Save old snapshot
	oldSnapshot := &UsageSnapshot{
		Timestamp:            time.Now().AddDate(0, 0, -100), // 100 days ago
		BillingCycleStart:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PremiumRequestsUsed: 100,
		PremiumRequestsLimit: 500,
		UsagePercentage:     20.0,
	}

	store.SaveUsageSnapshot(oldSnapshot)

	// Cleanup data older than 90 days
	if err := store.CleanupOldData(90); err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// Old snapshot should be gone
	snapshot, err := store.GetLatestSnapshot()
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}

	if snapshot != nil {
		t.Error("Expected old snapshot to be deleted")
	}
}
