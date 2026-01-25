package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Polling.IntervalMinutes != 15 {
		t.Errorf("Expected default interval 15, got %d", cfg.Polling.IntervalMinutes)
	}

	if len(cfg.Alerts.Thresholds) != 3 {
		t.Errorf("Expected 3 default thresholds, got %d", len(cfg.Alerts.Thresholds))
	}

	if cfg.Database.RetentionDays != 90 {
		t.Errorf("Expected default retention 90 days, got %d", cfg.Database.RetentionDays)
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantHome bool
	}{
		{
			name:     "tilde expansion",
			path:     "~/.test",
			wantHome: true,
		},
		{
			name:     "no tilde",
			path:     "/absolute/path",
			wantHome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandPath(tt.path)
			if err != nil {
				t.Errorf("expandPath() error = %v", err)
				return
			}

			if tt.wantHome {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, ".test")
				if got != expected {
					t.Errorf("expandPath() = %v, want %v", got, expected)
				}
			} else {
				if got != tt.path {
					t.Errorf("expandPath() = %v, want %v", got, tt.path)
				}
			}
		})
	}
}
