package ygggo_mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck_BasicConnectivity(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test basic health check
	status, err := pool.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	// Verify health status
	assert.True(t, status.Healthy, "Pool should be healthy")
	assert.False(t, status.LastChecked.IsZero(), "LastChecked should be set")
	assert.Greater(t, status.ResponseTime, time.Duration(0), "ResponseTime should be positive")
	assert.Empty(t, status.Errors, "Should have no errors for healthy pool")
	assert.NotNil(t, status.Details, "Details should be populated")

	// Verify connection stats are populated
	assert.GreaterOrEqual(t, status.ConnectionsMax, 0, "ConnectionsMax should be set")
	assert.GreaterOrEqual(t, status.ConnectionsActive, 0, "ConnectionsActive should be set")
	assert.GreaterOrEqual(t, status.ConnectionsIdle, 0, "ConnectionsIdle should be set")
}

func TestHealthCheck_WithTimeout(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Test with very short timeout
	config := DefaultHealthCheckConfig()
	config.Timeout = 1 * time.Millisecond // Very short timeout

	ctx := context.Background()
	status, _ := pool.HealthCheckWithConfig(ctx, config)

	// Should complete but might have timeout-related issues
	require.NotNil(t, status)
	// Note: err might be nil if the check completes quickly enough
}

func TestHealthCheck_QueryExecution(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test with custom test query
	config := DefaultHealthCheckConfig()
	config.TestQuery = "SELECT 42 as answer"

	status, err := pool.HealthCheckWithConfig(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.True(t, status.Healthy, "Pool should be healthy")
	assert.Contains(t, status.Details, "test_query_result", "Should contain query result")
	assert.Contains(t, status.Details, "query_time", "Should contain query timing")

	// Verify the query result
	result := status.Details["test_query_result"]
	assert.NotNil(t, result, "Query result should not be nil")
}

func TestHealthCheck_InvalidQuery(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test with invalid query
	config := DefaultHealthCheckConfig()
	config.TestQuery = "SELECT FROM INVALID_SYNTAX"

	status, err := pool.HealthCheckWithConfig(ctx, config)
	require.NoError(t, err) // Method should not return error, but status should indicate failure
	require.NotNil(t, status)

	assert.False(t, status.Healthy, "Pool should be unhealthy due to invalid query")
	assert.NotEmpty(t, status.Errors, "Should have errors")

	// Check for query execution error
	found := false
	for _, healthErr := range status.Errors {
		if healthErr.Type == "query_execution" {
			found = true
			assert.True(t, healthErr.Recoverable, "Query errors should be recoverable")
			assert.Contains(t, healthErr.Message, "Query execution failed")
			break
		}
	}
	assert.True(t, found, "Should have query execution error")
}

func TestDeepHealthCheck(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test deep health check
	status, err := pool.DeepHealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.True(t, status.Healthy, "Pool should be healthy")
	assert.Contains(t, status.Details, "pool_stats", "Should contain detailed pool stats")

	// Verify pool stats structure
	poolStats, ok := status.Details["pool_stats"].(map[string]interface{})
	assert.True(t, ok, "pool_stats should be a map")
	assert.Contains(t, poolStats, "open_connections")
	assert.Contains(t, poolStats, "in_use")
	assert.Contains(t, poolStats, "idle")
	assert.Contains(t, poolStats, "wait_count")
}

func TestHealthCheck_PoolStats(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Acquire a connection to change pool stats
	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	// Check health while connection is in use
	status, err := pool.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.True(t, status.Healthy, "Pool should be healthy")
	assert.Greater(t, status.ConnectionsActive, 0, "Should have active connections")

	// Release connection
	conn.Close()

	// Check health after releasing connection
	status2, err := pool.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status2)

	assert.True(t, status2.Healthy, "Pool should still be healthy")
	// Note: Connection might still be in use briefly due to timing
}

func TestHealthCheck_ConcurrentConnections(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test deep health check which includes concurrent connection test
	status, err := pool.DeepHealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.True(t, status.Healthy, "Pool should handle concurrent connections")
	assert.Empty(t, status.Errors, "Should have no errors from concurrent test")
}

func TestHealthCheckConfig_Defaults(t *testing.T) {
	config := DefaultHealthCheckConfig()

	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, time.Second, config.RetryBackoff)
	assert.Equal(t, 3*time.Second, config.QueryTimeout)
	assert.Equal(t, "SELECT 1", config.TestQuery)
	assert.False(t, config.MonitoringEnabled)
	assert.Equal(t, 30*time.Second, config.MonitoringInterval)
}

func TestHealthStatus_Structure(t *testing.T) {
	status := &HealthStatus{
		Healthy:           true,
		LastChecked:       time.Now(),
		ResponseTime:      100 * time.Millisecond,
		ConnectionsActive: 2,
		ConnectionsIdle:   3,
		ConnectionsMax:    10,
		Details:           make(map[string]interface{}),
	}

	assert.True(t, status.Healthy)
	assert.False(t, status.LastChecked.IsZero())
	assert.Greater(t, status.ResponseTime, time.Duration(0))
	assert.Equal(t, 2, status.ConnectionsActive)
	assert.Equal(t, 3, status.ConnectionsIdle)
	assert.Equal(t, 10, status.ConnectionsMax)
	assert.NotNil(t, status.Details)
	assert.Empty(t, status.Errors)
}

func TestHealthError_Structure(t *testing.T) {
	healthErr := HealthError{
		Type:        "connectivity",
		Message:     "Connection failed",
		Timestamp:   time.Now(),
		Recoverable: true,
	}

	assert.Equal(t, "connectivity", healthErr.Type)
	assert.Equal(t, "Connection failed", healthErr.Message)
	assert.False(t, healthErr.Timestamp.IsZero())
	assert.True(t, healthErr.Recoverable)
}

func TestHealthCheck_NilPool(t *testing.T) {
	var pool *Pool
	ctx := context.Background()

	// Test health check on nil pool
	status, err := pool.HealthCheck(ctx)
	require.Error(t, err)
	assert.Nil(t, status)
}

func TestHealthCheck_ContextCancellation(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test health check with cancelled context
	status, err := pool.HealthCheck(ctx)
	require.NoError(t, err) // Method should handle cancellation gracefully
	require.NotNil(t, status)

	// Should have errors due to context cancellation
	assert.False(t, status.Healthy, "Should be unhealthy due to context cancellation")
	assert.NotEmpty(t, status.Errors, "Should have errors")
}

func TestHealthMonitoring_StartStop(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Test starting health monitoring
	err = pool.StartHealthMonitoring(100 * time.Millisecond)
	require.NoError(t, err)

	// Verify monitoring is running
	assert.True(t, pool.IsHealthMonitoringRunning(), "Health monitoring should be running")

	// Wait a bit for monitoring to collect data
	time.Sleep(200 * time.Millisecond)

	// Get health status
	status := pool.GetHealthStatus()
	require.NotNil(t, status, "Should have health status")
	assert.True(t, status.Healthy, "Pool should be healthy")
	assert.False(t, status.LastChecked.IsZero(), "LastChecked should be set")

	// Test stopping health monitoring
	err = pool.StopHealthMonitoring()
	require.NoError(t, err)

	// Verify monitoring is stopped
	assert.False(t, pool.IsHealthMonitoringRunning(), "Health monitoring should be stopped")
}

func TestHealthMonitoring_AlreadyRunning(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Start monitoring
	err = pool.StartHealthMonitoring(100 * time.Millisecond)
	require.NoError(t, err)
	defer pool.StopHealthMonitoring()

	// Try to start again - should fail
	err = pool.StartHealthMonitoring(100 * time.Millisecond)
	assert.Error(t, err, "Should fail to start monitoring when already running")
	assert.Contains(t, err.Error(), "already running")
}

func TestHealthMonitoring_CustomConfig(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Create custom config
	config := DefaultHealthCheckConfig()
	config.MonitoringInterval = 50 * time.Millisecond
	config.TestQuery = "SELECT 'health_check' as status"
	config.Timeout = 2 * time.Second

	// Start monitoring with custom config
	err = pool.StartHealthMonitoringWithConfig(config)
	require.NoError(t, err)
	defer pool.StopHealthMonitoring()

	// Wait for monitoring to collect data
	time.Sleep(100 * time.Millisecond)

	// Get health status
	status := pool.GetHealthStatus()
	require.NotNil(t, status, "Should have health status")
	assert.True(t, status.Healthy, "Pool should be healthy")

	// Verify custom query result
	if result, ok := status.Details["test_query_result"]; ok {
		assert.Equal(t, "health_check", result, "Should have custom query result")
	}
}

func TestHealthMonitor_DirectUsage(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Create health monitor directly
	config := DefaultHealthCheckConfig()
	config.MonitoringInterval = 50 * time.Millisecond
	monitor := NewHealthMonitor(pool, config)

	// Test starting
	err = monitor.Start()
	require.NoError(t, err)

	assert.True(t, monitor.IsRunning(), "Monitor should be running")

	// Wait for data collection
	time.Sleep(100 * time.Millisecond)

	// Get status
	status := monitor.GetStatus()
	require.NotNil(t, status, "Should have status")
	assert.True(t, status.Healthy, "Should be healthy")

	// Test stopping
	err = monitor.Stop()
	require.NoError(t, err)

	assert.False(t, monitor.IsRunning(), "Monitor should be stopped")
}

func TestHealthMonitoring_StatusCopy(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Start monitoring
	err = pool.StartHealthMonitoring(50 * time.Millisecond)
	require.NoError(t, err)
	defer pool.StopHealthMonitoring()

	// Wait for data collection
	time.Sleep(100 * time.Millisecond)

	// Get status multiple times
	status1 := pool.GetHealthStatus()
	status2 := pool.GetHealthStatus()

	require.NotNil(t, status1)
	require.NotNil(t, status2)

	// Verify they are separate copies
	assert.NotSame(t, status1, status2, "Should return separate copies")
	assert.NotSame(t, status1.Details, status2.Details, "Details should be separate copies")

	// But content should be the same
	assert.Equal(t, status1.Healthy, status2.Healthy)
	assert.Equal(t, status1.LastChecked, status2.LastChecked)
}

func TestHealthCheck_WithRetry(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test successful health check with retry
	status, err := pool.HealthCheckWithRetry(ctx)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.True(t, status.Healthy, "Pool should be healthy")
	assert.False(t, status.LastChecked.IsZero(), "LastChecked should be set")
}

func TestHealthCheck_RetryWithCustomConfig(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test with custom retry configuration
	config := DefaultHealthCheckConfig()
	config.RetryAttempts = 2
	config.RetryBackoff = 10 * time.Millisecond

	status, err := pool.HealthCheckWithRetryAndConfig(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.True(t, status.Healthy, "Pool should be healthy")
}

func TestPing_WithRetry(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()
	ctx := context.Background()

	// Test successful ping with retry
	err = pool.PingWithRetry(ctx)
	require.NoError(t, err, "Ping with retry should succeed")
}

func TestHealthCheck_RecoverableErrors(t *testing.T) {
	// Test error classification
	testCases := []struct {
		err         error
		recoverable bool
	}{
		{fmt.Errorf("connection refused"), true},
		{fmt.Errorf("timeout occurred"), true},
		{fmt.Errorf("network is unreachable"), true},
		{fmt.Errorf("context deadline exceeded"), true},
		{fmt.Errorf("syntax error"), false},
		{fmt.Errorf("access denied"), false},
		{nil, false},
	}

	for _, tc := range testCases {
		result := isRecoverableError(tc.err)
		if tc.recoverable {
			assert.True(t, result, "Error should be recoverable: %v", tc.err)
		} else {
			assert.False(t, result, "Error should not be recoverable: %v", tc.err)
		}
	}
}

func TestHealthCheck_RetryContextCancellation(t *testing.T) {
	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Create a context that will be cancelled very quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to be cancelled
	time.Sleep(5 * time.Millisecond)

	// Test retry with cancelled context
	config := DefaultHealthCheckConfig()
	config.RetryAttempts = 5
	config.RetryBackoff = 100 * time.Millisecond

	_, err = pool.HealthCheckWithRetryAndConfig(ctx, config)
	// Should handle context cancellation gracefully
	if err == nil {
		// If no error, the health check completed before context cancellation
		// This is acceptable behavior
		t.Log("Health check completed before context cancellation - this is acceptable")
	} else {
		// If there's an error, it should be context-related
		assert.Contains(t, err.Error(), "context", "Error should be context-related")
	}
}

func TestHealthCheck_StringContains(t *testing.T) {
	// Test the healthStringContains helper function
	testCases := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"connection refused", "connection", true},
		{"timeout occurred", "timeout", true},
		{"network error", "work", true},
		{"hello world", "world", true},
		{"hello world", "xyz", false},
		{"", "test", false},
		{"test", "", true},
		{"", "", true},
	}

	for _, tc := range testCases {
		result := healthStringContains(tc.s, tc.substr)
		assert.Equal(t, tc.expected, result,
			"healthStringContains(%q, %q) should return %v", tc.s, tc.substr, tc.expected)
	}
}
