package importcsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cursor-monitor/internal/storage"
)

// ImportUsageEventsCSV imports usage events from a CSV file
func ImportUsageEventsCSV(csvPath string, store *storage.Storage) error {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file has no data rows (only header)")
	}

	// Parse header to find column indices
	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.TrimSpace(col)] = i
	}

	// Required columns
	dateCol, ok := colMap["Date"]
	if !ok {
		return fmt.Errorf("CSV missing 'Date' column")
	}
	kindCol, ok := colMap["Kind"]
	if !ok {
		return fmt.Errorf("CSV missing 'Kind' column")
	}
	modelCol, ok := colMap["Model"]
	if !ok {
		return fmt.Errorf("CSV missing 'Model' column")
	}

	// Optional columns (with defaults)
	maxModeCol := colMap["Max Mode"]
	inputWithCacheCol := colMap["Input (w/ Cache Write)"]
	inputWithoutCacheCol := colMap["Input (w/o Cache Write)"]
	cacheReadCol := colMap["Cache Read"]
	outputTokensCol := colMap["Output Tokens"]
	totalTokensCol := colMap["Total Tokens"]
	costCol := colMap["Cost"]

	var events []storage.UsageEvent
	var billingCycles = make(map[string]bool)

	for i, record := range records[1:] {
		if len(record) <= dateCol {
			continue // Skip malformed rows
		}

		// Parse date
		dateStr := strings.Trim(record[dateCol], `"`)
		eventDate, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			// Try alternative format
			eventDate, err = time.Parse("2006-01-02T15:04:05.000Z", dateStr)
			if err != nil {
				fmt.Printf("Warning: Skipping row %d - invalid date: %s\n", i+2, dateStr)
				continue
			}
		}

		// Determine billing cycle (month of the event)
		billingCycle := eventDate.Format("2006-01-02")
		billingCycles[billingCycle] = true

		// Parse values with defaults
		kind := strings.Trim(record[kindCol], `"`)
		model := strings.Trim(record[modelCol], `"`)
		maxMode := ""
		if maxModeCol < len(record) {
			maxMode = strings.Trim(record[maxModeCol], `"`)
		}

		// Parse numeric fields
		inputWithCache := parseInt(record, inputWithCacheCol)
		inputWithoutCache := parseInt(record, inputWithoutCacheCol)
		cacheRead := parseInt(record, cacheReadCol)
		outputTokens := parseInt(record, outputTokensCol)
		totalTokens := parseInt(record, totalTokensCol)
		cost := parseFloat64(record, costCol)

		events = append(events, storage.UsageEvent{
			EventDate:             eventDate,
			BillingCycle:          billingCycle,
			Kind:                  kind,
			Model:                 model,
			MaxMode:               maxMode,
			InputWithCacheWrite:   inputWithCache,
			InputWithoutCacheWrite: inputWithoutCache,
			CacheRead:             cacheRead,
			OutputTokens:          outputTokens,
			TotalTokens:           totalTokens,
			Cost:                  cost,
			FetchedAt:             time.Now(),
		})
	}

	// Group events by billing cycle and save
	for cycle := range billingCycles {
		var cycleEvents []storage.UsageEvent
		for _, event := range events {
			if event.BillingCycle == cycle {
				cycleEvents = append(cycleEvents, event)
			}
		}
		if len(cycleEvents) > 0 {
			if err := store.SaveUsageEvents(cycle, cycleEvents); err != nil {
				return fmt.Errorf("saving events for cycle %s: %w", cycle, err)
			}
			fmt.Printf("Imported %d usage events for billing cycle %s\n", len(cycleEvents), cycle)
		}
	}

	fmt.Printf("Successfully imported %d total usage events from CSV\n", len(events))
	return nil
}

func parseInt(record []string, col int) int {
	if col >= len(record) {
		return 0
	}
	val := strings.Trim(record[col], `"`)
	if val == "" {
		return 0
	}
	i, _ := strconv.Atoi(val)
	return i
}

func parseFloat64(record []string, col int) float64 {
	if col >= len(record) {
		return 0
	}
	val := strings.Trim(record[col], `"`)
	if val == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(val, 64)
	return f
}
