# Connection Pool Manager

The Connection Pool Manager provides comprehensive management and monitoring capabilities for MySQL connection pools. It includes advanced configuration options, real-time statistics, dynamic scaling, health monitoring, and connection lifecycle management.

## Features

- **Enhanced Pool Configuration**: Comprehensive pool settings with validation and presets
- **Real-time Statistics**: Detailed pool metrics and performance monitoring
- **Dynamic Pool Management**: Runtime scaling, resizing, and configuration updates
- **Health Monitoring**: Comprehensive pool health checks and issue detection
- **Connection Lifecycle**: Connection warming, draining, and leak detection
- **Environment Presets**: Pre-configured settings for development, production, testing, and high-performance scenarios
- **Validation**: Comprehensive configuration validation with clear error messages

## Basic Usage

### Creating a Pool Manager

```go
// Create pool with configuration
config := ygggo.Config{
    Host:     "localhost",
    Port:     3306,
    Username: "user",
    Password: "password",
    Database: "mydb",
    Pool:     ygggo.ProductionPoolConfig(),
}

ctx := context.Background()
pool, err := ygggo.NewPool(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Create pool manager
manager := ygggo.NewPoolManager(pool)
```

### Basic Pool Operations

```go
// Get current configuration
config := manager.GetConfig()
fmt.Printf("MaxOpen: %d, MaxIdle: %d\n", config.MaxOpen, config.MaxIdle)

// Get pool statistics
stats := manager.Stats()
fmt.Printf("In Use: %d, Idle: %d, Utilization: %.1f%%\n", 
    stats.InUse, stats.Idle, stats.ConnectionUtilization)

// Perform health check
health, err := manager.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
} else {
    fmt.Printf("Pool healthy: %v\n", health.Healthy)
}
```

## Pool Configuration

### Default Configuration

```go
config := ygggo.DefaultPoolConfig()
// Returns:
// MaxOpen: 25
// MaxIdle: 10
// ConnMaxLifetime: 30 minutes
// ConnMaxIdleTime: 10 minutes
```

### Configuration Validation

```go
config := ygggo.PoolConfig{
    MaxOpen:         25,
    MaxIdle:         10,
    ConnMaxLifetime: 30 * time.Minute,
    ConnMaxIdleTime: 10 * time.Minute,
}

if err := ygggo.ValidatePoolConfig(config); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

### Environment Presets

#### Development Preset
```go
config := ygggo.DevelopmentPoolConfig()
// Optimized for local development:
// - Limited connections (MaxOpen: 5, MaxIdle: 2)
// - Shorter connection lifetime (10 minutes)
// - Quick connection recycling
```

#### Production Preset
```go
config := ygggo.ProductionPoolConfig()
// Optimized for production:
// - More connections (MaxOpen: 50, MaxIdle: 20)
// - Longer connection lifetime (60 minutes)
// - Conservative settings for stability
```

#### Testing Preset
```go
config := ygggo.TestingPoolConfig()
// Optimized for testing:
// - Minimal connections (MaxOpen: 3, MaxIdle: 1)
// - Short connection lifetime (2 minutes)
// - Fast connection recycling
```

#### High Performance Preset
```go
config := ygggo.HighPerformancePoolConfig()
// Optimized for high performance:
// - Many connections (MaxOpen: 100, MaxIdle: 50)
// - Long connection lifetime (120 minutes)
// - Performance-focused settings
```

## Pool Statistics and Monitoring

### Real-time Statistics

```go
stats := manager.Stats()

// Basic connection statistics
fmt.Printf("Open Connections: %d\n", stats.OpenConnections)
fmt.Printf("In Use: %d\n", stats.InUse)
fmt.Printf("Idle: %d\n", stats.Idle)

// Wait statistics
fmt.Printf("Wait Count: %d\n", stats.WaitCount)
fmt.Printf("Wait Duration: %v\n", stats.WaitDuration)
fmt.Printf("Average Wait Time: %v\n", stats.AverageWaitTime)

// Connection lifecycle statistics
fmt.Printf("Max Idle Closed: %d\n", stats.MaxIdleClosed)
fmt.Printf("Max Lifetime Closed: %d\n", stats.MaxLifetimeClosed)
fmt.Printf("Max Idle Time Closed: %d\n", stats.MaxIdleTimeClosed)

// Enhanced statistics
fmt.Printf("Total Connections: %d\n", stats.TotalConnections)
fmt.Printf("Failed Connections: %d\n", stats.FailedConnections)
fmt.Printf("Leaked Connections: %d\n", stats.LeakedConnections)
fmt.Printf("Connection Utilization: %.1f%%\n", stats.ConnectionUtilization)
```

### Health Monitoring

```go
health, err := manager.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
    return
}

fmt.Printf("Healthy: %v\n", health.Healthy)
fmt.Printf("Last Checked: %v\n", health.LastChecked)
fmt.Printf("Response Time: %v\n", health.ResponseTime)

if len(health.Issues) > 0 {
    fmt.Println("Issues detected:")
    for _, issue := range health.Issues {
        fmt.Printf("  - %s\n", issue)
    }
}

// Quick health check
if manager.IsHealthy() {
    fmt.Println("✅ Pool is healthy")
} else {
    fmt.Println("❌ Pool has issues")
}
```

## Dynamic Pool Management

### Pool Scaling

```go
// Scale up - add more connections
err := manager.ScaleUp(10)
if err != nil {
    log.Printf("Scale up failed: %v", err)
}

// Scale down - reduce connections
err = manager.ScaleDown(5)
if err != nil {
    log.Printf("Scale down failed: %v", err)
}

// Resize - set specific limits
err = manager.Resize(30, 15) // MaxOpen: 30, MaxIdle: 15
if err != nil {
    log.Printf("Resize failed: %v", err)
}
```

### Configuration Updates

```go
// Get current configuration
currentConfig := manager.GetConfig()

// Modify configuration
newConfig := currentConfig
newConfig.MaxOpen = 50
newConfig.ConnMaxLifetime = 60 * time.Minute

// Update configuration
err := manager.UpdateConfig(newConfig)
if err != nil {
    log.Printf("Config update failed: %v", err)
}
```

## Connection Lifecycle Management

### Connection Warming

```go
// Pre-create connections up to MaxIdle
err := manager.WarmUp(ctx)
if err != nil {
    log.Printf("Warm up failed: %v", err)
} else {
    fmt.Println("Pool warmed up successfully")
}
```

### Connection Draining

```go
// Gracefully close idle connections
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := manager.DrainConnections(ctx)
if err != nil {
    log.Printf("Drain failed: %v", err)
} else {
    fmt.Println("Connections drained successfully")
}
```

### Connection Utilization Monitoring

```go
utilization := manager.GetConnectionUtilization()
fmt.Printf("Current utilization: %.1f%%\n", utilization)

if utilization > 90 {
    fmt.Println("⚠️  High utilization - consider scaling up")
} else if utilization < 10 {
    fmt.Println("💡 Low utilization - consider scaling down")
}
```

## Best Practices

### 1. Choose Appropriate Presets

```go
// Use environment-specific presets
var poolConfig ygggo.PoolConfig

switch os.Getenv("ENVIRONMENT") {
case "production":
    poolConfig = ygggo.ProductionPoolConfig()
case "testing":
    poolConfig = ygggo.TestingPoolConfig()
default:
    poolConfig = ygggo.DevelopmentPoolConfig()
}
```

### 2. Monitor Pool Health

```go
// Regular health monitoring
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        if !manager.IsHealthy() {
            log.Println("⚠️  Pool health issue detected")
            
            health, _ := manager.HealthCheck(context.Background())
            for _, issue := range health.Issues {
                log.Printf("Issue: %s", issue)
            }
        }
    }
}()
```

### 3. Dynamic Scaling Based on Load

```go
// Auto-scaling based on utilization
utilization := manager.GetConnectionUtilization()

if utilization > 85 {
    // Scale up when utilization is high
    manager.ScaleUp(5)
    log.Println("Scaled up due to high utilization")
} else if utilization < 20 {
    // Scale down when utilization is low
    manager.ScaleDown(2)
    log.Println("Scaled down due to low utilization")
}
```

### 4. Validate Configuration Changes

```go
// Always validate before applying
newConfig := manager.GetConfig()
newConfig.MaxOpen = 100

if err := ygggo.ValidatePoolConfig(newConfig); err != nil {
    log.Printf("Invalid configuration: %v", err)
    return
}

if err := manager.UpdateConfig(newConfig); err != nil {
    log.Printf("Failed to update configuration: %v", err)
}
```

### 5. Warm Up Connections at Startup

```go
// Warm up pool after creation
pool, err := ygggo.NewPool(ctx, config)
if err != nil {
    log.Fatal(err)
}

manager := ygggo.NewPoolManager(pool)

// Pre-create connections for faster initial requests
if err := manager.WarmUp(ctx); err != nil {
    log.Printf("Warning: Failed to warm up pool: %v", err)
}
```

## Common Patterns

### Environment-Based Configuration

```go
func createPoolConfig(env string) ygggo.PoolConfig {
    switch env {
    case "production":
        return ygggo.ProductionPoolConfig()
    case "staging":
        config := ygggo.ProductionPoolConfig()
        config.MaxOpen = 25 // Smaller than production
        return config
    case "testing":
        return ygggo.TestingPoolConfig()
    default:
        return ygggo.DevelopmentPoolConfig()
    }
}
```

### Pool Metrics Collection

```go
type PoolMetrics struct {
    Timestamp           time.Time
    OpenConnections     int
    InUse              int
    Idle               int
    ConnectionUtilization float64
    AverageWaitTime    time.Duration
    FailedConnections  int64
}

func collectMetrics(manager *ygggo.PoolManager) PoolMetrics {
    stats := manager.Stats()
    return PoolMetrics{
        Timestamp:           time.Now(),
        OpenConnections:     stats.OpenConnections,
        InUse:              stats.InUse,
        Idle:               stats.Idle,
        ConnectionUtilization: stats.ConnectionUtilization,
        AverageWaitTime:    stats.AverageWaitTime,
        FailedConnections:  stats.FailedConnections,
    }
}
```

### Health Check Integration

```go
// HTTP health endpoint
func healthHandler(manager *ygggo.PoolManager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()
        
        health, err := manager.HealthCheck(ctx)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        if !health.Healthy {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        
        json.NewEncoder(w).Encode(health)
    }
}
```

## API Reference

### Pool Configuration Methods

- `DefaultPoolConfig()` - Default pool configuration
- `DevelopmentPoolConfig()` - Development environment preset
- `ProductionPoolConfig()` - Production environment preset
- `TestingPoolConfig()` - Testing environment preset
- `HighPerformancePoolConfig()` - High performance preset
- `ValidatePoolConfig(config)` - Validate pool configuration

### Pool Manager Methods

- `NewPoolManager(pool)` - Create new pool manager
- `GetConfig()` - Get current pool configuration
- `UpdateConfig(config)` - Update pool configuration
- `Stats()` - Get detailed pool statistics
- `HealthCheck(ctx)` - Perform comprehensive health check
- `IsHealthy()` - Quick health status check
- `GetConnectionUtilization()` - Get connection utilization percentage

### Dynamic Management Methods

- `ScaleUp(connections)` - Increase maximum connections
- `ScaleDown(connections)` - Decrease maximum connections
- `Resize(maxOpen, maxIdle)` - Set specific connection limits
- `WarmUp(ctx)` - Pre-create connections
- `DrainConnections(ctx)` - Gracefully close idle connections

## Troubleshooting

### High Connection Utilization

```go
stats := manager.Stats()
if stats.ConnectionUtilization > 90 {
    // Consider scaling up
    manager.ScaleUp(10)
    
    // Or investigate slow queries
    if stats.AverageWaitTime > 100*time.Millisecond {
        log.Println("High wait times detected - check for slow queries")
    }
}
```

### Connection Leaks

```go
stats := manager.Stats()
if stats.LeakedConnections > 0 {
    log.Printf("Connection leaks detected: %d", stats.LeakedConnections)
    
    // Force close idle connections
    manager.DrainConnections(context.Background())
}
```

### Failed Connections

```go
stats := manager.Stats()
if stats.FailedConnections > 0 {
    log.Printf("Failed connections: %d", stats.FailedConnections)
    
    // Check pool health
    health, _ := manager.HealthCheck(context.Background())
    for _, issue := range health.Issues {
        log.Printf("Health issue: %s", issue)
    }
}
```
