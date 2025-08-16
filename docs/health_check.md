# Connection Health Check

The Health Check system provides comprehensive monitoring and diagnostics for database connections and connection pools. It includes basic connectivity checks, query execution validation, connection pool analysis, continuous monitoring, and automatic retry mechanisms.

## Features

- **Basic Health Checks**: Simple ping and query execution validation
- **Deep Health Checks**: Comprehensive analysis including pool statistics and performance metrics
- **Continuous Monitoring**: Background health monitoring with configurable intervals
- **Retry Logic**: Automatic retry with exponential backoff for transient failures
- **Detailed Status Reporting**: Rich health status information with metrics and error details
- **JSON Serialization**: Health status can be easily serialized for APIs and logging

## Basic Usage

### Simple Health Check

```go
ctx := context.Background()
status, err := pool.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
    return
}

if status.Healthy {
    fmt.Printf("✅ Pool is healthy (Response time: %v)\n", status.ResponseTime)
} else {
    fmt.Printf("❌ Pool is unhealthy (%d errors)\n", len(status.Errors))
}
```

### Health Check with Custom Configuration

```go
config := ygggo.DefaultHealthCheckConfig()
config.Timeout = 10 * time.Second
config.TestQuery = "SELECT 'custom_check' as status"
config.QueryTimeout = 5 * time.Second

status, err := pool.HealthCheckWithConfig(ctx, config)
```

### Deep Health Check

```go
status, err := pool.DeepHealthCheck(ctx)
if err != nil {
    return err
}

// Access detailed pool statistics
if poolStats, ok := status.Details["pool_stats"].(map[string]interface{}); ok {
    fmt.Printf("Open connections: %v\n", poolStats["open_connections"])
    fmt.Printf("Wait count: %v\n", poolStats["wait_count"])
}
```

## Continuous Monitoring

### Start/Stop Monitoring

```go
// Start monitoring with 30-second intervals
err := pool.StartHealthMonitoring(30 * time.Second)
if err != nil {
    return err
}

// Check if monitoring is running
if pool.IsHealthMonitoringRunning() {
    fmt.Println("Health monitoring is active")
}

// Get cached health status
status := pool.GetHealthStatus()
if status != nil {
    fmt.Printf("Last check: %v\n", status.LastChecked)
}

// Stop monitoring
err = pool.StopHealthMonitoring()
```

### Custom Monitoring Configuration

```go
config := ygggo.DefaultHealthCheckConfig()
config.MonitoringInterval = 15 * time.Second
config.Timeout = 5 * time.Second
config.RetryAttempts = 3
config.TestQuery = "SELECT CONNECTION_ID()"

err := pool.StartHealthMonitoringWithConfig(config)
```

## Retry Logic

### Health Check with Retry

```go
// Automatic retry with exponential backoff
status, err := pool.HealthCheckWithRetry(ctx)
if err != nil {
    log.Printf("Health check failed after retries: %v", err)
    return
}
```

### Ping with Retry

```go
// Simple ping with retry
err := pool.PingWithRetry(ctx)
if err != nil {
    log.Printf("Ping failed after retries: %v", err)
}
```

### Custom Retry Configuration

```go
config := ygggo.DefaultHealthCheckConfig()
config.RetryAttempts = 5
config.RetryBackoff = 2 * time.Second

status, err := pool.HealthCheckWithRetryAndConfig(ctx, config)
```

## Health Status Structure

### HealthStatus Fields

```go
type HealthStatus struct {
    Healthy           bool                   // Overall health status
    LastChecked       time.Time              // When the check was performed
    ResponseTime      time.Duration          // Total check duration
    ConnectionsActive int                    // Active connections
    ConnectionsIdle   int                    // Idle connections
    ConnectionsMax    int                    // Maximum connections
    Errors            []HealthError          // Any errors encountered
    Details           map[string]interface{} // Additional details
}
```

### HealthError Structure

```go
type HealthError struct {
    Type        string    // Error category (e.g., "connectivity", "query_execution")
    Message     string    // Human-readable error message
    Timestamp   time.Time // When the error occurred
    Recoverable bool      // Whether the error is likely recoverable
}
```

### Health Details

The `Details` map contains additional information:

- `ping_time`: Duration of the ping operation
- `query_time`: Duration of the test query
- `test_query_result`: Result of the test query
- `pool_stats`: Detailed connection pool statistics
- `connection_leak_warning`: Boolean indicating potential connection leaks
- `high_wait_time_warning`: Boolean indicating high connection wait times

## Configuration

### HealthCheckConfig Options

```go
type HealthCheckConfig struct {
    Timeout            time.Duration // Overall timeout for health check
    RetryAttempts      int           // Number of retry attempts
    RetryBackoff       time.Duration // Base delay between retries
    QueryTimeout       time.Duration // Timeout for test query
    TestQuery          string        // SQL query to execute
    MonitoringEnabled  bool          // Enable continuous monitoring
    MonitoringInterval time.Duration // Interval between monitoring checks
}
```

### Default Configuration

```go
config := ygggo.DefaultHealthCheckConfig()
// Returns:
// Timeout: 5 seconds
// RetryAttempts: 3
// RetryBackoff: 1 second
// QueryTimeout: 3 seconds
// TestQuery: "SELECT 1"
// MonitoringEnabled: false
// MonitoringInterval: 30 seconds
```

## Error Handling and Recovery

### Recoverable vs Non-Recoverable Errors

The health check system automatically classifies errors:

**Recoverable Errors** (will trigger retries):
- Connection refused
- Network timeouts
- Temporary network failures
- Context deadline exceeded
- Connection reset

**Non-Recoverable Errors** (will not trigger retries):
- Authentication failures
- SQL syntax errors
- Permission denied

### Custom Error Classification

```go
// Check if an error is recoverable
if isRecoverableError(err) {
    // Implement custom retry logic
}
```

## Integration Examples

### HTTP Health Endpoint

```go
func healthHandler(pool *ygggo.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
        defer cancel()

        status, err := pool.HealthCheck(ctx)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        if !status.Healthy {
            w.WriteHeader(http.StatusServiceUnavailable)
        }

        json.NewEncoder(w).Encode(status)
    }
}
```

### Kubernetes Liveness Probe

```go
func livenessProbe(pool *ygggo.Pool) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := pool.PingWithRetry(ctx)
    return err == nil
}
```

### Metrics Integration

```go
func collectHealthMetrics(pool *ygggo.Pool) {
    status := pool.GetHealthStatus()
    if status == nil {
        return
    }

    // Export metrics to your monitoring system
    healthGauge.Set(boolToFloat(status.Healthy))
    responseTimeHistogram.Observe(status.ResponseTime.Seconds())
    activeConnectionsGauge.Set(float64(status.ConnectionsActive))
    idleConnectionsGauge.Set(float64(status.ConnectionsIdle))
}
```

## Best Practices

1. **Use Appropriate Timeouts**: Set reasonable timeouts based on your application's requirements
2. **Monitor Continuously**: Enable continuous monitoring for production systems
3. **Handle Errors Gracefully**: Always check both the error and health status
4. **Configure Retries**: Use retry logic for transient network issues
5. **Monitor Pool Statistics**: Watch for connection leaks and high wait times
6. **Integrate with Monitoring**: Export health metrics to your monitoring system
7. **Use Deep Checks Sparingly**: Deep health checks are more resource-intensive

## Troubleshooting

### Common Issues

**High Response Times**:
- Check network latency to database
- Verify database performance
- Review connection pool configuration

**Connection Pool Warnings**:
- Monitor for connection leaks
- Adjust pool size if needed
- Check application connection usage patterns

**Frequent Health Check Failures**:
- Review database logs
- Check network connectivity
- Verify database resource availability
- Adjust timeout and retry settings

### Debug Information

Enable detailed logging to troubleshoot health check issues:

```go
// Health status contains detailed timing and error information
status, _ := pool.DeepHealthCheck(ctx)
for _, err := range status.Errors {
    log.Printf("Health error: %s - %s (recoverable: %v)", 
        err.Type, err.Message, err.Recoverable)
}
```
