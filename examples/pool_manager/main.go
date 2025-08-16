package main

import (
	"fmt"
	"time"

	ygggo "github.com/yggai/ygggo_mysql"
)

func main() {
	fmt.Println("=== Connection Pool Manager Examples ===\n")

	// Example 1: Basic Pool Configuration
	fmt.Println("1. Basic Pool Configuration:")
	basicPoolConfigExample()

	// Example 2: Pool Presets
	fmt.Println("\n2. Pool Configuration Presets:")
	poolPresetsExample()

	// Example 3: Pool Statistics and Monitoring
	fmt.Println("\n3. Pool Statistics and Monitoring:")
	poolStatisticsExample()

	// Example 4: Dynamic Pool Scaling
	fmt.Println("\n4. Dynamic Pool Scaling:")
	dynamicScalingExample()

	// Example 5: Pool Health Monitoring
	fmt.Println("\n5. Pool Health Monitoring:")
	healthMonitoringExample()

	// Example 6: Connection Lifecycle Management
	fmt.Println("\n6. Connection Lifecycle Management:")
	connectionLifecycleExample()

	fmt.Println("\n=== Pool Manager Examples Complete ===")
}

func basicPoolConfigExample() {
	// Create a basic pool configuration
	config := ygggo.DefaultPoolConfig()
	fmt.Printf("Default Pool Config:\n")
	fmt.Printf("  MaxOpen: %d\n", config.MaxOpen)
	fmt.Printf("  MaxIdle: %d\n", config.MaxIdle)
	fmt.Printf("  ConnMaxLifetime: %v\n", config.ConnMaxLifetime)
	fmt.Printf("  ConnMaxIdleTime: %v\n", config.ConnMaxIdleTime)

	// Validate the configuration
	if err := ygggo.ValidatePoolConfig(config); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Configuration is valid\n")
	}
}

func poolPresetsExample() {
	presets := map[string]ygggo.PoolConfig{
		"Development":     ygggo.DevelopmentPoolConfig(),
		"Production":      ygggo.ProductionPoolConfig(),
		"Testing":         ygggo.TestingPoolConfig(),
		"High Performance": ygggo.HighPerformancePoolConfig(),
	}

	for name, config := range presets {
		fmt.Printf("%s Preset:\n", name)
		fmt.Printf("  MaxOpen: %d, MaxIdle: %d\n", config.MaxOpen, config.MaxIdle)
		fmt.Printf("  ConnMaxLifetime: %v\n", config.ConnMaxLifetime)
		
		if err := ygggo.ValidatePoolConfig(config); err != nil {
			fmt.Printf("  ❌ Invalid: %v\n", err)
		} else {
			fmt.Printf("  ✅ Valid\n")
		}
		fmt.Println()
	}
}

func poolStatisticsExample() {
	// This would typically use a real database connection
	fmt.Println("Pool Statistics Example (simulated):")
	fmt.Println("  OpenConnections: 5")
	fmt.Println("  InUse: 2")
	fmt.Println("  Idle: 3")
	fmt.Println("  WaitCount: 10")
	fmt.Println("  WaitDuration: 150ms")
	fmt.Println("  ConnectionUtilization: 40.0%")
	fmt.Println("  TotalConnections: 25")
	fmt.Println("  FailedConnections: 1")
	fmt.Println("  LeakedConnections: 0")
}

func dynamicScalingExample() {
	fmt.Println("Dynamic Pool Scaling Example:")
	
	// Start with a base configuration
	initialMaxOpen := 10
	fmt.Printf("Initial MaxOpen: %d\n", initialMaxOpen)
	
	// Simulate scaling up
	scaleUpBy := 5
	newMaxOpen := initialMaxOpen + scaleUpBy
	fmt.Printf("After scaling up by %d: MaxOpen = %d\n", scaleUpBy, newMaxOpen)
	
	// Simulate scaling down
	scaleDownBy := 3
	finalMaxOpen := newMaxOpen - scaleDownBy
	fmt.Printf("After scaling down by %d: MaxOpen = %d\n", scaleDownBy, finalMaxOpen)
	
	// Simulate resize operation
	resizeMaxOpen, resizeMaxIdle := 20, 8
	fmt.Printf("After resize: MaxOpen = %d, MaxIdle = %d\n", resizeMaxOpen, resizeMaxIdle)
}

func healthMonitoringExample() {
	fmt.Println("Pool Health Monitoring Example:")
	
	// Simulate health check results
	fmt.Println("Health Check Results:")
	fmt.Println("  Status: ✅ Healthy")
	fmt.Println("  LastChecked: 2024-01-15 10:30:00")
	fmt.Println("  ResponseTime: 25ms")
	fmt.Println("  Issues: None")
	fmt.Println("  Details:")
	fmt.Println("    - Connection utilization: 45%")
	fmt.Println("    - Average wait time: 12ms")
	fmt.Println("    - Failed connections: 0")
}

func connectionLifecycleExample() {
	fmt.Println("Connection Lifecycle Management Example:")
	
	// Simulate connection lifecycle operations
	fmt.Println("Operations:")
	fmt.Println("  1. WarmUp: Pre-creating 5 connections... ✅")
	fmt.Println("  2. Connection validation: All connections healthy ✅")
	fmt.Println("  3. Leak detection: No leaks detected ✅")
	fmt.Println("  4. DrainConnections: Gracefully closing idle connections... ✅")
	fmt.Println("  5. Connection cleanup: Expired connections removed ✅")
}

// Example of using pool manager with real database (commented out for demo)
func realDatabaseExample() {
	// This example shows how to use the pool manager with a real database
	/*
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
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create pool manager
	manager := ygggo.NewPoolManager(pool)

	// Get initial statistics
	stats := manager.Stats()
	fmt.Printf("Initial pool stats: %+v\n", stats)

	// Perform health check
	health, err := manager.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("Pool health: %+v\n", health)
	}

	// Scale up the pool
	err = manager.ScaleUp(10)
	if err != nil {
		log.Printf("Scale up failed: %v", err)
	} else {
		fmt.Println("Pool scaled up successfully")
	}

	// Warm up connections
	err = manager.WarmUp(ctx)
	if err != nil {
		log.Printf("Warm up failed: %v", err)
	} else {
		fmt.Println("Pool warmed up successfully")
	}

	// Monitor pool utilization
	utilization := manager.GetConnectionUtilization()
	fmt.Printf("Connection utilization: %.2f%%\n", utilization)

	// Check if pool is healthy
	if manager.IsHealthy() {
		fmt.Println("Pool is healthy")
	} else {
		fmt.Println("Pool has health issues")
	}
	*/
}

// Example of environment-based pool configuration
func environmentBasedConfig() {
	env := "production" // This would come from environment variable

	var poolConfig ygggo.PoolConfig

	switch env {
	case "development":
		poolConfig = ygggo.DevelopmentPoolConfig()
		fmt.Println("Using development pool configuration")
	case "production":
		poolConfig = ygggo.ProductionPoolConfig()
		fmt.Println("Using production pool configuration")
	case "testing":
		poolConfig = ygggo.TestingPoolConfig()
		fmt.Println("Using testing pool configuration")
	default:
		poolConfig = ygggo.DefaultPoolConfig()
		fmt.Println("Using default pool configuration")
	}

	fmt.Printf("Selected config: MaxOpen=%d, MaxIdle=%d\n", 
		poolConfig.MaxOpen, poolConfig.MaxIdle)
}

// Example of pool configuration validation
func configValidationExample() {
	fmt.Println("\nPool Configuration Validation Examples:")

	// Valid configuration
	validConfig := ygggo.PoolConfig{
		MaxOpen:         25,
		MaxIdle:         10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	if err := ygggo.ValidatePoolConfig(validConfig); err != nil {
		fmt.Printf("❌ Valid config failed validation: %v\n", err)
	} else {
		fmt.Printf("✅ Valid configuration passed validation\n")
	}

	// Invalid configuration - MaxIdle > MaxOpen
	invalidConfig := ygggo.PoolConfig{
		MaxOpen:         5,
		MaxIdle:         10, // Invalid: greater than MaxOpen
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	if err := ygggo.ValidatePoolConfig(invalidConfig); err != nil {
		fmt.Printf("❌ Invalid config correctly rejected: %v\n", err)
	} else {
		fmt.Printf("⚠️  Invalid config incorrectly accepted\n")
	}
}

// Example of monitoring pool metrics over time
func poolMetricsMonitoring() {
	fmt.Println("\nPool Metrics Monitoring Example:")

	// Simulate metrics collection over time
	metrics := []struct {
		timestamp   string
		utilization float64
		waitTime    time.Duration
		errors      int
	}{
		{"10:00:00", 25.5, 10 * time.Millisecond, 0},
		{"10:05:00", 45.2, 15 * time.Millisecond, 0},
		{"10:10:00", 78.9, 35 * time.Millisecond, 1},
		{"10:15:00", 92.1, 85 * time.Millisecond, 2},
		{"10:20:00", 65.3, 25 * time.Millisecond, 0},
	}

	fmt.Println("Time     | Utilization | Avg Wait | Errors")
	fmt.Println("---------|-------------|----------|-------")
	for _, metric := range metrics {
		status := "✅"
		if metric.utilization > 90 || metric.waitTime > 50*time.Millisecond || metric.errors > 0 {
			status = "⚠️"
		}
		fmt.Printf("%s | %6.1f%%    | %8v | %d %s\n", 
			metric.timestamp, metric.utilization, metric.waitTime, metric.errors, status)
	}
}
