package main

import (
	"fmt"
	"time"

	ygggo "github.com/yggai/ygggo_mysql"
)

func main() {
	fmt.Println("=== Connection Probe & Auto-Reconnect Examples ===\n")

	// Example 1: Basic Probe Configuration
	fmt.Println("1. Basic Probe Configuration:")
	basicProbeConfigExample()

	// Example 2: Probe Configuration Validation
	fmt.Println("\n2. Probe Configuration Validation:")
	probeConfigValidationExample()

	// Example 3: Exponential Backoff Algorithm
	fmt.Println("\n3. Exponential Backoff Algorithm:")
	exponentialBackoffExample()

	// Example 4: Probe Event Handling
	fmt.Println("\n4. Probe Event Handling:")
	probeEventHandlingExample()

	// Example 5: Probe Metrics and Monitoring
	fmt.Println("\n5. Probe Metrics and Monitoring:")
	probeMetricsExample()

	// Example 6: Environment-based Probe Configuration
	fmt.Println("\n6. Environment-based Probe Configuration:")
	environmentProbeConfigExample()

	fmt.Println("\n=== Connection Probe Examples Complete ===")
}

func basicProbeConfigExample() {
	// Default probe configuration
	defaultConfig := ygggo.DefaultProbeConfig()
	fmt.Printf("Default Probe Config:\n")
	fmt.Printf("  Interval: %v\n", defaultConfig.Interval)
	fmt.Printf("  Timeout: %v\n", defaultConfig.Timeout)
	fmt.Printf("  Failure Threshold: %d\n", defaultConfig.FailureThreshold)
	fmt.Printf("  Success Threshold: %d\n", defaultConfig.SuccessThreshold)
	fmt.Printf("  Auto-Reconnect: %v\n", defaultConfig.EnableAutoReconnect)

	// Reconnect policy
	fmt.Printf("  Reconnect Policy:\n")
	fmt.Printf("    Max Attempts: %d\n", defaultConfig.ReconnectPolicy.MaxAttempts)
	fmt.Printf("    Initial Backoff: %v\n", defaultConfig.ReconnectPolicy.InitialBackoff)
	fmt.Printf("    Max Backoff: %v\n", defaultConfig.ReconnectPolicy.MaxBackoff)
	fmt.Printf("    Backoff Multiplier: %.1f\n", defaultConfig.ReconnectPolicy.BackoffMultiplier)
	fmt.Printf("    Jitter: %v\n", defaultConfig.ReconnectPolicy.Jitter)
	fmt.Printf("    Max Elapsed: %v\n", defaultConfig.ReconnectPolicy.MaxElapsed)

	// Validate configuration
	if err := ygggo.ValidateProbeConfig(defaultConfig); err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Configuration is valid\n")
	}
}

func probeConfigValidationExample() {
	configs := []struct {
		name   string
		config ygggo.ProbeConfig
	}{
		{
			name: "Valid Configuration",
			config: ygggo.ProbeConfig{
				Interval:         10 * time.Second,
				Timeout:          5 * time.Second,
				FailureThreshold: 3,
				SuccessThreshold: 2,
				EnableAutoReconnect: true,
				ReconnectPolicy: ygggo.ReconnectPolicy{
					MaxAttempts:       5,
					InitialBackoff:    time.Second,
					MaxBackoff:        30 * time.Second,
					BackoffMultiplier: 2.0,
					Jitter:            true,
				},
			},
		},
		{
			name: "Invalid - Zero Interval",
			config: ygggo.ProbeConfig{
				Interval: 0,
				Timeout:  5 * time.Second,
			},
		},
		{
			name: "Invalid - Timeout > Interval",
			config: ygggo.ProbeConfig{
				Interval: 5 * time.Second,
				Timeout:  10 * time.Second,
			},
		},
	}

	for _, cfg := range configs {
		fmt.Printf("%s:\n", cfg.name)
		if err := ygggo.ValidateProbeConfig(cfg.config); err != nil {
			fmt.Printf("  ❌ %v\n", err)
		} else {
			fmt.Printf("  ✅ Valid\n")
		}
	}
}

func exponentialBackoffExample() {
	policy := ygggo.ReconnectPolicy{
		MaxAttempts:       6,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	fmt.Println("Exponential Backoff Sequence (without jitter):")
	fmt.Println("Attempt | Backoff Duration")
	fmt.Println("--------|------------------")

	for i := 0; i < 6; i++ {
		// This is a simulated call - in real usage, calculateBackoff is internal
		// We'll show the expected values based on the algorithm
		expectedBackoff := time.Duration(float64(policy.InitialBackoff) *
			pow(policy.BackoffMultiplier, float64(i)))
		if expectedBackoff > policy.MaxBackoff {
			expectedBackoff = policy.MaxBackoff
		}

		fmt.Printf("   %d    | %v\n", i+1, expectedBackoff)
	}

	fmt.Println("\nWith Jitter enabled, each attempt would have random variation")
}

func probeEventHandlingExample() {
	fmt.Println("Probe Event Types and Handling:")
	
	eventTypes := []ygggo.ProbeEventType{
		ygggo.ProbeEventHealthy,
		ygggo.ProbeEventUnhealthy,
		ygggo.ProbeEventReconnectStarted,
		ygggo.ProbeEventReconnectSuccess,
		ygggo.ProbeEventReconnectFailed,
		ygggo.ProbeEventReconnectAbandoned,
	}

	for _, eventType := range eventTypes {
		fmt.Printf("  %s: %s\n", eventType.String(), getEventDescription(eventType))
	}

	fmt.Println("\nExample Event Handler Implementation:")
	fmt.Println("```go")
	fmt.Println("type MyEventHandler struct {}")
	fmt.Println("")
	fmt.Println("func (h *MyEventHandler) HandleProbeEvent(event ygggo.ProbeEvent) {")
	fmt.Println("    switch event.Type {")
	fmt.Println("    case ygggo.ProbeEventUnhealthy:")
	fmt.Println("        log.Printf(\"Connection unhealthy: %s\", event.Message)")
	fmt.Println("        // Send alert, update metrics, etc.")
	fmt.Println("    case ygggo.ProbeEventReconnectSuccess:")
	fmt.Println("        log.Printf(\"Reconnection successful: %s\", event.Message)")
	fmt.Println("        // Update status, clear alerts, etc.")
	fmt.Println("    }")
	fmt.Println("}")
	fmt.Println("```")
}

func probeMetricsExample() {
	fmt.Println("Probe Metrics Example (simulated):")
	
	// Simulate metrics
	metrics := struct {
		TotalProbes          int64
		TotalFailures        int64
		ConsecutiveFailures  int
		ConsecutiveSuccesses int
		SuccessRate          float64
		Uptime               time.Duration
		Downtime             time.Duration
		IsReconnecting       bool
		ReconnectAttempts    int
	}{
		TotalProbes:          150,
		TotalFailures:        8,
		ConsecutiveFailures:  0,
		ConsecutiveSuccesses: 12,
		SuccessRate:          94.7,
		Uptime:               2*time.Hour + 15*time.Minute,
		Downtime:             3*time.Minute + 20*time.Second,
		IsReconnecting:       false,
		ReconnectAttempts:    2,
	}

	fmt.Printf("Connection Health Metrics:\n")
	fmt.Printf("  Total Probes: %d\n", metrics.TotalProbes)
	fmt.Printf("  Total Failures: %d\n", metrics.TotalFailures)
	fmt.Printf("  Success Rate: %.1f%%\n", metrics.SuccessRate)
	fmt.Printf("  Consecutive Successes: %d\n", metrics.ConsecutiveSuccesses)
	fmt.Printf("  Consecutive Failures: %d\n", metrics.ConsecutiveFailures)
	fmt.Printf("  Uptime: %v\n", metrics.Uptime)
	fmt.Printf("  Downtime: %v\n", metrics.Downtime)
	fmt.Printf("  Is Reconnecting: %v\n", metrics.IsReconnecting)
	fmt.Printf("  Reconnect Attempts: %d\n", metrics.ReconnectAttempts)

	// Health assessment
	if metrics.SuccessRate >= 95.0 && metrics.ConsecutiveFailures == 0 {
		fmt.Printf("  Status: ✅ Excellent\n")
	} else if metrics.SuccessRate >= 90.0 && metrics.ConsecutiveFailures < 3 {
		fmt.Printf("  Status: ✅ Good\n")
	} else if metrics.SuccessRate >= 80.0 {
		fmt.Printf("  Status: ⚠️  Fair\n")
	} else {
		fmt.Printf("  Status: ❌ Poor\n")
	}
}

func environmentProbeConfigExample() {
	environments := map[string]ygggo.ProbeConfig{
		"development": {
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
				Jitter:            false,
			},
		},
		"production": {
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
				Jitter:            true,
				MaxElapsed:        10 * time.Minute,
			},
		},
		"testing": {
			Interval:         1 * time.Second,
			Timeout:          500 * time.Millisecond,
			FailureThreshold: 1,
			SuccessThreshold: 1,
			EnableAutoReconnect: false,
		},
	}

	for env, config := range environments {
		fmt.Printf("%s Environment:\n", env)
		fmt.Printf("  Interval: %v\n", config.Interval)
		fmt.Printf("  Timeout: %v\n", config.Timeout)
		fmt.Printf("  Failure Threshold: %d\n", config.FailureThreshold)
		fmt.Printf("  Auto-Reconnect: %v\n", config.EnableAutoReconnect)
		if config.EnableAutoReconnect {
			fmt.Printf("  Max Reconnect Attempts: %d\n", config.ReconnectPolicy.MaxAttempts)
			fmt.Printf("  Max Backoff: %v\n", config.ReconnectPolicy.MaxBackoff)
		}
		fmt.Println()
	}
}

func getEventDescription(eventType ygggo.ProbeEventType) string {
	switch eventType {
	case ygggo.ProbeEventHealthy:
		return "Connection is healthy and responding"
	case ygggo.ProbeEventUnhealthy:
		return "Connection has failed health checks"
	case ygggo.ProbeEventReconnectStarted:
		return "Auto-reconnection process has started"
	case ygggo.ProbeEventReconnectSuccess:
		return "Auto-reconnection was successful"
	case ygggo.ProbeEventReconnectFailed:
		return "Auto-reconnection attempt failed"
	case ygggo.ProbeEventReconnectAbandoned:
		return "Auto-reconnection was abandoned after max attempts"
	default:
		return "Unknown event type"
	}
}

// Helper function for power calculation
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// Example of using connection probe with real database (commented out for demo)
func realDatabaseProbeExample() {
	/*
	// This example shows how to use connection probe with a real database
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

	// Create probe with production settings
	probeConfig := ygggo.ProbeConfig{
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		EnableAutoReconnect: true,
		ReconnectPolicy: ygggo.ReconnectPolicy{
			MaxAttempts:       5,
			InitialBackoff:    time.Second,
			MaxBackoff:        30 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            true,
		},
	}

	probe := ygggo.NewConnectionProbe(pool, probeConfig)

	// Add event handler
	eventHandler := &MyEventHandler{}
	probe.AddEventHandler(eventHandler)

	// Start probing
	err = probe.Start()
	if err != nil {
		log.Fatalf("Failed to start probe: %v", err)
	}
	defer probe.Stop()

	// Monitor for a while
	time.Sleep(5 * time.Minute)

	// Get final metrics
	metrics := probe.GetMetrics()
	log.Printf("Final metrics: %+v", metrics)
	*/
}
