package ygggo_mysql

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ProbeConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ProbeConfig{
				Interval:         5 * time.Second,
				Timeout:          3 * time.Second,
				FailureThreshold: 3,
				SuccessThreshold: 2,
				EnableAutoReconnect: true,
				ReconnectPolicy: ReconnectPolicy{
					MaxAttempts:       5,
					InitialBackoff:    time.Second,
					MaxBackoff:        30 * time.Second,
					BackoffMultiplier: 2.0,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid interval",
			config: ProbeConfig{
				Interval: 0,
				Timeout:  3 * time.Second,
			},
			wantErr: true,
			errMsg:  "interval must be positive",
		},
		{
			name: "timeout greater than interval",
			config: ProbeConfig{
				Interval: 2 * time.Second,
				Timeout:  5 * time.Second,
			},
			wantErr: true,
			errMsg:  "timeout cannot be greater than interval",
		},
		{
			name: "invalid failure threshold",
			config: ProbeConfig{
				Interval:         5 * time.Second,
				Timeout:          3 * time.Second,
				FailureThreshold: 0,
			},
			wantErr: true,
			errMsg:  "failure threshold must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProbeConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProbeConfig_Defaults(t *testing.T) {
	config := DefaultProbeConfig()

	assert.Greater(t, config.Interval, time.Duration(0))
	assert.Greater(t, config.Timeout, time.Duration(0))
	assert.Less(t, config.Timeout, config.Interval)
	assert.Greater(t, config.FailureThreshold, 0)
	assert.Greater(t, config.SuccessThreshold, 0)
	assert.NotNil(t, config.ReconnectPolicy)
}

func TestConnectionProbe_BasicProbing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Create probe with short intervals for testing
	probeConfig := ProbeConfig{
		Interval:         100 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		EnableAutoReconnect: false,
	}

	probe := NewConnectionProbe(pool, probeConfig)
	require.NotNil(t, probe)

	// Test initial state
	state := probe.GetState()
	assert.Equal(t, ProbeStatusHealthy, state.Status)
	assert.Equal(t, int64(0), state.TotalProbes)
	assert.Equal(t, 0, state.ConsecutiveFailures)

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	assert.True(t, probe.IsRunning())

	// Wait for a few probe cycles
	time.Sleep(300 * time.Millisecond)

	// Check that probing occurred
	state = probe.GetState()
	assert.Greater(t, state.TotalProbes, int64(0))
	assert.False(t, state.LastProbeTime.IsZero())

	// Stop probing
	err = probe.Stop()
	require.NoError(t, err)
	assert.False(t, probe.IsRunning())
}

func TestConnectionProbe_ProbeFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)

	// Create probe
	probeConfig := ProbeConfig{
		Interval:         100 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		EnableAutoReconnect: false,
	}

	probe := NewConnectionProbe(pool, probeConfig)

	// Close the pool to simulate connection failure
	pool.Close()

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	defer probe.Stop()

	// Wait for failures to accumulate
	time.Sleep(300 * time.Millisecond)

	// Check that failures were detected
	state := probe.GetState()
	assert.Greater(t, state.TotalFailures, int64(0))
	assert.Greater(t, state.ConsecutiveFailures, 0)
	assert.False(t, state.LastFailureTime.IsZero())
}

func TestConnectionProbe_EventHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Create probe
	probeConfig := ProbeConfig{
		Interval:         100 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		EnableAutoReconnect: false,
	}

	probe := NewConnectionProbe(pool, probeConfig)

	// Create event handler
	eventHandler := &TestProbeEventHandler{}
	probe.AddEventHandler(eventHandler)

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	defer probe.Stop()

	// Wait for events with multiple checks
	maxWait := 500 * time.Millisecond
	checkInterval := 50 * time.Millisecond
	waited := time.Duration(0)

	for waited < maxWait {
		time.Sleep(checkInterval)
		waited += checkInterval

		eventHandler.mutex.Lock()
		eventCount := len(eventHandler.events)
		eventHandler.mutex.Unlock()

		if eventCount > 0 {
			break
		}
	}

	// Check that events were received
	eventHandler.mutex.Lock()
	events := eventHandler.events
	eventHandler.mutex.Unlock()

	// Debug information
	state := probe.GetState()
	t.Logf("Probe state: TotalProbes=%d, Status=%s", state.TotalProbes, state.Status)
	t.Logf("Events received: %d", len(events))

	assert.Greater(t, len(events), 0, "Should have received probe events")
}

func TestAutoReconnector_ExponentialBackoff(t *testing.T) {
	policy := ReconnectPolicy{
		MaxAttempts:       5,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	reconnector := NewAutoReconnector(nil, policy)

	// Test backoff calculation
	backoffs := make([]time.Duration, 5)
	for i := 0; i < 5; i++ {
		backoffs[i] = reconnector.calculateBackoff(i)
	}

	// Verify exponential growth
	assert.Equal(t, 100*time.Millisecond, backoffs[0])
	assert.Equal(t, 200*time.Millisecond, backoffs[1])
	assert.Equal(t, 400*time.Millisecond, backoffs[2])
	assert.Equal(t, 800*time.Millisecond, backoffs[3])
	assert.Equal(t, 1600*time.Millisecond, backoffs[4])
}

func TestAutoReconnector_MaxBackoff(t *testing.T) {
	policy := ReconnectPolicy{
		MaxAttempts:       10,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        500 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	reconnector := NewAutoReconnector(nil, policy)

	// Test that backoff doesn't exceed maximum
	for i := 0; i < 10; i++ {
		backoff := reconnector.calculateBackoff(i)
		assert.LessOrEqual(t, backoff, policy.MaxBackoff,
			"Backoff should not exceed maximum at attempt %d", i)
	}
}

func TestAutoReconnector_Jitter(t *testing.T) {
	policy := ReconnectPolicy{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
	}

	reconnector := NewAutoReconnector(nil, policy)

	// Test that jitter produces different values
	backoffs := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		backoffs[i] = reconnector.calculateBackoff(1) // Same attempt number
	}

	// Check that we get some variation (not all values are the same)
	allSame := true
	for i := 1; i < len(backoffs); i++ {
		if backoffs[i] != backoffs[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "Jitter should produce different backoff values")
}

// TestProbeEventHandler is a test implementation of ProbeEventHandler
type TestProbeEventHandler struct {
	events []ProbeEvent
	mutex  sync.Mutex
}

func (h *TestProbeEventHandler) HandleProbeEvent(event ProbeEvent) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.events = append(h.events, event)
}

func (h *TestProbeEventHandler) GetEvents() []ProbeEvent {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	events := make([]ProbeEvent, len(h.events))
	copy(events, h.events)
	return events
}

func (h *TestProbeEventHandler) Clear() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.events = nil
}

func TestConnectionProbe_ForceProbe(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Create probe
	probeConfig := DefaultProbeConfig()
	probeConfig.EnableAutoReconnect = false
	probe := NewConnectionProbe(pool, probeConfig)

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	defer probe.Stop()

	// Get initial state
	initialState := probe.GetState()

	// Force a probe
	err = probe.ForceProbe(ctx)
	assert.NoError(t, err)

	// Check that probe count increased
	newState := probe.GetState()
	assert.Greater(t, newState.TotalProbes, initialState.TotalProbes)
	assert.True(t, newState.LastProbeTime.After(initialState.LastProbeTime))
}

func TestConnectionProbe_ConfigUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Create probe
	probeConfig := DefaultProbeConfig()
	probe := NewConnectionProbe(pool, probeConfig)

	// Update configuration
	newConfig := probeConfig
	newConfig.Interval = 10 * time.Second
	newConfig.FailureThreshold = 5

	err = probe.UpdateConfig(newConfig)
	assert.NoError(t, err)

	// Verify configuration was updated
	updatedConfig := probe.GetConfig()
	assert.Equal(t, 10*time.Second, updatedConfig.Interval)
	assert.Equal(t, 5, updatedConfig.FailureThreshold)

	// Test invalid configuration update
	invalidConfig := newConfig
	invalidConfig.Interval = 0
	err = probe.UpdateConfig(invalidConfig)
	assert.Error(t, err)
}

func TestConnectionProbe_Metrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Create probe
	probeConfig := ProbeConfig{
		Interval:         100 * time.Millisecond,
		Timeout:          50 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		EnableAutoReconnect: false,
	}

	probe := NewConnectionProbe(pool, probeConfig)

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	defer probe.Stop()

	// Wait for some probes
	time.Sleep(300 * time.Millisecond)

	// Get metrics
	metrics := probe.GetMetrics()
	assert.Greater(t, metrics.TotalProbes, int64(0))
	assert.GreaterOrEqual(t, metrics.SuccessRate, 0.0)
	assert.LessOrEqual(t, metrics.SuccessRate, 100.0)
	assert.False(t, metrics.LastProbeTime.IsZero())
}

func TestConnectionProbe_StatusTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)

	// Create probe with low thresholds for faster testing
	probeConfig := ProbeConfig{
		Interval:         50 * time.Millisecond,
		Timeout:          25 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		EnableAutoReconnect: false,
	}

	probe := NewConnectionProbe(pool, probeConfig)

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	defer probe.Stop()

	// Initially should be healthy
	state := probe.GetState()
	assert.Equal(t, ProbeStatusHealthy, state.Status)

	// Close pool to cause failures
	pool.Close()

	// Wait for failures to accumulate
	time.Sleep(200 * time.Millisecond)

	// Should now be unhealthy
	state = probe.GetState()
	assert.Equal(t, ProbeStatusUnhealthy, state.Status)
	assert.Greater(t, state.ConsecutiveFailures, 0)
	assert.Greater(t, state.TotalFailures, int64(0))
}

func TestConnectionProbe_EventSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)

	// Create probe
	probeConfig := ProbeConfig{
		Interval:         50 * time.Millisecond,
		Timeout:          25 * time.Millisecond,
		FailureThreshold: 2,
		SuccessThreshold: 1,
		EnableAutoReconnect: false,
	}

	probe := NewConnectionProbe(pool, probeConfig)

	// Create event handler
	eventHandler := &TestProbeEventHandler{}
	probe.AddEventHandler(eventHandler)

	// Start probing
	err = probe.Start()
	require.NoError(t, err)
	defer probe.Stop()

	// Wait for initial healthy events
	time.Sleep(100 * time.Millisecond)

	// Close pool to trigger unhealthy events
	pool.Close()

	// Wait for unhealthy events
	time.Sleep(200 * time.Millisecond)

	// Check event sequence
	events := eventHandler.GetEvents()
	assert.Greater(t, len(events), 0, "Should have received events")

	// Look for unhealthy event
	foundUnhealthy := false
	for _, event := range events {
		if event.Type == ProbeEventUnhealthy {
			foundUnhealthy = true
			assert.NotEmpty(t, event.Message)
			assert.False(t, event.Timestamp.IsZero())
			break
		}
	}
	assert.True(t, foundUnhealthy, "Should have received unhealthy event")
}
