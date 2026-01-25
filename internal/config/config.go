package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Polling  PollingConfig  `yaml:"polling"`
	Alerts   AlertsConfig   `yaml:"alerts"`
	Database DatabaseConfig `yaml:"database"`
	BYOK     BYOKConfig     `yaml:"byok"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// PollingConfig holds polling interval settings
type PollingConfig struct {
	IntervalMinutes int `yaml:"interval_minutes"`
}

// AlertsConfig holds alert threshold and notification settings
type AlertsConfig struct {
	Thresholds        []float64 `yaml:"thresholds"`
	OnDemandCritical  bool      `yaml:"on_demand_critical"`
	Sound             string    `yaml:"sound"`
}

// DatabaseConfig holds database path and retention settings
type DatabaseConfig struct {
	Path          string `yaml:"path"`
	RetentionDays int    `yaml:"retention_days"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	File      string `yaml:"file"`      // Log file path (empty = stdout only)
	Level     string `yaml:"level"`     // Log level: debug, info, warn, error
	MaxSizeMB int    `yaml:"max_size_mb"` // Max log file size in MB before rotation
}

// BYOKConfig holds BYOK cost comparison settings
type BYOKConfig struct {
	Enabled       bool `yaml:"enabled"`
	ShowComparison bool `yaml:"show_comparison"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Polling: PollingConfig{
			IntervalMinutes: 15,
		},
		Alerts: AlertsConfig{
			Thresholds:       []float64{75, 90, 100},
			OnDemandCritical: true,
			Sound:            "default",
		},
		Database: DatabaseConfig{
			Path:          "~/.cursor_monitor/metrics.db",
			RetentionDays: 90,
		},
		BYOK: BYOKConfig{
			Enabled:        false,
			ShowComparison: true,
		},
		Logging: LoggingConfig{
			File:      "~/.cursor_monitor/cursor-monitor.log",
			Level:     "info",
			MaxSizeMB: 10,
		},
	}
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	// Expand ~ in path
	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("expanding path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Expand ~ in database path
	config.Database.Path, err = expandPath(config.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("expanding database path: %w", err)
	}

	// Expand ~ in log file path
	if config.Logging.File != "" {
		config.Logging.File, err = expandPath(config.Logging.File)
		if err != nil {
			return nil, fmt.Errorf("expanding log file path: %w", err)
		}
	}

	return config, nil
}

// PollInterval returns the polling interval as a time.Duration
func (c *Config) PollInterval() time.Duration {
	return time.Duration(c.Polling.IntervalMinutes) * time.Minute
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("getting current user: %w", err)
	}

	if path == "~" {
		return usr.HomeDir, nil
	}

	return filepath.Join(usr.HomeDir, path[2:]), nil
}
