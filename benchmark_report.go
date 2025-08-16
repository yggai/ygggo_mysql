package ygggo_mysql

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// BenchmarkReport contains comprehensive benchmark results
type BenchmarkReport struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Summary     ReportSummary      `json:"summary"`
	Results     []*BenchmarkResult `json:"results"`
	Comparisons []Comparison       `json:"comparisons,omitempty"`
}

// ReportSummary provides high-level summary of all benchmark results
type ReportSummary struct {
	TotalTests       int           `json:"total_tests"`
	TotalDuration    time.Duration `json:"total_duration"`
	TotalOperations  int64         `json:"total_operations"`
	OverallThroughput float64      `json:"overall_throughput"`
	AverageLatency   time.Duration `json:"average_latency"`
	BestPerformer    string        `json:"best_performer"`
	WorstPerformer   string        `json:"worst_performer"`
}

// Comparison represents a comparison between two benchmark results
type Comparison struct {
	TestA      string  `json:"test_a"`
	TestB      string  `json:"test_b"`
	Metric     string  `json:"metric"`
	Difference float64 `json:"difference"` // Percentage difference
	Better     string  `json:"better"`     // Which test performed better
}

// BenchmarkReportGenerator generates comprehensive benchmark reports
type BenchmarkReportGenerator struct {
	results []*BenchmarkResult
}

// NewBenchmarkReportGenerator creates a new report generator
func NewBenchmarkReportGenerator() *BenchmarkReportGenerator {
	return &BenchmarkReportGenerator{
		results: make([]*BenchmarkResult, 0),
	}
}

// AddResult adds a benchmark result to the report
func (g *BenchmarkReportGenerator) AddResult(result *BenchmarkResult) {
	g.results = append(g.results, result)
}

// AddResults adds multiple benchmark results to the report
func (g *BenchmarkReportGenerator) AddResults(results []*BenchmarkResult) {
	g.results = append(g.results, results...)
}

// GenerateReport generates a comprehensive benchmark report
func (g *BenchmarkReportGenerator) GenerateReport() *BenchmarkReport {
	summary := g.generateSummary()
	comparisons := g.generateComparisons()
	
	return &BenchmarkReport{
		GeneratedAt: time.Now(),
		Summary:     summary,
		Results:     g.results,
		Comparisons: comparisons,
	}
}

// generateSummary creates a summary of all benchmark results
func (g *BenchmarkReportGenerator) generateSummary() ReportSummary {
	if len(g.results) == 0 {
		return ReportSummary{}
	}
	
	var totalDuration time.Duration
	var totalOperations int64
	var totalLatency time.Duration
	var bestThroughput float64
	var worstThroughput float64
	var bestPerformer, worstPerformer string
	
	for i, result := range g.results {
		totalDuration += result.Duration
		totalOperations += result.TotalOps
		totalLatency += result.AvgLatency
		
		if i == 0 || result.ThroughputOPS > bestThroughput {
			bestThroughput = result.ThroughputOPS
			bestPerformer = result.TestName
		}
		
		if i == 0 || result.ThroughputOPS < worstThroughput {
			worstThroughput = result.ThroughputOPS
			worstPerformer = result.TestName
		}
	}
	
	avgLatency := time.Duration(int64(totalLatency) / int64(len(g.results)))
	overallThroughput := float64(totalOperations) / totalDuration.Seconds()
	
	return ReportSummary{
		TotalTests:        len(g.results),
		TotalDuration:     totalDuration,
		TotalOperations:   totalOperations,
		OverallThroughput: overallThroughput,
		AverageLatency:    avgLatency,
		BestPerformer:     bestPerformer,
		WorstPerformer:    worstPerformer,
	}
}

// generateComparisons creates comparisons between benchmark results
func (g *BenchmarkReportGenerator) generateComparisons() []Comparison {
	if len(g.results) < 2 {
		return []Comparison{}
	}
	
	var comparisons []Comparison
	
	// Compare throughput between all pairs
	for i := 0; i < len(g.results); i++ {
		for j := i + 1; j < len(g.results); j++ {
			resultA := g.results[i]
			resultB := g.results[j]
			
			// Throughput comparison
			if resultA.ThroughputOPS > 0 && resultB.ThroughputOPS > 0 {
				diff := ((resultA.ThroughputOPS - resultB.ThroughputOPS) / resultB.ThroughputOPS) * 100
				better := resultA.TestName
				if diff < 0 {
					diff = -diff
					better = resultB.TestName
				}
				
				comparisons = append(comparisons, Comparison{
					TestA:      resultA.TestName,
					TestB:      resultB.TestName,
					Metric:     "throughput",
					Difference: diff,
					Better:     better,
				})
			}
			
			// Latency comparison
			if resultA.AvgLatency > 0 && resultB.AvgLatency > 0 {
				diff := ((float64(resultA.AvgLatency) - float64(resultB.AvgLatency)) / float64(resultB.AvgLatency)) * 100
				better := resultB.TestName // Lower latency is better
				if diff < 0 {
					diff = -diff
					better = resultA.TestName
				}
				
				comparisons = append(comparisons, Comparison{
					TestA:      resultA.TestName,
					TestB:      resultB.TestName,
					Metric:     "latency",
					Difference: diff,
					Better:     better,
				})
			}
		}
	}
	
	return comparisons
}

// WriteTextReport writes a human-readable text report
func (g *BenchmarkReportGenerator) WriteTextReport(w io.Writer) error {
	report := g.GenerateReport()
	
	// Header
	fmt.Fprintf(w, "Benchmark Report\n")
	fmt.Fprintf(w, "Generated at: %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "================\n\n")
	
	// Summary
	fmt.Fprintf(w, "Summary:\n")
	fmt.Fprintf(w, "  Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Fprintf(w, "  Total Duration: %v\n", report.Summary.TotalDuration)
	fmt.Fprintf(w, "  Total Operations: %d\n", report.Summary.TotalOperations)
	fmt.Fprintf(w, "  Overall Throughput: %.2f ops/sec\n", report.Summary.OverallThroughput)
	fmt.Fprintf(w, "  Average Latency: %v\n", report.Summary.AverageLatency)
	fmt.Fprintf(w, "  Best Performer: %s\n", report.Summary.BestPerformer)
	fmt.Fprintf(w, "  Worst Performer: %s\n", report.Summary.WorstPerformer)
	fmt.Fprintf(w, "\n")
	
	// Individual Results
	fmt.Fprintf(w, "Individual Results:\n")
	fmt.Fprintf(w, "==================\n\n")
	
	// Sort results by throughput (descending)
	sortedResults := make([]*BenchmarkResult, len(report.Results))
	copy(sortedResults, report.Results)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].ThroughputOPS > sortedResults[j].ThroughputOPS
	})
	
	for i, result := range sortedResults {
		fmt.Fprintf(w, "%d. %s\n", i+1, result.TestName)
		fmt.Fprintf(w, "   Duration: %v\n", result.Duration)
		fmt.Fprintf(w, "   Operations: %d (Success: %d, Errors: %d)\n", 
			result.TotalOps, result.SuccessOps, result.ErrorOps)
		fmt.Fprintf(w, "   Throughput: %.2f ops/sec\n", result.ThroughputOPS)
		fmt.Fprintf(w, "   Latency: avg=%v, min=%v, max=%v\n", 
			result.AvgLatency, result.MinLatency, result.MaxLatency)
		fmt.Fprintf(w, "   Percentiles: P50=%v, P95=%v, P99=%v\n", 
			result.P50Latency, result.P95Latency, result.P99Latency)
		fmt.Fprintf(w, "   Peak Connections: %d\n", result.PeakConnections)
		
		if len(result.Errors) > 0 {
			fmt.Fprintf(w, "   Errors:\n")
			for _, err := range result.Errors {
				fmt.Fprintf(w, "     - %s (count: %d)\n", err.Message, err.Count)
			}
		}
		fmt.Fprintf(w, "\n")
	}
	
	// Comparisons
	if len(report.Comparisons) > 0 {
		fmt.Fprintf(w, "Performance Comparisons:\n")
		fmt.Fprintf(w, "=======================\n\n")
		
		for _, comp := range report.Comparisons {
			fmt.Fprintf(w, "%s vs %s (%s):\n", comp.TestA, comp.TestB, comp.Metric)
			fmt.Fprintf(w, "  %s is %.2f%% better\n\n", comp.Better, comp.Difference)
		}
	}
	
	return nil
}

// WriteJSONReport writes a JSON report
func (g *BenchmarkReportGenerator) WriteJSONReport(w io.Writer) error {
	report := g.GenerateReport()
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// WriteCSVReport writes a CSV report with basic metrics
func (g *BenchmarkReportGenerator) WriteCSVReport(w io.Writer) error {
	// CSV Header
	fmt.Fprintf(w, "Test Name,Duration (ms),Total Ops,Success Ops,Error Ops,Throughput (ops/sec),Avg Latency (ms),P95 Latency (ms),P99 Latency (ms)\n")
	
	// Data rows
	for _, result := range g.results {
		fmt.Fprintf(w, "%s,%d,%d,%d,%d,%.2f,%.2f,%.2f,%.2f\n",
			escapeCSV(result.TestName),
			result.Duration.Milliseconds(),
			result.TotalOps,
			result.SuccessOps,
			result.ErrorOps,
			result.ThroughputOPS,
			float64(result.AvgLatency.Nanoseconds())/1e6,
			float64(result.P95Latency.Nanoseconds())/1e6,
			float64(result.P99Latency.Nanoseconds())/1e6)
	}
	
	return nil
}

// escapeCSV escapes CSV values
func escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\"") || strings.Contains(value, "\n") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}

// GetTopPerformers returns the top N performers by throughput
func (g *BenchmarkReportGenerator) GetTopPerformers(n int) []*BenchmarkResult {
	if len(g.results) == 0 {
		return []*BenchmarkResult{}
	}
	
	sorted := make([]*BenchmarkResult, len(g.results))
	copy(sorted, g.results)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ThroughputOPS > sorted[j].ThroughputOPS
	})
	
	if n > len(sorted) {
		n = len(sorted)
	}
	
	return sorted[:n]
}

// GetWorstPerformers returns the worst N performers by throughput
func (g *BenchmarkReportGenerator) GetWorstPerformers(n int) []*BenchmarkResult {
	if len(g.results) == 0 {
		return []*BenchmarkResult{}
	}
	
	sorted := make([]*BenchmarkResult, len(g.results))
	copy(sorted, g.results)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ThroughputOPS < sorted[j].ThroughputOPS
	})
	
	if n > len(sorted) {
		n = len(sorted)
	}
	
	return sorted[:n]
}
