package ygggo_mysql

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// SlowQueryAnalyzer provides advanced analysis of slow queries
type SlowQueryAnalyzer struct {
	storage SlowQueryStorage
}

// NewSlowQueryAnalyzer creates a new slow query analyzer
func NewSlowQueryAnalyzer(storage SlowQueryStorage) *SlowQueryAnalyzer {
	return &SlowQueryAnalyzer{
		storage: storage,
	}
}

// AnalysisReport contains comprehensive analysis results
type AnalysisReport struct {
	Summary          AnalysisSummary    `json:"summary"`
	TopSlowQueries   []*SlowQueryRecord `json:"top_slow_queries"`
	FrequentPatterns []*QueryPattern    `json:"frequent_patterns"`
	TimeDistribution []TimeSlot         `json:"time_distribution"`
	Recommendations  []string           `json:"recommendations"`
	GeneratedAt      time.Time          `json:"generated_at"`
}

// AnalysisSummary provides high-level statistics
type AnalysisSummary struct {
	TotalQueries     int64         `json:"total_queries"`
	UniquePatterns   int64         `json:"unique_patterns"`
	AverageDuration  time.Duration `json:"average_duration"`
	MedianDuration   time.Duration `json:"median_duration"`
	P95Duration      time.Duration `json:"p95_duration"`
	P99Duration      time.Duration `json:"p99_duration"`
	SlowestQuery     time.Duration `json:"slowest_query"`
	TimeRange        TimeRange     `json:"time_range"`
}

// TimeRange represents a time period
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TimeSlot represents query distribution in a time period
type TimeSlot struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	QueryCount   int64         `json:"query_count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
}

// GenerateReport generates a comprehensive analysis report
func (a *SlowQueryAnalyzer) GenerateReport(ctx context.Context, filter SlowQueryFilter) (*AnalysisReport, error) {
	// Get all records matching the filter
	records, err := a.storage.GetRecords(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}
	
	if len(records) == 0 {
		return &AnalysisReport{
			GeneratedAt: time.Now(),
		}, nil
	}
	
	// Generate summary
	summary := a.generateSummary(records)
	
	// Get top slow queries
	topSlow := a.getTopSlowQueries(records, 10)
	
	// Get frequent patterns
	patterns, err := a.storage.GetPatterns(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns: %w", err)
	}
	
	// Generate time distribution
	timeDistribution := a.generateTimeDistribution(records, 24) // 24 hour slots
	
	// Generate recommendations
	recommendations := a.generateRecommendations(records, patterns)
	
	return &AnalysisReport{
		Summary:          summary,
		TopSlowQueries:   topSlow,
		FrequentPatterns: patterns,
		TimeDistribution: timeDistribution,
		Recommendations:  recommendations,
		GeneratedAt:      time.Now(),
	}, nil
}

// generateSummary creates summary statistics
func (a *SlowQueryAnalyzer) generateSummary(records []*SlowQueryRecord) AnalysisSummary {
	if len(records) == 0 {
		return AnalysisSummary{}
	}
	
	// Sort by duration for percentile calculations
	durations := make([]time.Duration, len(records))
	for i, record := range records {
		durations[i] = record.Duration
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	
	// Calculate statistics
	var totalDuration time.Duration
	var minTime, maxTime time.Time
	patterns := make(map[string]bool)
	
	for i, record := range records {
		totalDuration += record.Duration
		patterns[record.NormalizedQuery] = true
		
		if i == 0 || record.Timestamp.Before(minTime) {
			minTime = record.Timestamp
		}
		if i == 0 || record.Timestamp.After(maxTime) {
			maxTime = record.Timestamp
		}
	}
	
	avgDuration := time.Duration(int64(totalDuration) / int64(len(records)))
	medianDuration := durations[len(durations)/2]
	p95Duration := durations[int(float64(len(durations))*0.95)]
	p99Duration := durations[int(float64(len(durations))*0.99)]
	
	return AnalysisSummary{
		TotalQueries:    int64(len(records)),
		UniquePatterns:  int64(len(patterns)),
		AverageDuration: avgDuration,
		MedianDuration:  medianDuration,
		P95Duration:     p95Duration,
		P99Duration:     p99Duration,
		SlowestQuery:    durations[len(durations)-1],
		TimeRange: TimeRange{
			Start: minTime,
			End:   maxTime,
		},
	}
}

// getTopSlowQueries returns the slowest queries
func (a *SlowQueryAnalyzer) getTopSlowQueries(records []*SlowQueryRecord, limit int) []*SlowQueryRecord {
	// Sort by duration (slowest first)
	sorted := make([]*SlowQueryRecord, len(records))
	copy(sorted, records)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Duration > sorted[j].Duration
	})
	
	if limit > 0 && limit < len(sorted) {
		sorted = sorted[:limit]
	}
	
	return sorted
}

// generateTimeDistribution creates time-based distribution
func (a *SlowQueryAnalyzer) generateTimeDistribution(records []*SlowQueryRecord, slots int) []TimeSlot {
	if len(records) == 0 {
		return []TimeSlot{}
	}
	
	// Find time range
	var minTime, maxTime time.Time
	for i, record := range records {
		if i == 0 || record.Timestamp.Before(minTime) {
			minTime = record.Timestamp
		}
		if i == 0 || record.Timestamp.After(maxTime) {
			maxTime = record.Timestamp
		}
	}
	
	// Calculate slot duration
	totalDuration := maxTime.Sub(minTime)
	if totalDuration == 0 {
		totalDuration = time.Hour // Default to 1 hour if all queries at same time
	}
	slotDuration := totalDuration / time.Duration(slots)
	
	// Initialize slots
	timeSlots := make([]TimeSlot, slots)
	for i := 0; i < slots; i++ {
		startTime := minTime.Add(time.Duration(i) * slotDuration)
		endTime := startTime.Add(slotDuration)
		timeSlots[i] = TimeSlot{
			StartTime: startTime,
			EndTime:   endTime,
		}
	}
	
	// Distribute records into slots
	for _, record := range records {
		slotIndex := int(record.Timestamp.Sub(minTime) / slotDuration)
		if slotIndex >= slots {
			slotIndex = slots - 1
		}
		
		timeSlots[slotIndex].QueryCount++
		timeSlots[slotIndex].TotalTime += record.Duration
	}
	
	// Calculate averages
	for i := range timeSlots {
		if timeSlots[i].QueryCount > 0 {
			timeSlots[i].AverageTime = time.Duration(int64(timeSlots[i].TotalTime) / timeSlots[i].QueryCount)
		}
	}
	
	return timeSlots
}

// generateRecommendations creates optimization recommendations
func (a *SlowQueryAnalyzer) generateRecommendations(records []*SlowQueryRecord, patterns []*QueryPattern) []string {
	var recommendations []string
	
	if len(records) == 0 {
		return recommendations
	}
	
	// Analyze patterns for common issues
	for _, pattern := range patterns {
		if pattern.Count > 10 && pattern.AverageDuration > 500*time.Millisecond {
			recommendations = append(recommendations, 
				fmt.Sprintf("Consider optimizing frequent slow query pattern: %s (executed %d times, avg: %v)", 
					pattern.NormalizedQuery, pattern.Count, pattern.AverageDuration))
		}
		
		if containsKeyword(pattern.NormalizedQuery, "SELECT *") {
			recommendations = append(recommendations, 
				"Avoid SELECT * queries. Specify only needed columns to reduce data transfer.")
		}
		
		if containsKeyword(pattern.NormalizedQuery, "ORDER BY") && !containsKeyword(pattern.NormalizedQuery, "LIMIT") {
			recommendations = append(recommendations, 
				"ORDER BY without LIMIT can be expensive. Consider adding LIMIT if appropriate.")
		}
		
		if containsKeyword(pattern.NormalizedQuery, "LIKE") {
			recommendations = append(recommendations,
				"LIKE patterns can be expensive. Consider using indexes or full-text search for better performance.")
		}
	}
	
	// General recommendations based on statistics
	if len(records) > 100 {
		recommendations = append(recommendations, 
			"High number of slow queries detected. Consider reviewing query optimization and indexing strategy.")
	}
	
	// Calculate average duration
	var totalDuration time.Duration
	for _, record := range records {
		totalDuration += record.Duration
	}
	avgDuration := time.Duration(int64(totalDuration) / int64(len(records)))
	
	if avgDuration > time.Second {
		recommendations = append(recommendations, 
			"Average query duration is high. Consider database performance tuning.")
	}
	
	return recommendations
}

// Helper function to check if query contains a keyword
func containsKeyword(query, keyword string) bool {
	return len(query) >= len(keyword) && findSubstring(query, keyword)
}
