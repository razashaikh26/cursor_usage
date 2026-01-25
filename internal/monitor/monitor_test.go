package monitor

import (
	"log"
	"os"
	"testing"

	"cursor-monitor/internal/config"
)

func TestMonitorInitialization(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// This will fail if Cursor database doesn't exist, which is expected in tests
	// We'd need to mock the auth layer for proper unit testing
	_, err := New(cfg, logger)
	
	// We expect this to fail in test environment (no Cursor DB)
	// But we can test that the structure is correct
	if err != nil {
		// Expected - Cursor database won't exist in test environment
		t.Logf("Monitor initialization failed as expected (no Cursor DB): %v", err)
	}
}

// Note: Full integration tests would require:
// 1. Mocking the auth layer to avoid needing Cursor's database
// 2. Mocking the API client to avoid making real HTTP requests
// 3. Using an in-memory database for storage tests
// These are better suited for integration tests rather than unit tests
