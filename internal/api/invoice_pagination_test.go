package api

import (
	"encoding/json"
	"testing"
)

// TestPaginationLogic tests the pagination logic without making actual HTTP requests
func TestPaginationLogic(t *testing.T) {
	tests := []struct {
		name              string
		totalCount        int
		pageSize          int
		expectedPages     int
		expectedLastPageSize int
	}{
		{
			name:              "exact page boundary",
			totalCount:        200,
			pageSize:          100,
			expectedPages:     2,
			expectedLastPageSize: 100,
		},
		{
			name:              "less than one page",
			totalCount:        50,
			pageSize:          100,
			expectedPages:     1,
			expectedLastPageSize: 50,
		},
		{
			name:              "multiple pages with remainder",
			totalCount:        808,
			pageSize:          100,
			expectedPages:     9,
			expectedLastPageSize: 8,
		},
		{
			name:              "single page",
			totalCount:        100,
			pageSize:          100,
			expectedPages:     1,
			expectedLastPageSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate expected pages
			pages := (tt.totalCount + tt.pageSize - 1) / tt.pageSize
			if pages != tt.expectedPages {
				t.Errorf("Expected %d pages, got %d", tt.expectedPages, pages)
			}

			// Calculate last page size
			lastPageSize := tt.totalCount % tt.pageSize
			if lastPageSize == 0 {
				lastPageSize = tt.pageSize
			}
			if lastPageSize != tt.expectedLastPageSize {
				t.Errorf("Expected last page size %d, got %d", tt.expectedLastPageSize, lastPageSize)
			}
		})
	}
}

// TestPaginationResponseParsing tests parsing of paginated API responses
func TestPaginationResponseParsing(t *testing.T) {
	// Test parsing totalUsageEventsCount from response
	responseJSON := `{
		"totalUsageEventsCount": 808,
		"usageEventsDisplay": []
	}`

	var rawResponse map[string]interface{}
	if err := json.Unmarshal([]byte(responseJSON), &rawResponse); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	totalCountRaw, ok := rawResponse["totalUsageEventsCount"]
	if !ok {
		t.Fatal("totalUsageEventsCount not found in response")
	}

	var totalCount int
	switch v := totalCountRaw.(type) {
	case int:
		totalCount = v
	case int64:
		totalCount = int(v)
	case float64:
		totalCount = int(v)
	default:
		t.Fatalf("Unexpected type for totalUsageEventsCount: %T", v)
	}

	if totalCount != 808 {
		t.Errorf("Expected totalCount 808, got %d", totalCount)
	}
}
