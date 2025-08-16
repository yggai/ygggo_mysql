package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	ygggo "github.com/yggai/ygggo_mysql"
)

func main() {
	// This example demonstrates the Health Check functionality
	// Note: This requires a running MySQL instance
	
	config := ygggo.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
		Pool: ygggo.PoolConfig{
			MaxOpen: 10,
		},
	}

	ctx := context.Background()
	
	pool, err := ygggo.NewPool(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	fmt.Println("=== Health Check Examples ===\n")

	// Example 1: Basic Health Check
	fmt.Println("1. Basic Health Check:")
	basicHealthCheck(ctx, pool)

	// Example 2: Health Check with Custom Configuration
	fmt.Println("\n2. Health Check with Custom Configuration:")
	customHealthCheck(ctx, pool)

	// Example 3: Deep Health Check
	fmt.Println("\n3. Deep Health Check:")
	deepHealthCheck(ctx, pool)

	// Example 4: Health Check with Retry
	fmt.Println("\n4. Health Check with Retry:")
	healthCheckWithRetry(ctx, pool)

	// Example 5: Continuous Health Monitoring
	fmt.Println("\n5. Continuous Health Monitoring:")
	continuousMonitoring(pool)

	// Example 6: Ping with Retry
	fmt.Println("\n6. Ping with Retry:")
	pingWithRetry(ctx, pool)

	fmt.Println("\n=== Health Check Examples Complete ===")
}

func basicHealthCheck(ctx context.Context, pool *ygggo.Pool) {
	status, err := pool.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}

	fmt.Printf("Health Status: %s\n", healthStatusString(status.Healthy))
	fmt.Printf("Response Time: %v\n", status.ResponseTime)
	fmt.Printf("Connections - Active: %d, Idle: %d, Max: %d\n", 
		status.ConnectionsActive, status.ConnectionsIdle, status.ConnectionsMax)
	
	if len(status.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(status.Errors))
		for _, err := range status.Errors {
			fmt.Printf("  - %s: %s (Recoverable: %v)\n", 
				err.Type, err.Message, err.Recoverable)
		}
	}
}

func customHealthCheck(ctx context.Context, pool *ygggo.Pool) {
	config := ygggo.DefaultHealthCheckConfig()
	config.TestQuery = "SELECT 'custom_health_check' as status, NOW() as timestamp"
	config.Timeout = 3 * time.Second
	config.QueryTimeout = 2 * time.Second

	status, err := pool.HealthCheckWithConfig(ctx, config)
	if err != nil {
		log.Printf("Custom health check failed: %v", err)
		return
	}

	fmt.Printf("Health Status: %s\n", healthStatusString(status.Healthy))
	fmt.Printf("Response Time: %v\n", status.ResponseTime)
	
	// Display custom query result
	if result, ok := status.Details["test_query_result"]; ok {
		fmt.Printf("Custom Query Result: %v\n", result)
	}
	
	// Display query timing
	if queryTime, ok := status.Details["query_time"]; ok {
		fmt.Printf("Query Execution Time: %v\n", queryTime)
	}
}

func deepHealthCheck(ctx context.Context, pool *ygggo.Pool) {
	status, err := pool.DeepHealthCheck(ctx)
	if err != nil {
		log.Printf("Deep health check failed: %v", err)
		return
	}

	fmt.Printf("Health Status: %s\n", healthStatusString(status.Healthy))
	fmt.Printf("Response Time: %v\n", status.ResponseTime)
	
	// Display detailed pool statistics
	if poolStats, ok := status.Details["pool_stats"].(map[string]interface{}); ok {
		fmt.Println("Pool Statistics:")
		fmt.Printf("  Open Connections: %v\n", poolStats["open_connections"])
		fmt.Printf("  In Use: %v\n", poolStats["in_use"])
		fmt.Printf("  Idle: %v\n", poolStats["idle"])
		fmt.Printf("  Wait Count: %v\n", poolStats["wait_count"])
		fmt.Printf("  Wait Duration: %v\n", poolStats["wait_duration"])
	}

	// Check for warnings
	if warning, ok := status.Details["connection_leak_warning"]; ok && warning.(bool) {
		fmt.Println("⚠️  Warning: Potential connection leak detected")
	}
	
	if warning, ok := status.Details["high_wait_time_warning"]; ok && warning.(bool) {
		fmt.Println("⚠️  Warning: High connection wait times detected")
	}
}

func healthCheckWithRetry(ctx context.Context, pool *ygggo.Pool) {
	status, err := pool.HealthCheckWithRetry(ctx)
	if err != nil {
		log.Printf("Health check with retry failed: %v", err)
		return
	}

	fmt.Printf("Health Status: %s\n", healthStatusString(status.Healthy))
	fmt.Printf("Response Time: %v\n", status.ResponseTime)
	fmt.Println("✅ Health check completed successfully with retry capability")
}

func continuousMonitoring(pool *ygggo.Pool) {
	// Start continuous health monitoring
	err := pool.StartHealthMonitoring(2 * time.Second)
	if err != nil {
		log.Printf("Failed to start health monitoring: %v", err)
		return
	}

	fmt.Println("Started continuous health monitoring...")
	fmt.Printf("Monitoring Status: %s\n", 
		runningStatusString(pool.IsHealthMonitoringRunning()))

	// Wait for a few monitoring cycles
	time.Sleep(5 * time.Second)

	// Get cached health status
	status := pool.GetHealthStatus()
	if status != nil {
		fmt.Printf("Cached Health Status: %s\n", healthStatusString(status.Healthy))
		fmt.Printf("Last Checked: %v\n", status.LastChecked.Format(time.RFC3339))
		fmt.Printf("Response Time: %v\n", status.ResponseTime)
	}

	// Stop monitoring
	err = pool.StopHealthMonitoring()
	if err != nil {
		log.Printf("Failed to stop health monitoring: %v", err)
		return
	}

	fmt.Printf("Monitoring Status: %s\n", 
		runningStatusString(pool.IsHealthMonitoringRunning()))
	fmt.Println("✅ Continuous monitoring example completed")
}

func pingWithRetry(ctx context.Context, pool *ygggo.Pool) {
	err := pool.PingWithRetry(ctx)
	if err != nil {
		log.Printf("Ping with retry failed: %v", err)
		return
	}

	fmt.Println("✅ Ping with retry completed successfully")
}

func healthStatusString(healthy bool) string {
	if healthy {
		return "🟢 HEALTHY"
	}
	return "🔴 UNHEALTHY"
}

func runningStatusString(running bool) string {
	if running {
		return "🟢 RUNNING"
	}
	return "🔴 STOPPED"
}

// Example of health status JSON serialization
func demonstrateJSONSerialization(status *ygggo.HealthStatus) {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal health status: %v", err)
		return
	}

	fmt.Println("Health Status JSON:")
	fmt.Println(string(jsonData))
}

// Example of custom health monitoring with configuration
func advancedMonitoringExample(pool *ygggo.Pool) {
	config := ygggo.DefaultHealthCheckConfig()
	config.MonitoringInterval = 1 * time.Second
	config.Timeout = 5 * time.Second
	config.RetryAttempts = 2
	config.RetryBackoff = 500 * time.Millisecond
	config.TestQuery = "SELECT CONNECTION_ID() as conn_id, NOW() as timestamp"

	err := pool.StartHealthMonitoringWithConfig(config)
	if err != nil {
		log.Printf("Failed to start advanced monitoring: %v", err)
		return
	}
	defer pool.StopHealthMonitoring()

	fmt.Println("Advanced monitoring started with custom configuration...")
	
	// Monitor for a while
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		status := pool.GetHealthStatus()
		if status != nil {
			fmt.Printf("Cycle %d: %s (Response: %v)\n", 
				i+1, healthStatusString(status.Healthy), status.ResponseTime)
		}
	}

	fmt.Println("✅ Advanced monitoring example completed")
}
