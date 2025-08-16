package ygggo_mysql

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkConfig holds configuration for benchmark tests
type BenchmarkConfig struct {
	// Test execution parameters
	Duration     time.Duration `json:"duration"`      // Test duration
	Concurrency  int           `json:"concurrency"`   // Number of concurrent workers
	Iterations   int           `json:"iterations"`    // Number of iterations (0 = duration-based)
	WarmupTime   time.Duration `json:"warmup_time"`   // Warmup duration before actual test
	
	// Data parameters
	DataSize     int    `json:"data_size"`     // Size of test data
	TableName    string `json:"table_name"`    // Test table name
	CleanupData  bool   `json:"cleanup_data"`  // Whether to cleanup test data
	
	// Test behavior
	ReportInterval time.Duration `json:"report_interval"` // Interval for progress reporting
	CollectMetrics bool          `json:"collect_metrics"` // Whether to collect detailed metrics
}

// DefaultBenchmarkConfig returns default benchmark configuration
func DefaultBenchmarkConfig() BenchmarkConfig {
	return BenchmarkConfig{
		Duration:       30 * time.Second,
		Concurrency:    10,
		Iterations:     0,
		WarmupTime:     5 * time.Second,
		DataSize:       1000,
		TableName:      "benchmark_test",
		CleanupData:    true,
		ReportInterval: 5 * time.Second,
		CollectMetrics: true,
	}
}

// BenchmarkResult contains the results of a benchmark test
type BenchmarkResult struct {
	TestName     string        `json:"test_name"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	
	// Basic metrics
	TotalOps     int64         `json:"total_ops"`      // Total operations completed
	SuccessOps   int64         `json:"success_ops"`    // Successful operations
	ErrorOps     int64         `json:"error_ops"`      // Failed operations
	ThroughputOPS float64      `json:"throughput_ops"` // Operations per second
	
	// Latency metrics
	AvgLatency   time.Duration `json:"avg_latency"`    // Average latency
	MinLatency   time.Duration `json:"min_latency"`    // Minimum latency
	MaxLatency   time.Duration `json:"max_latency"`    // Maximum latency
	P50Latency   time.Duration `json:"p50_latency"`    // 50th percentile latency
	P95Latency   time.Duration `json:"p95_latency"`    // 95th percentile latency
	P99Latency   time.Duration `json:"p99_latency"`    // 99th percentile latency
	
	// Resource metrics
	PeakConnections int     `json:"peak_connections"` // Peak connection count
	AvgCPUUsage     float64 `json:"avg_cpu_usage"`    // Average CPU usage (if available)
	AvgMemoryUsage  int64   `json:"avg_memory_usage"` // Average memory usage (if available)
	
	// Error details
	Errors       []BenchmarkError `json:"errors,omitempty"`
	
	// Configuration used
	Config       BenchmarkConfig  `json:"config"`
}

// BenchmarkError represents an error that occurred during benchmarking
type BenchmarkError struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
}

// BenchmarkMetrics holds real-time metrics during benchmark execution
type BenchmarkMetrics struct {
	mutex        sync.RWMutex
	startTime    time.Time
	operations   int64
	errors       int64
	latencies    []time.Duration
	connections  int
	lastReport   time.Time
	errorDetails map[string]*BenchmarkError
}

// NewBenchmarkMetrics creates a new metrics collector
func NewBenchmarkMetrics() *BenchmarkMetrics {
	return &BenchmarkMetrics{
		startTime:    time.Now(),
		latencies:    make([]time.Duration, 0, 10000),
		lastReport:   time.Now(),
		errorDetails: make(map[string]*BenchmarkError),
	}
}

// RecordOperation records a completed operation
func (m *BenchmarkMetrics) RecordOperation(latency time.Duration, success bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.operations++
	if !success {
		m.errors++
	}

	// Store latency (with sampling for large datasets)
	if len(m.latencies) < 10000 {
		m.latencies = append(m.latencies, latency)
	} else {
		// Random sampling to keep memory usage bounded
		if m.operations%100 == 0 {
			idx := int(m.operations % 10000)
			m.latencies[idx] = latency
		}
	}
}

// RecordError records an error with details
func (m *BenchmarkMetrics) RecordError(err error) {
	if err == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	errMsg := err.Error()
	if existing, ok := m.errorDetails[errMsg]; ok {
		existing.Count++
	} else {
		m.errorDetails[errMsg] = &BenchmarkError{
			Timestamp: time.Now(),
			Message:   errMsg,
			Count:     1,
		}
	}
}

// SetConnections updates the current connection count
func (m *BenchmarkMetrics) SetConnections(count int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connections = count
}

// GetSnapshot returns a snapshot of current metrics
func (m *BenchmarkMetrics) GetSnapshot() BenchmarkSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	elapsed := time.Since(m.startTime)
	throughput := float64(m.operations) / elapsed.Seconds()
	
	return BenchmarkSnapshot{
		Elapsed:     elapsed,
		Operations:  m.operations,
		Errors:      m.errors,
		Throughput:  throughput,
		Connections: m.connections,
	}
}

// BenchmarkSnapshot represents a point-in-time metrics snapshot
type BenchmarkSnapshot struct {
	Elapsed     time.Duration `json:"elapsed"`
	Operations  int64         `json:"operations"`
	Errors      int64         `json:"errors"`
	Throughput  float64       `json:"throughput"`
	Connections int           `json:"connections"`
}

// BenchmarkTest defines the interface for benchmark tests
type BenchmarkTest interface {
	// Name returns the test name
	Name() string
	
	// Setup prepares the test environment
	Setup(ctx context.Context, pool DatabasePool) error
	
	// Run executes a single operation
	Run(ctx context.Context, pool DatabasePool, workerID int) error
	
	// Cleanup cleans up the test environment
	Cleanup(ctx context.Context, pool DatabasePool) error
}

// BenchmarkRunner executes benchmark tests
type BenchmarkRunner struct {
	config  BenchmarkConfig
	metrics *BenchmarkMetrics
	pool    DatabasePool
}

// NewBenchmarkRunner creates a new benchmark runner
func NewBenchmarkRunner(config BenchmarkConfig, pool DatabasePool) *BenchmarkRunner {
	return &BenchmarkRunner{
		config:  config,
		metrics: NewBenchmarkMetrics(),
		pool:    pool,
	}
}

// RunBenchmark executes a benchmark test
func (r *BenchmarkRunner) RunBenchmark(ctx context.Context, test BenchmarkTest) (*BenchmarkResult, error) {
	// Setup test environment
	if err := test.Setup(ctx, r.pool); err != nil {
		return nil, fmt.Errorf("setup failed: %w", err)
	}

	// Cleanup after test
	defer func() {
		if r.config.CleanupData {
			if err := test.Cleanup(ctx, r.pool); err != nil {
				fmt.Printf("Cleanup failed: %v\n", err)
			}
		}
	}()

	// Warmup phase
	if r.config.WarmupTime > 0 {
		if err := r.runWarmup(ctx, test); err != nil {
			return nil, fmt.Errorf("warmup failed: %w", err)
		}
	}

	// Reset metrics after warmup
	r.metrics = NewBenchmarkMetrics()

	// Run actual benchmark
	result, err := r.runTest(ctx, test)
	if err != nil {
		return nil, fmt.Errorf("test execution failed: %w", err)
	}

	return result, nil
}

// runWarmup executes the warmup phase
func (r *BenchmarkRunner) runWarmup(ctx context.Context, test BenchmarkTest) error {
	if r.config.WarmupTime <= 0 {
		return nil
	}

	warmupCtx, cancel := context.WithTimeout(ctx, r.config.WarmupTime)
	defer cancel()

	// Run warmup with reduced concurrency
	warmupConcurrency := max(1, r.config.Concurrency/2)

	var wg sync.WaitGroup
	for i := 0; i < warmupConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-warmupCtx.Done():
					return
				default:
					test.Run(warmupCtx, r.pool, workerID)
				}
			}
		}(i)
	}

	wg.Wait()
	return nil
}

// runTest executes the main benchmark test
func (r *BenchmarkRunner) runTest(ctx context.Context, test BenchmarkTest) (*BenchmarkResult, error) {
	startTime := time.Now()

	// Create test context
	var testCtx context.Context
	var cancel context.CancelFunc

	if r.config.Duration > 0 {
		testCtx, cancel = context.WithTimeout(ctx, r.config.Duration)
	} else {
		testCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	var operationCount int64

	for i := 0; i < r.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r.runWorker(testCtx, test, workerID, &operationCount)
		}(i)
	}

	// Start progress reporting if enabled
	var reportDone chan struct{}
	if r.config.ReportInterval > 0 {
		reportDone = make(chan struct{})
		go r.reportProgress(testCtx, reportDone)
	}

	// Wait for workers to complete
	wg.Wait()

	// Stop progress reporting
	if reportDone != nil {
		close(reportDone)
	}

	endTime := time.Now()

	// Generate result
	return r.generateResult(test.Name(), startTime, endTime), nil
}

// runWorker executes benchmark operations for a single worker
func (r *BenchmarkRunner) runWorker(ctx context.Context, test BenchmarkTest, workerID int, operationCount *int64) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check iteration limit
			if r.config.Iterations > 0 {
				current := atomic.LoadInt64(operationCount)
				if current >= int64(r.config.Iterations) {
					return
				}
			}

			// Update connection count (approximate)
			r.metrics.SetConnections(r.config.Concurrency)

			// Execute operation
			start := time.Now()
			err := test.Run(ctx, r.pool, workerID)
			latency := time.Since(start)

			// Record metrics
			r.metrics.RecordOperation(latency, err == nil)
			if err != nil {
				r.metrics.RecordError(err)
			}
			atomic.AddInt64(operationCount, 1)
		}
	}
}

// reportProgress reports benchmark progress at regular intervals
func (r *BenchmarkRunner) reportProgress(ctx context.Context, done chan struct{}) {
	ticker := time.NewTicker(r.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			snapshot := r.metrics.GetSnapshot()
			fmt.Printf("Progress: %d ops, %.2f ops/sec, %d errors\n",
				snapshot.Operations, snapshot.Throughput, snapshot.Errors)
		}
	}
}

// generateResult creates a BenchmarkResult from collected metrics
func (r *BenchmarkRunner) generateResult(testName string, startTime, endTime time.Time) *BenchmarkResult {
	snapshot := r.metrics.GetSnapshot()
	duration := endTime.Sub(startTime)

	// Calculate latency statistics
	latencies := r.getLatencies()
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var avgLatency, minLatency, maxLatency time.Duration
	var p50, p95, p99 time.Duration

	if len(latencies) > 0 {
		minLatency = latencies[0]
		maxLatency = latencies[len(latencies)-1]

		// Calculate average
		var total time.Duration
		for _, lat := range latencies {
			total += lat
		}
		avgLatency = time.Duration(int64(total) / int64(len(latencies)))

		// Calculate percentiles
		p50 = latencies[int(float64(len(latencies))*0.5)]
		p95 = latencies[int(float64(len(latencies))*0.95)]
		p99 = latencies[int(float64(len(latencies))*0.99)]
	}

	// Calculate throughput
	throughput := float64(snapshot.Operations) / duration.Seconds()

	return &BenchmarkResult{
		TestName:        testName,
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        duration,
		TotalOps:        snapshot.Operations,
		SuccessOps:      snapshot.Operations - snapshot.Errors,
		ErrorOps:        snapshot.Errors,
		ThroughputOPS:   throughput,
		AvgLatency:      avgLatency,
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
		P50Latency:      p50,
		P95Latency:      p95,
		P99Latency:      p99,
		PeakConnections: snapshot.Connections,
		Config:          r.config,
		Errors:          r.collectErrors(),
	}
}

// getLatencies returns a copy of collected latencies
func (r *BenchmarkRunner) getLatencies() []time.Duration {
	r.metrics.mutex.RLock()
	defer r.metrics.mutex.RUnlock()

	latencies := make([]time.Duration, len(r.metrics.latencies))
	copy(latencies, r.metrics.latencies)
	return latencies
}

// collectErrors collects error information
func (r *BenchmarkRunner) collectErrors() []BenchmarkError {
	r.metrics.mutex.RLock()
	defer r.metrics.mutex.RUnlock()

	errors := make([]BenchmarkError, 0, len(r.metrics.errorDetails))
	for _, err := range r.metrics.errorDetails {
		errors = append(errors, *err)
	}
	return errors
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BenchmarkSuite manages multiple benchmark tests
type BenchmarkSuite struct {
	tests  []BenchmarkTest
	config BenchmarkConfig
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(config BenchmarkConfig) *BenchmarkSuite {
	return &BenchmarkSuite{
		tests:  make([]BenchmarkTest, 0),
		config: config,
	}
}

// AddTest adds a test to the suite
func (s *BenchmarkSuite) AddTest(test BenchmarkTest) {
	s.tests = append(s.tests, test)
}

// RunAll executes all tests in the suite
func (s *BenchmarkSuite) RunAll(ctx context.Context, pool DatabasePool) ([]*BenchmarkResult, error) {
	results := make([]*BenchmarkResult, 0, len(s.tests))
	
	for _, test := range s.tests {
		runner := NewBenchmarkRunner(s.config, pool)
		result, err := runner.RunBenchmark(ctx, test)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	
	return results, nil
}
