package storage

import (
	"fmt"
)

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	TotalRequests      int
	IncludedRequests   int
	OnDemandRequests   int
	TotalTokens        int64
	IncludedTokens     int64
	OnDemandTokens     int64
	TotalCostUSD       float64
	IncludedCostUSD    float64 // Should be 0, but calculated for completeness
	OnDemandCostUSD    float64
	RequestsByModel    map[string]int
	TokensByModel      map[string]int64
	CostByModel        map[string]float64
	RequestsByType     map[string]int // "Included", "On-Demand", etc.
	TokensByType       map[string]int64
	CostByType         map[string]float64
}

// CalculateUsageStats calculates comprehensive usage statistics from events
func (s *Storage) CalculateUsageStats(billingCycle string) (*UsageStats, error) {
	events, err := s.GetUsageEventsForCycle(billingCycle)
	if err != nil {
		return nil, fmt.Errorf("getting usage events: %w", err)
	}

	stats := &UsageStats{
		RequestsByModel: make(map[string]int),
		TokensByModel:   make(map[string]int64),
		CostByModel:     make(map[string]float64),
		RequestsByType:  make(map[string]int),
		TokensByType:    make(map[string]int64),
		CostByType:      make(map[string]float64),
	}

	for _, event := range events {
		// Count requests
		stats.TotalRequests++
		stats.RequestsByModel[event.Model]++
		stats.RequestsByType[event.Kind]++

		if event.Kind == "Included" {
			stats.IncludedRequests++
		} else if event.Kind == "On-Demand" {
			stats.OnDemandRequests++
		}

		// Sum tokens
		totalTokens := int64(event.TotalTokens)
		stats.TotalTokens += totalTokens
		stats.TokensByModel[event.Model] += totalTokens
		stats.TokensByType[event.Kind] += totalTokens

		if event.Kind == "Included" {
			stats.IncludedTokens += totalTokens
		} else if event.Kind == "On-Demand" {
			stats.OnDemandTokens += totalTokens
		}

		// Sum costs
		stats.TotalCostUSD += event.Cost
		stats.CostByModel[event.Model] += event.Cost
		stats.CostByType[event.Kind] += event.Cost

		if event.Kind == "Included" {
			stats.IncludedCostUSD += event.Cost
		} else if event.Kind == "On-Demand" {
			stats.OnDemandCostUSD += event.Cost
		}
	}

	return stats, nil
}

// GetIncludedUsageUSD calculates the dollar value of included usage from events
// This is more accurate than request counts since different models have different costs
func (s *Storage) GetIncludedUsageUSD(billingCycle string) (float64, error) {
	stats, err := s.CalculateUsageStats(billingCycle)
	if err != nil {
		return 0, err
	}
	return stats.IncludedCostUSD, nil
}

// GetOnDemandUsageUSD gets the on-demand spending from events
func (s *Storage) GetOnDemandUsageUSD(billingCycle string) (float64, error) {
	stats, err := s.CalculateUsageStats(billingCycle)
	if err != nil {
		return 0, err
	}
	return stats.OnDemandCostUSD, nil
}

// GetTotalUsageUSD gets total usage (included + on-demand) in dollars
func (s *Storage) GetTotalUsageUSD(billingCycle string) (float64, error) {
	stats, err := s.CalculateUsageStats(billingCycle)
	if err != nil {
		return 0, err
	}
	return stats.TotalCostUSD, nil
}

// AggregateUsageStats represents aggregate usage statistics matching CSV export format
type AggregateUsageStats struct {
	TotalEvents            int
	TotalTokens            int64
	InputTokensWithCache   int64
	InputTokensNoCache     int64
	CacheReads             int64
	OutputTokens           int64
	TotalRecordedCost      float64
	AvgCostPerRequest      float64
	MaxSingleRequest       float64
}

// CalculateAggregateStats calculates aggregate usage statistics from events
// This matches the format shown in exported CSV aggregate tables
func (s *Storage) CalculateAggregateStats(billingCycle string) (*AggregateUsageStats, error) {
	events, err := s.GetUsageEventsForCycle(billingCycle)
	if err != nil {
		return nil, fmt.Errorf("getting usage events: %w", err)
	}

	stats := &AggregateUsageStats{
		TotalEvents: len(events),
	}

	var maxCost float64

	for _, event := range events {
		// Sum tokens
		stats.TotalTokens += int64(event.TotalTokens)
		stats.InputTokensWithCache += int64(event.InputWithCacheWrite)
		stats.InputTokensNoCache += int64(event.InputWithoutCacheWrite)
		stats.CacheReads += int64(event.CacheRead)
		stats.OutputTokens += int64(event.OutputTokens)

		// Sum costs
		stats.TotalRecordedCost += event.Cost

		// Track max single request cost
		if event.Cost > maxCost {
			maxCost = event.Cost
		}
	}

	stats.MaxSingleRequest = maxCost

	// Calculate average cost per request
	if stats.TotalEvents > 0 {
		stats.AvgCostPerRequest = stats.TotalRecordedCost / float64(stats.TotalEvents)
	}

	return stats, nil
}
