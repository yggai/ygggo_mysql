# Connection Probe & Auto-Reconnect

The Connection Probe system provides comprehensive connection health monitoring and automatic reconnection capabilities with intelligent exponential backoff algorithms. It ensures database connectivity resilience and provides detailed monitoring and event handling.

## Features

- **Intelligent Health Probing**: Configurable interval-based connection health checks
- **Exponential Backoff Reconnection**: Smart reconnection with exponential backoff and jitter
- **Event-Driven Architecture**: Comprehensive event system for monitoring and alerting
- **Configurable Thresholds**: Customizable failure and success thresholds
- **Real-time Metrics**: Detailed connection health metrics and statistics
- **Force Operations**: Manual probe and reconnection triggers
- **Environment Presets**: Pre-configured settings for different environments

## Basic Usage

### Creating a Connection Probe

```go
// Create pool
config := ygggo.Config{
    Host:     "localhost",
    Port:     3306,
    Username: "user",
    Password: "password",
    Database: "mydb",
}

ctx := context.Background()
pool, err := ygggo.NewPool(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Create probe with default configuration
probeConfig := ygggo.DefaultProbeConfig()
probe := ygggo.NewConnectionProbe(pool, probeConfig)

// Start probing
err = probe.Start()
if err != nil {
    log.Fatal(err)
}
defer probe.Stop()
```

### Basic Probe Operations

```go
// Check if probe is running
if probe.IsRunning() {
    fmt.Println("Probe is active")
}

// Get current state
state := probe.GetState()
fmt.Printf("Status: %s\n", state.Status)
fmt.Printf("Total Probes: %d\n", state.TotalProbes)
fmt.Printf("Consecutive Failures: %d\n", state.ConsecutiveFailures)

// Force an immediate probe
err := probe.ForceProbe(ctx)
if err != nil {
    log.Printf("Force probe failed: %v", err)
}

// Get detailed metrics
metrics := probe.GetMetrics()
fmt.Printf("Success Rate: %.1f%%\n", metrics.SuccessRate)
fmt.Printf("Uptime: %v\n", metrics.Uptime)
```

## Configuration

### Default Configuration

```go
config := ygggo.DefaultProbeConfig()
// Returns:
// Interval: 30 seconds
// Timeout: 5 seconds
// FailureThreshold: 3
// SuccessThreshold: 2
// EnableAutoReconnect: true
// ReconnectPolicy: {MaxAttempts: 5, InitialBackoff: 1s, MaxBackoff: 30s, ...}
```

### Custom Configuration

```go
config := ygggo.ProbeConfig{
    Interval:         10 * time.Second,  // Probe every 10 seconds
    Timeout:          3 * time.Second,   // 3 second timeout per probe
    FailureThreshold: 3,                 // Mark unhealthy after 3 failures
    SuccessThreshold: 2,                 // Mark healthy after 2 successes
    EnableAutoReconnect: true,           // Enable auto-reconnection
    ReconnectPolicy: ygggo.ReconnectPolicy{
        MaxAttempts:       5,            // Maximum 5 reconnection attempts
        InitialBackoff:    time.Second,  // Start with 1 second backoff
        MaxBackoff:        30 * time.Second, // Maximum 30 second backoff
        BackoffMultiplier: 2.0,          // Double backoff each attempt
        Jitter:           true,          // Add random jitter
        MaxElapsed:       5 * time.Minute, // Give up after 5 minutes
    },
}

// Validate configuration
if err := ygggo.ValidateProbeConfig(config); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

### Environment-Specific Configurations

#### Development Environment
```go
devConfig := ygggo.ProbeConfig{
    Interval:         5 * time.Second,
    Timeout:          2 * time.Second,
    FailureThreshold: 2,
    SuccessThreshold: 1,
    EnableAutoReconnect: true,
    ReconnectPolicy: ygggo.ReconnectPolicy{
        MaxAttempts:       3,
        InitialBackoff:    500 * time.Millisecond,
        MaxBackoff:        5 * time.Second,
        BackoffMultiplier: 2.0,
        Jitter:           false, // Predictable for development
    },
}
```

#### Production Environment
```go
prodConfig := ygggo.ProbeConfig{
    Interval:         30 * time.Second,
    Timeout:          10 * time.Second,
    FailureThreshold: 5,
    SuccessThreshold: 3,
    EnableAutoReconnect: true,
    ReconnectPolicy: ygggo.ReconnectPolicy{
        MaxAttempts:       10,
        InitialBackoff:    2 * time.Second,
        MaxBackoff:        60 * time.Second,
        BackoffMultiplier: 2.0,
        Jitter:           true,
        MaxElapsed:       10 * time.Minute,
    },
}
```

#### Testing Environment
```go
testConfig := ygggo.ProbeConfig{
    Interval:         1 * time.Second,
    Timeout:          500 * time.Millisecond,
    FailureThreshold: 1,
    SuccessThreshold: 1,
    EnableAutoReconnect: false, // Fail fast in tests
}
```

## Exponential Backoff Algorithm

The auto-reconnection system uses an intelligent exponential backoff algorithm:

### Basic Algorithm
```
backoff = InitialBackoff * (BackoffMultiplier ^ attempt)
if backoff > MaxBackoff:
    backoff = MaxBackoff
```

### With Jitter
```
jitter = random(0, backoff * 0.1)  // Up to 10% jitter
final_backoff = backoff + jitter
```

### Example Backoff Sequence
```go
policy := ygggo.ReconnectPolicy{
    InitialBackoff:    1 * time.Second,
    MaxBackoff:        30 * time.Second,
    BackoffMultiplier: 2.0,
    Jitter:           false,
}

// Backoff sequence:
// Attempt 1: 1s
// Attempt 2: 2s
// Attempt 3: 4s
// Attempt 4: 8s
// Attempt 5: 16s
// Attempt 6: 30s (capped at MaxBackoff)
```

## Event System

### Event Types

```go
const (
    ProbeEventHealthy           // Connection is healthy
    ProbeEventUnhealthy         // Connection failed health checks
    ProbeEventReconnectStarted  // Auto-reconnection started
    ProbeEventReconnectSuccess  // Reconnection successful
    ProbeEventReconnectFailed   // Reconnection attempt failed
    ProbeEventReconnectAbandoned // Reconnection abandoned
)
```

### Event Handler Implementation

```go
type MyEventHandler struct {
    logger *log.Logger
}

func (h *MyEventHandler) HandleProbeEvent(event ygggo.ProbeEvent) {
    switch event.Type {
    case ygggo.ProbeEventUnhealthy:
        h.logger.Printf("🔴 Connection unhealthy: %s", event.Message)
        // Send alert, update dashboard, etc.
        
    case ygggo.ProbeEventReconnectStarted:
        h.logger.Printf("🔄 Reconnection started: %s", event.Message)
        
    case ygggo.ProbeEventReconnectSuccess:
        h.logger.Printf("✅ Reconnection successful: %s", event.Message)
        // Clear alerts, update status, etc.
        
    case ygggo.ProbeEventReconnectAbandoned:
        h.logger.Printf("❌ Reconnection abandoned: %s", event.Message)
        // Escalate alert, trigger manual intervention, etc.
    }
}

// Register event handler
probe.AddEventHandler(&MyEventHandler{logger: log.Default()})
```

## Probe States and Status

### Probe Status Values

```go
const (
    ProbeStatusHealthy      // Connection is healthy
    ProbeStatusUnhealthy    // Connection has failed health checks
    ProbeStatusReconnecting // Auto-reconnection in progress
    ProbeStatusFailed       // Reconnection failed, manual intervention needed
)
```

### State Information

```go
state := probe.GetState()

// Current status
fmt.Printf("Status: %s\n", state.Status)

// Timing information
fmt.Printf("Last Probe: %v\n", state.LastProbeTime)
fmt.Printf("Last Success: %v\n", state.LastSuccessTime)
fmt.Printf("Last Failure: %v\n", state.LastFailureTime)

// Counters
fmt.Printf("Total Probes: %d\n", state.TotalProbes)
fmt.Printf("Total Failures: %d\n", state.TotalFailures)
fmt.Printf("Consecutive Failures: %d\n", state.ConsecutiveFailures)
fmt.Printf("Consecutive Successes: %d\n", state.ConsecutiveSuccesses)

// Reconnection status
fmt.Printf("Is Reconnecting: %v\n", state.IsReconnecting)
fmt.Printf("Reconnect Attempts: %d\n", state.ReconnectAttempts)
```

## Metrics and Monitoring

### Detailed Metrics

```go
metrics := probe.GetMetrics()

// Performance metrics
fmt.Printf("Success Rate: %.1f%%\n", metrics.SuccessRate)
fmt.Printf("Uptime: %v\n", metrics.Uptime)
fmt.Printf("Downtime: %v\n", metrics.Downtime)

// Probe statistics
fmt.Printf("Total Probes: %d\n", metrics.TotalProbes)
fmt.Printf("Total Failures: %d\n", metrics.TotalFailures)

// Current status
fmt.Printf("Consecutive Failures: %d\n", metrics.ConsecutiveFailures)
fmt.Printf("Consecutive Successes: %d\n", metrics.ConsecutiveSuccesses)
fmt.Printf("Is Reconnecting: %v\n", metrics.IsReconnecting)
```

### Health Assessment

```go
func assessHealth(metrics ygggo.ProbeMetrics) string {
    if metrics.SuccessRate >= 95.0 && metrics.ConsecutiveFailures == 0 {
        return "Excellent"
    } else if metrics.SuccessRate >= 90.0 && metrics.ConsecutiveFailures < 3 {
        return "Good"
    } else if metrics.SuccessRate >= 80.0 {
        return "Fair"
    } else {
        return "Poor"
    }
}
```

## Advanced Usage

### Dynamic Configuration Updates

```go
// Get current configuration
currentConfig := probe.GetConfig()

// Modify configuration
newConfig := currentConfig
newConfig.Interval = 15 * time.Second
newConfig.FailureThreshold = 5

// Update configuration
err := probe.UpdateConfig(newConfig)
if err != nil {
    log.Printf("Failed to update config: %v", err)
}
```

### Force Reconnection

```go
// Force immediate reconnection
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := probe.ForceReconnect(ctx)
if err != nil {
    log.Printf("Force reconnection failed: %v", err)
} else {
    log.Println("Force reconnection successful")
}
```

### Multiple Event Handlers

```go
// Add multiple event handlers
probe.AddEventHandler(&AlertHandler{})
probe.AddEventHandler(&MetricsHandler{})
probe.AddEventHandler(&LoggingHandler{})

// Remove specific handler
probe.RemoveEventHandler(alertHandler)
```

## Best Practices

### 1. Environment-Appropriate Configuration

```go
func createProbeConfig(env string) ygggo.ProbeConfig {
    switch env {
    case "production":
        return productionProbeConfig()
    case "staging":
        return stagingProbeConfig()
    case "testing":
        return testingProbeConfig()
    default:
        return developmentProbeConfig()
    }
}
```

### 2. Comprehensive Event Handling

```go
type ComprehensiveEventHandler struct {
    alerter   Alerter
    metrics   MetricsCollector
    logger    Logger
}

func (h *ComprehensiveEventHandler) HandleProbeEvent(event ygggo.ProbeEvent) {
    // Always log
    h.logger.LogEvent(event)
    
    // Update metrics
    h.metrics.RecordProbeEvent(event)
    
    // Handle critical events
    switch event.Type {
    case ygggo.ProbeEventUnhealthy:
        h.alerter.SendAlert("Database connection unhealthy", event)
    case ygggo.ProbeEventReconnectAbandoned:
        h.alerter.SendCriticalAlert("Database reconnection failed", event)
    }
}
```

### 3. Graceful Shutdown

```go
func gracefulShutdown(probe *ygggo.ConnectionProbe) {
    // Stop probing
    if err := probe.Stop(); err != nil {
        log.Printf("Error stopping probe: %v", err)
    }
    
    // Wait for any ongoing operations
    time.Sleep(100 * time.Millisecond)
    
    log.Println("Probe shutdown complete")
}
```

### 4. Health Check Integration

```go
// HTTP health endpoint
func healthHandler(probe *ygggo.ConnectionProbe) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        state := probe.GetState()
        metrics := probe.GetMetrics()
        
        health := map[string]interface{}{
            "status":       state.Status.String(),
            "success_rate": metrics.SuccessRate,
            "uptime":       metrics.Uptime.String(),
            "last_probe":   state.LastProbeTime,
        }
        
        if state.Status != ygggo.ProbeStatusHealthy {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        
        json.NewEncoder(w).Encode(health)
    }
}
```

## API Reference

### Core Types

- `ProbeConfig` - Probe configuration settings
- `ReconnectPolicy` - Reconnection strategy configuration
- `ProbeState` - Current probe state information
- `ProbeMetrics` - Detailed probe performance metrics
- `ProbeEvent` - Probe event information
- `ProbeEventHandler` - Event handler interface

### Main Methods

- `NewConnectionProbe(pool, config)` - Create new connection probe
- `Start()` - Start probing
- `Stop()` - Stop probing
- `IsRunning()` - Check if probe is active
- `GetState()` - Get current probe state
- `GetConfig()` - Get current configuration
- `GetMetrics()` - Get detailed metrics
- `ForceProbe(ctx)` - Force immediate probe
- `ForceReconnect(ctx)` - Force immediate reconnection
- `UpdateConfig(config)` - Update probe configuration
- `AddEventHandler(handler)` - Add event handler
- `RemoveEventHandler(handler)` - Remove event handler

### Configuration Methods

- `DefaultProbeConfig()` - Get default configuration
- `ValidateProbeConfig(config)` - Validate configuration
- `ValidateReconnectPolicy(policy)` - Validate reconnect policy

### Auto-Reconnector Methods

- `NewAutoReconnector(pool, policy)` - Create auto-reconnector
- `Reconnect(ctx)` - Perform reconnection with backoff
- `GetState()` - Get reconnection state
- `IsActive()` - Check if reconnection is active

## Troubleshooting

### High Failure Rate

```go
metrics := probe.GetMetrics()
if metrics.SuccessRate < 90.0 {
    log.Printf("High failure rate: %.1f%%", metrics.SuccessRate)
    
    // Check network connectivity
    // Verify database server status
    // Review timeout settings
    // Consider increasing failure threshold
}
```

### Frequent Reconnections

```go
state := probe.GetState()
if state.ReconnectAttempts > 10 {
    log.Printf("Frequent reconnections: %d attempts", state.ReconnectAttempts)
    
    // Check for network instability
    // Review reconnection policy
    // Consider increasing success threshold
    // Monitor database server health
}
```

### Probe Not Starting

```go
err := probe.Start()
if err != nil {
    log.Printf("Failed to start probe: %v", err)
    
    // Check if probe is already running
    // Verify configuration is valid
    // Ensure pool is not nil
    // Check for resource constraints
}
```
