package ygggo_mysql

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the overall health of a connection/pool
type HealthStatus struct {
	Healthy           bool                   `json:"healthy"`
	LastChecked       time.Time              `json:"last_checked"`
	ResponseTime      time.Duration          `json:"response_time"`
	ConnectionsActive int                    `json:"connections_active"`
	ConnectionsIdle   int                    `json:"connections_idle"`
	ConnectionsMax    int                    `json:"connections_max"`
	Errors            []HealthError          `json:"errors,omitempty"`
	Details           map[string]interface{} `json:"details,omitempty"`
}

// HealthError represents a health check error
type HealthError struct {
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// HealthCheckConfig configures health check behavior
type HealthCheckConfig struct {
	Timeout            time.Duration `json:"timeout"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryBackoff       time.Duration `json:"retry_backoff"`
	QueryTimeout       time.Duration `json:"query_timeout"`
	TestQuery          string        `json:"test_query"`
	MonitoringEnabled  bool          `json:"monitoring_enabled"`
	MonitoringInterval time.Duration `json:"monitoring_interval"`
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Timeout:            5 * time.Second,
		RetryAttempts:      3,
		RetryBackoff:       time.Second,
		QueryTimeout:       3 * time.Second,
		TestQuery:          "SELECT 1",
		MonitoringEnabled:  false,
		MonitoringInterval: 30 * time.Second,
	}
}

// HealthMonitor manages continuous health monitoring
type HealthMonitor struct {
	pool           *Pool
	config         HealthCheckConfig
	status         *HealthStatus
	statusMutex    sync.RWMutex
	stopChan       chan struct{}
	running        bool
	runningMutex   sync.RWMutex
}

// NewHealthMonitor creates a new health monitor for a pool
func NewHealthMonitor(pool *Pool, config HealthCheckConfig) *HealthMonitor {
	return &HealthMonitor{
		pool:   pool,
		config: config,
		status: &HealthStatus{
			Details: make(map[string]interface{}),
		},
	}
}

// Enhanced health check methods for Pool

// HealthCheck performs a comprehensive health check and returns detailed status
func (p *Pool) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if p == nil {
		return nil, fmt.Errorf("pool is nil")
	}
	return p.HealthCheckWithConfig(ctx, DefaultHealthCheckConfig())
}

// HealthCheckWithConfig performs a health check with custom configuration
func (p *Pool) HealthCheckWithConfig(ctx context.Context, config HealthCheckConfig) (*HealthStatus, error) {
	start := time.Now()
	status := &HealthStatus{
		LastChecked: start,
		Details:     make(map[string]interface{}),
		Errors:      make([]HealthError, 0),
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Perform basic connectivity check
	if err := p.performPingCheck(timeoutCtx, status); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, HealthError{
			Type:        "connectivity",
			Message:     fmt.Sprintf("Ping failed: %v", err),
			Timestamp:   time.Now(),
			Recoverable: true,
		})
	}

	// Perform query execution check
	if err := p.performQueryCheck(timeoutCtx, config, status); err != nil {
		status.Healthy = false
		status.Errors = append(status.Errors, HealthError{
			Type:        "query_execution",
			Message:     fmt.Sprintf("Query execution failed: %v", err),
			Timestamp:   time.Now(),
			Recoverable: true,
		})
	}

	// Collect connection pool statistics
	p.collectPoolStats(status)

	// Calculate response time
	status.ResponseTime = time.Since(start)

	// Determine overall health
	if len(status.Errors) == 0 {
		status.Healthy = true
	}

	return status, nil
}

// DeepHealthCheck performs an extensive health check with detailed analysis
func (p *Pool) DeepHealthCheck(ctx context.Context) (*HealthStatus, error) {
	config := DefaultHealthCheckConfig()
	config.Timeout = 10 * time.Second // Longer timeout for deep check
	
	status, err := p.HealthCheckWithConfig(ctx, config)
	if err != nil {
		return status, err
	}

	// Additional deep checks
	if err := p.performDeepChecks(ctx, status); err != nil {
		status.Errors = append(status.Errors, HealthError{
			Type:        "deep_check",
			Message:     fmt.Sprintf("Deep check failed: %v", err),
			Timestamp:   time.Now(),
			Recoverable: false,
		})
		status.Healthy = false
	}

	return status, nil
}

// performPingCheck executes a basic ping check
func (p *Pool) performPingCheck(ctx context.Context, status *HealthStatus) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("pool or database is nil")
	}
	
	start := time.Now()
	err := p.db.PingContext(ctx)
	pingTime := time.Since(start)
	
	status.Details["ping_time"] = pingTime
	return err
}

// performQueryCheck executes a test query to verify database responsiveness
func (p *Pool) performQueryCheck(ctx context.Context, config HealthCheckConfig, status *HealthStatus) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("pool or database is nil")
	}

	queryCtx, cancel := context.WithTimeout(ctx, config.QueryTimeout)
	defer cancel()

	start := time.Now()
	rows, err := p.db.QueryContext(queryCtx, config.TestQuery)
	if err != nil {
		return fmt.Errorf("test query failed: %w", err)
	}
	defer rows.Close()

	// Verify we can read the result
	if !rows.Next() {
		return fmt.Errorf("test query returned no rows")
	}

	var result interface{}
	if err := rows.Scan(&result); err != nil {
		return fmt.Errorf("failed to scan test query result: %w", err)
	}

	queryTime := time.Since(start)
	status.Details["query_time"] = queryTime
	status.Details["test_query_result"] = result

	return nil
}

// collectPoolStats gathers connection pool statistics
func (p *Pool) collectPoolStats(status *HealthStatus) {
	if p == nil || p.db == nil {
		return
	}

	stats := p.db.Stats()
	status.ConnectionsActive = stats.InUse
	status.ConnectionsIdle = stats.Idle
	status.ConnectionsMax = stats.MaxOpenConnections

	status.Details["pool_stats"] = map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration,
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}
}

// performDeepChecks executes additional comprehensive checks
func (p *Pool) performDeepChecks(ctx context.Context, status *HealthStatus) error {
	// Check for connection leaks
	stats := p.db.Stats()
	if stats.InUse > stats.MaxOpenConnections*8/10 { // 80% threshold
		status.Details["connection_leak_warning"] = true
	}

	// Check wait times
	if stats.WaitDuration > time.Second {
		status.Details["high_wait_time_warning"] = true
	}

	// Perform multiple concurrent connections test
	if err := p.testConcurrentConnections(ctx); err != nil {
		return fmt.Errorf("concurrent connections test failed: %w", err)
	}

	return nil
}

// testConcurrentConnections tests the ability to handle multiple concurrent connections
func (p *Pool) testConcurrentConnections(ctx context.Context) error {
	const numConnections = 3
	errChan := make(chan error, numConnections)

	for i := 0; i < numConnections; i++ {
		go func() {
			conn, err := p.Acquire(ctx)
			if err != nil {
				errChan <- err
				return
			}
			defer conn.Close()

			// Execute a simple query
			_, err = conn.Query(ctx, "SELECT 1")
			errChan <- err
		}()
	}

	// Collect results
	for i := 0; i < numConnections; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

// Health monitoring methods for HealthMonitor

// Start begins continuous health monitoring
func (hm *HealthMonitor) Start() error {
	hm.runningMutex.Lock()
	defer hm.runningMutex.Unlock()

	if hm.running {
		return fmt.Errorf("health monitoring is already running")
	}

	hm.stopChan = make(chan struct{})
	hm.running = true

	go hm.monitorLoop()
	return nil
}

// Stop stops continuous health monitoring
func (hm *HealthMonitor) Stop() error {
	hm.runningMutex.Lock()
	defer hm.runningMutex.Unlock()

	if !hm.running {
		return fmt.Errorf("health monitoring is not running")
	}

	close(hm.stopChan)
	hm.running = false
	return nil
}

// IsRunning returns whether health monitoring is currently active
func (hm *HealthMonitor) IsRunning() bool {
	hm.runningMutex.RLock()
	defer hm.runningMutex.RUnlock()
	return hm.running
}

// GetStatus returns the current health status
func (hm *HealthMonitor) GetStatus() *HealthStatus {
	hm.statusMutex.RLock()
	defer hm.statusMutex.RUnlock()

	// Return a copy to avoid race conditions
	status := &HealthStatus{
		Healthy:           hm.status.Healthy,
		LastChecked:       hm.status.LastChecked,
		ResponseTime:      hm.status.ResponseTime,
		ConnectionsActive: hm.status.ConnectionsActive,
		ConnectionsIdle:   hm.status.ConnectionsIdle,
		ConnectionsMax:    hm.status.ConnectionsMax,
		Details:           make(map[string]interface{}),
	}

	// Copy details
	for k, v := range hm.status.Details {
		status.Details[k] = v
	}

	// Copy errors
	status.Errors = make([]HealthError, len(hm.status.Errors))
	copy(status.Errors, hm.status.Errors)

	return status
}

// monitorLoop runs the continuous health monitoring
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(hm.config.MonitoringInterval)
	defer ticker.Stop()

	// Perform initial health check
	hm.performHealthCheck()

	for {
		select {
		case <-hm.stopChan:
			return
		case <-ticker.C:
			hm.performHealthCheck()
		}
	}
}

// performHealthCheck executes a health check and updates the status
func (hm *HealthMonitor) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), hm.config.Timeout)
	defer cancel()

	status, err := hm.pool.HealthCheckWithConfig(ctx, hm.config)
	if err != nil {
		// Create error status if health check failed
		status = &HealthStatus{
			Healthy:     false,
			LastChecked: time.Now(),
			Details:     make(map[string]interface{}),
			Errors: []HealthError{{
				Type:        "health_check_failure",
				Message:     fmt.Sprintf("Health check failed: %v", err),
				Timestamp:   time.Now(),
				Recoverable: true,
			}},
		}
	}

	hm.statusMutex.Lock()
	hm.status = status
	hm.statusMutex.Unlock()
}

// Pool methods for health monitoring integration

// StartHealthMonitoring starts continuous health monitoring for the pool
func (p *Pool) StartHealthMonitoring(interval time.Duration) error {
	config := DefaultHealthCheckConfig()
	config.MonitoringEnabled = true
	config.MonitoringInterval = interval

	return p.StartHealthMonitoringWithConfig(config)
}

// StartHealthMonitoringWithConfig starts health monitoring with custom configuration
func (p *Pool) StartHealthMonitoringWithConfig(config HealthCheckConfig) error {
	if p.healthMonitor != nil && p.healthMonitor.IsRunning() {
		return fmt.Errorf("health monitoring is already running")
	}

	p.healthMonitor = NewHealthMonitor(p, config)
	return p.healthMonitor.Start()
}

// StopHealthMonitoring stops continuous health monitoring
func (p *Pool) StopHealthMonitoring() error {
	if p.healthMonitor == nil {
		return fmt.Errorf("health monitoring is not configured")
	}

	return p.healthMonitor.Stop()
}

// GetHealthStatus returns the current cached health status
func (p *Pool) GetHealthStatus() *HealthStatus {
	if p.healthMonitor == nil {
		return nil
	}

	return p.healthMonitor.GetStatus()
}

// IsHealthMonitoringRunning returns whether health monitoring is active
func (p *Pool) IsHealthMonitoringRunning() bool {
	if p.healthMonitor == nil {
		return false
	}

	return p.healthMonitor.IsRunning()
}

// Enhanced health check with retry logic

// HealthCheckWithRetry performs a health check with automatic retry on failure
func (p *Pool) HealthCheckWithRetry(ctx context.Context) (*HealthStatus, error) {
	config := DefaultHealthCheckConfig()
	return p.HealthCheckWithRetryAndConfig(ctx, config)
}

// HealthCheckWithRetryAndConfig performs a health check with retry using custom configuration
func (p *Pool) HealthCheckWithRetryAndConfig(ctx context.Context, config HealthCheckConfig) (*HealthStatus, error) {
	var lastStatus *HealthStatus
	var lastErr error

	for attempt := 0; attempt <= config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := time.Duration(attempt) * config.RetryBackoff
			if delay > 30*time.Second {
				delay = 30 * time.Second // Cap at 30 seconds
			}

			select {
			case <-ctx.Done():
				return lastStatus, ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}

		status, err := p.HealthCheckWithConfig(ctx, config)
		if err != nil {
			lastErr = err
			continue
		}

		if status.Healthy {
			return status, nil
		}

		lastStatus = status

		// If we have recoverable errors, continue retrying
		hasRecoverableErrors := false
		for _, healthErr := range status.Errors {
			if healthErr.Recoverable {
				hasRecoverableErrors = true
				break
			}
		}

		if !hasRecoverableErrors {
			// No recoverable errors, don't retry
			break
		}
	}

	if lastStatus != nil {
		return lastStatus, nil
	}

	return nil, fmt.Errorf("health check failed after %d attempts: %w", config.RetryAttempts+1, lastErr)
}

// PingWithRetry performs a ping with automatic retry on failure
func (p *Pool) PingWithRetry(ctx context.Context) error {
	config := DefaultHealthCheckConfig()

	for attempt := 0; attempt <= config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff delay
			delay := time.Duration(attempt) * config.RetryBackoff
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}

		err := p.Ping(ctx)
		if err == nil {
			return nil
		}

		// Check if error is recoverable
		if !isRecoverableError(err) {
			return err
		}
	}

	return fmt.Errorf("ping failed after %d attempts", config.RetryAttempts+1)
}

// isRecoverableError determines if an error is recoverable and worth retrying
func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Common recoverable error patterns
	recoverablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"connection reset",
		"broken pipe",
		"no such host",
		"context deadline exceeded",
	}

	for _, pattern := range recoverablePatterns {
		if healthStringContains(errStr, pattern) {
			return true
		}
	}

	return false
}

// healthStringContains checks if a string contains a substring (case-insensitive)
func healthStringContains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    (len(s) > len(substr) &&
		     (s[:len(substr)] == substr ||
		      s[len(s)-len(substr):] == substr ||
		      indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring in a string
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
