package config

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// SetupLogger creates a logger based on config settings
// Returns a logger and a closer function to close log file if needed
func SetupLogger(cfg *LoggingConfig) (*log.Logger, func() error, error) {
	var writers []io.Writer
	
	// Always log to stdout
	writers = append(writers, os.Stdout)
	
	var closer func() error = func() error { return nil }
	
	// Also log to file if configured
	if cfg.File != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(cfg.File)
		if logDir != "." && logDir != "" {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				// If we can't create directory, log warning but continue with stdout only
				log.New(os.Stderr, "WARN: ", log.LstdFlags).Printf("Could not create log directory %s: %v. Logging to stdout only.", logDir, err)
			} else {
				// Try to open log file
				file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					// If we can't open file, log warning but continue with stdout only
					log.New(os.Stderr, "WARN: ", log.LstdFlags).Printf("Could not open log file %s: %v. Logging to stdout only.", cfg.File, err)
				} else {
					writers = append(writers, file)
					closer = file.Close
				}
			}
		}
	}
	
	// Create multi-writer logger
	multiWriter := io.MultiWriter(writers...)
	logger := log.New(multiWriter, "", log.LstdFlags)
	
	return logger, closer, nil
}
