package ygggo_mysql

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ProbeStatus represents the current status of connection probing
type ProbeStatus int

const (
	ProbeStatusHealthy ProbeStatus = iota
	ProbeStatusUnhealthy
	ProbeStatusReconnecting
	ProbeStatusFailed
)

func (s ProbeStatus) String() string {
	switch s {
	case ProbeStatusHealthy:
		return "Healthy"
	case ProbeStatusUnhealthy:
		return "Unhealthy"
	case ProbeStatusReconnecting:
		return "Reconnecting"
	case ProbeStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// ProbeConfig configures connection probing behavior
type ProbeConfig struct {
	Interval            time.Duration   `json:"interval"`
	Timeout             time.Duration   `json:"timeout"`
	FailureThreshold    int             `json:"failure_threshold"`
	SuccessThreshold    int             `json:"success_threshold"`
	EnableAutoReconnect bool            `json:"enable_auto_reconnect"`
	ReconnectPolicy     ReconnectPolicy `json:"reconnect_policy"`
}

// ReconnectPolicy defines the reconnection strategy
type ReconnectPolicy struct {
	MaxAttempts       int           `json:"max_attempts"`
	InitialBackoff    time.Duration `json:"initial_backoff"`
	MaxBackoff        time.Duration `json:"max_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	Jitter            bool          `json:"jitter"`
	MaxElapsed        time.Duration `json:"max_elapsed"`
}

// ProbeState represents the current state of connection probing
type ProbeState struct {
	Status               ProbeStatus `json:"status"`
	LastProbeTime        time.Time   `json:"last_probe_time"`
	LastSuccessTime      time.Time   `json:"last_success_time"`
	LastFailureTime      time.Time   `json:"last_failure_time"`
	ConsecutiveFailures  int         `json:"consecutive_failures"`
	ConsecutiveSuccesses int         `json:"consecutive_successes"`
	TotalProbes          int64       `json:"total_probes"`
	TotalFailures        int64       `json:"total_failures"`
	ReconnectAttempts    int         `json:"reconnect_attempts"`
	IsReconnecting       bool        `json:"is_reconnecting"`
}

// ProbeEventType represents different types of probe events
type ProbeEventType int

const (
	ProbeEventHealthy ProbeEventType = iota
	ProbeEventUnhealthy
	ProbeEventReconnectStarted
	ProbeEventReconnectSuccess
	ProbeEventReconnectFailed
	ProbeEventReconnectAbandoned
)

func (t ProbeEventType) String() string {
	switch t {
	case ProbeEventHealthy:
		return "Healthy"
	case ProbeEventUnhealthy:
		return "Unhealthy"
	case ProbeEventReconnectStarted:
		return "ReconnectStarted"
	case ProbeEventReconnectSuccess:
		return "ReconnectSuccess"
	case ProbeEventReconnectFailed:
		return "ReconnectFailed"
	case ProbeEventReconnectAbandoned:
		return "ReconnectAbandoned"
	default:
		return "Unknown"
	}
}

// ProbeEvent represents a probe event
type ProbeEvent struct {
	Type      ProbeEventType `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Message   string         `json:"message"`
	Error     error          `json:"error,omitempty"`
	State     ProbeState     `json:"state"`
}

// ProbeEventHandler handles probe events
type ProbeEventHandler interface {
	HandleProbeEvent(event ProbeEvent)
}

// ConnectionProbe manages connection health probing and auto-reconnection
type ConnectionProbe struct {
	pool          *Pool
	config        ProbeConfig
	state         ProbeState
	reconnector   *AutoReconnector
	eventHandlers []ProbeEventHandler
	stopChan      chan struct{}
	running       bool
	mutex         sync.RWMutex
}

// NewConnectionProbe creates a new connection probe
func NewConnectionProbe(pool *Pool, config ProbeConfig) *ConnectionProbe {
	probe := &ConnectionProbe{
		pool:          pool,
		config:        config,
		state:         ProbeState{Status: ProbeStatusHealthy},
		eventHandlers: make([]ProbeEventHandler, 0),
	}

	if config.EnableAutoReconnect {
		probe.reconnector = NewAutoReconnector(pool, config.ReconnectPolicy)
	}

	return probe
}

// DefaultProbeConfig returns a default probe configuration
func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
		EnableAutoReconnect: true,
		ReconnectPolicy: ReconnectPolicy{
			MaxAttempts:       5,
			InitialBackoff:    time.Second,
			MaxBackoff:        30 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            true,
			MaxElapsed:        5 * time.Minute,
		},
	}
}

// ValidateProbeConfig validates a probe configuration
func ValidateProbeConfig(config ProbeConfig) error {
	if config.Interval <= 0 {
		return fmt.Errorf("interval must be positive, got %v", config.Interval)
	}
	
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", config.Timeout)
	}
	
	if config.Timeout >= config.Interval {
		return fmt.Errorf("timeout cannot be greater than interval (timeout: %v, interval: %v)", 
			config.Timeout, config.Interval)
	}
	
	if config.FailureThreshold <= 0 {
		return fmt.Errorf("failure threshold must be positive, got %d", config.FailureThreshold)
	}
	
	if config.SuccessThreshold <= 0 {
		return fmt.Errorf("success threshold must be positive, got %d", config.SuccessThreshold)
	}
	
	if config.EnableAutoReconnect {
		if err := ValidateReconnectPolicy(config.ReconnectPolicy); err != nil {
			return fmt.Errorf("invalid reconnect policy: %w", err)
		}
	}
	
	return nil
}

// ValidateReconnectPolicy validates a reconnect policy
func ValidateReconnectPolicy(policy ReconnectPolicy) error {
	if policy.MaxAttempts <= 0 {
		return fmt.Errorf("max attempts must be positive, got %d", policy.MaxAttempts)
	}
	
	if policy.InitialBackoff <= 0 {
		return fmt.Errorf("initial backoff must be positive, got %v", policy.InitialBackoff)
	}
	
	if policy.MaxBackoff <= 0 {
		return fmt.Errorf("max backoff must be positive, got %v", policy.MaxBackoff)
	}
	
	if policy.InitialBackoff > policy.MaxBackoff {
		return fmt.Errorf("initial backoff cannot be greater than max backoff")
	}
	
	if policy.BackoffMultiplier <= 1.0 {
		return fmt.Errorf("backoff multiplier must be greater than 1.0, got %f", policy.BackoffMultiplier)
	}
	
	return nil
}

// Start begins connection probing
func (cp *ConnectionProbe) Start() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	if cp.running {
		return fmt.Errorf("connection probe is already running")
	}
	
	cp.stopChan = make(chan struct{})
	cp.running = true
	
	go cp.probeLoop()
	
	return nil
}

// Stop stops connection probing
func (cp *ConnectionProbe) Stop() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	if !cp.running {
		return fmt.Errorf("connection probe is not running")
	}
	
	close(cp.stopChan)
	cp.running = false
	
	return nil
}

// IsRunning returns whether probing is currently active
func (cp *ConnectionProbe) IsRunning() bool {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.running
}

// GetState returns the current probe state
func (cp *ConnectionProbe) GetState() ProbeState {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.state
}

// GetConfig returns the current probe configuration
func (cp *ConnectionProbe) GetConfig() ProbeConfig {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.config
}

// AddEventHandler adds a probe event handler
func (cp *ConnectionProbe) AddEventHandler(handler ProbeEventHandler) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	cp.eventHandlers = append(cp.eventHandlers, handler)
}

// RemoveEventHandler removes a probe event handler
func (cp *ConnectionProbe) RemoveEventHandler(handler ProbeEventHandler) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	for i, h := range cp.eventHandlers {
		if h == handler {
			cp.eventHandlers = append(cp.eventHandlers[:i], cp.eventHandlers[i+1:]...)
			break
		}
	}
}

// probeLoop runs the main probing loop
func (cp *ConnectionProbe) probeLoop() {
	ticker := time.NewTicker(cp.config.Interval)
	defer ticker.Stop()
	
	// Perform initial probe
	cp.performProbe()
	
	for {
		select {
		case <-cp.stopChan:
			return
		case <-ticker.C:
			cp.performProbe()
		}
	}
}

// performProbe executes a single probe
func (cp *ConnectionProbe) performProbe() {
	ctx, cancel := context.WithTimeout(context.Background(), cp.config.Timeout)
	defer cancel()
	
	cp.mutex.Lock()
	cp.state.TotalProbes++
	cp.state.LastProbeTime = time.Now()
	cp.mutex.Unlock()
	
	err := cp.pool.Ping(ctx)
	
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	if err != nil {
		cp.handleProbeFailure(err)
	} else {
		cp.handleProbeSuccess()
	}
}

// handleProbeFailure handles a probe failure
func (cp *ConnectionProbe) handleProbeFailure(err error) {
	cp.state.TotalFailures++
	cp.state.ConsecutiveFailures++
	cp.state.ConsecutiveSuccesses = 0
	cp.state.LastFailureTime = time.Now()
	
	// Check if we've reached the failure threshold
	if cp.state.ConsecutiveFailures >= cp.config.FailureThreshold {
		if cp.state.Status == ProbeStatusHealthy {
			cp.state.Status = ProbeStatusUnhealthy
			cp.emitEvent(ProbeEventUnhealthy, fmt.Sprintf("Connection unhealthy after %d consecutive failures", cp.state.ConsecutiveFailures), err)
			
			// Start auto-reconnection if enabled
			if cp.config.EnableAutoReconnect && cp.reconnector != nil {
				go cp.startAutoReconnect()
			}
		}
	}
}

// handleProbeSuccess handles a probe success
func (cp *ConnectionProbe) handleProbeSuccess() {
	cp.state.ConsecutiveSuccesses++
	cp.state.ConsecutiveFailures = 0
	cp.state.LastSuccessTime = time.Now()

	// Emit event for first successful probe or when reaching success threshold
	if cp.state.ConsecutiveSuccesses == 1 {
		// Always emit event for first successful probe
		cp.emitEvent(ProbeEventHealthy, "Connection probe successful", nil)
	}

	// Check if we've reached the success threshold for status change
	if cp.state.ConsecutiveSuccesses >= cp.config.SuccessThreshold {
		if cp.state.Status != ProbeStatusHealthy {
			cp.state.Status = ProbeStatusHealthy
			cp.state.IsReconnecting = false
			cp.emitEvent(ProbeEventHealthy, fmt.Sprintf("Connection healthy after %d consecutive successes", cp.state.ConsecutiveSuccesses), nil)
		}
	}
}

// emitEvent emits a probe event to all handlers
func (cp *ConnectionProbe) emitEvent(eventType ProbeEventType, message string, err error) {
	event := ProbeEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Message:   message,
		Error:     err,
		State:     cp.state,
	}
	
	for _, handler := range cp.eventHandlers {
		go handler.HandleProbeEvent(event)
	}
}

// startAutoReconnect starts the auto-reconnection process
func (cp *ConnectionProbe) startAutoReconnect() {
	if cp.reconnector == nil {
		return
	}
	
	cp.mutex.Lock()
	if cp.state.IsReconnecting {
		cp.mutex.Unlock()
		return
	}
	cp.state.IsReconnecting = true
	cp.state.Status = ProbeStatusReconnecting
	cp.mutex.Unlock()
	
	cp.emitEvent(ProbeEventReconnectStarted, "Starting auto-reconnection", nil)
	
	success := cp.reconnector.Reconnect(context.Background())
	
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	if success {
		cp.state.Status = ProbeStatusHealthy
		cp.state.IsReconnecting = false
		cp.state.ConsecutiveFailures = 0
		cp.emitEvent(ProbeEventReconnectSuccess, "Auto-reconnection successful", nil)
	} else {
		cp.state.Status = ProbeStatusFailed
		cp.state.IsReconnecting = false
		cp.emitEvent(ProbeEventReconnectAbandoned, "Auto-reconnection abandoned", nil)
	}
}

// AutoReconnector handles automatic reconnection with exponential backoff
type AutoReconnector struct {
	pool   *Pool
	policy ReconnectPolicy
	state  ReconnectState
	mutex  sync.RWMutex
}

// ReconnectState represents the current state of reconnection
type ReconnectState struct {
	IsActive        bool      `json:"is_active"`
	StartTime       time.Time `json:"start_time"`
	Attempts        int       `json:"attempts"`
	LastAttemptTime time.Time `json:"last_attempt_time"`
	NextAttemptTime time.Time `json:"next_attempt_time"`
	LastError       error     `json:"last_error,omitempty"`
}

// NewAutoReconnector creates a new auto-reconnector
func NewAutoReconnector(pool *Pool, policy ReconnectPolicy) *AutoReconnector {
	return &AutoReconnector{
		pool:   pool,
		policy: policy,
	}
}

// Reconnect attempts to reconnect with exponential backoff
func (ar *AutoReconnector) Reconnect(ctx context.Context) bool {
	ar.mutex.Lock()
	if ar.state.IsActive {
		ar.mutex.Unlock()
		return false
	}

	ar.state = ReconnectState{
		IsActive:  true,
		StartTime: time.Now(),
	}
	ar.mutex.Unlock()

	defer func() {
		ar.mutex.Lock()
		ar.state.IsActive = false
		ar.mutex.Unlock()
	}()

	for attempt := 0; attempt < ar.policy.MaxAttempts; attempt++ {
		// Check if max elapsed time exceeded
		if ar.policy.MaxElapsed > 0 && time.Since(ar.state.StartTime) > ar.policy.MaxElapsed {
			break
		}

		// Calculate backoff for this attempt
		backoff := ar.calculateBackoff(attempt)

		ar.mutex.Lock()
		ar.state.Attempts = attempt + 1
		ar.state.NextAttemptTime = time.Now().Add(backoff)
		ar.mutex.Unlock()

		// Wait for backoff period (except for first attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(backoff):
				// Continue with reconnection attempt
			}
		}

		ar.mutex.Lock()
		ar.state.LastAttemptTime = time.Now()
		ar.mutex.Unlock()

		// Attempt to reconnect by testing the connection
		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := ar.pool.Ping(testCtx)
		cancel()

		ar.mutex.Lock()
		ar.state.LastError = err
		ar.mutex.Unlock()

		if err == nil {
			return true // Reconnection successful
		}
	}

	return false // All attempts failed
}

// calculateBackoff calculates the backoff duration for a given attempt
func (ar *AutoReconnector) calculateBackoff(attempt int) time.Duration {
	// Calculate exponential backoff
	backoff := float64(ar.policy.InitialBackoff) * math.Pow(ar.policy.BackoffMultiplier, float64(attempt))

	// Apply maximum backoff limit
	if backoff > float64(ar.policy.MaxBackoff) {
		backoff = float64(ar.policy.MaxBackoff)
	}

	duration := time.Duration(backoff)

	// Apply jitter if enabled
	if ar.policy.Jitter {
		// Add random jitter up to 10% of the backoff time
		jitter := time.Duration(rand.Float64() * float64(duration) * 0.1)
		duration += jitter
	}

	return duration
}

// GetState returns the current reconnection state
func (ar *AutoReconnector) GetState() ReconnectState {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()
	return ar.state
}

// IsActive returns whether reconnection is currently active
func (ar *AutoReconnector) IsActive() bool {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()
	return ar.state.IsActive
}

// ForceProbe performs an immediate probe outside of the regular schedule
func (cp *ConnectionProbe) ForceProbe(ctx context.Context) error {
	if !cp.IsRunning() {
		return fmt.Errorf("connection probe is not running")
	}

	probeCtx, cancel := context.WithTimeout(ctx, cp.config.Timeout)
	defer cancel()

	cp.mutex.Lock()
	cp.state.TotalProbes++
	cp.state.LastProbeTime = time.Now()
	cp.mutex.Unlock()

	err := cp.pool.Ping(probeCtx)

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if err != nil {
		cp.handleProbeFailure(err)
		return err
	} else {
		cp.handleProbeSuccess()
		return nil
	}
}

// ForceReconnect forces an immediate reconnection attempt
func (cp *ConnectionProbe) ForceReconnect(ctx context.Context) error {
	if cp.reconnector == nil {
		return fmt.Errorf("auto-reconnection is not enabled")
	}

	cp.mutex.Lock()
	if cp.state.IsReconnecting {
		cp.mutex.Unlock()
		return fmt.Errorf("reconnection is already in progress")
	}
	cp.state.IsReconnecting = true
	cp.state.Status = ProbeStatusReconnecting
	cp.mutex.Unlock()

	cp.emitEvent(ProbeEventReconnectStarted, "Force reconnection started", nil)

	success := cp.reconnector.Reconnect(ctx)

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if success {
		cp.state.Status = ProbeStatusHealthy
		cp.state.IsReconnecting = false
		cp.state.ConsecutiveFailures = 0
		cp.emitEvent(ProbeEventReconnectSuccess, "Force reconnection successful", nil)
		return nil
	} else {
		cp.state.Status = ProbeStatusFailed
		cp.state.IsReconnecting = false
		cp.emitEvent(ProbeEventReconnectFailed, "Force reconnection failed", nil)
		return fmt.Errorf("force reconnection failed")
	}
}

// UpdateConfig updates the probe configuration
func (cp *ConnectionProbe) UpdateConfig(config ProbeConfig) error {
	if err := ValidateProbeConfig(config); err != nil {
		return fmt.Errorf("invalid probe configuration: %w", err)
	}

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	cp.config = config

	// Update reconnector if auto-reconnect settings changed
	if config.EnableAutoReconnect {
		if cp.reconnector == nil {
			cp.reconnector = NewAutoReconnector(cp.pool, config.ReconnectPolicy)
		} else {
			cp.reconnector.policy = config.ReconnectPolicy
		}
	} else {
		cp.reconnector = nil
	}

	return nil
}

// GetMetrics returns probe metrics for monitoring
func (cp *ConnectionProbe) GetMetrics() ProbeMetrics {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	var uptime time.Duration
	if !cp.state.LastSuccessTime.IsZero() {
		uptime = time.Since(cp.state.LastSuccessTime)
	}

	var downtime time.Duration
	if !cp.state.LastFailureTime.IsZero() && cp.state.Status != ProbeStatusHealthy {
		downtime = time.Since(cp.state.LastFailureTime)
	}

	var successRate float64
	if cp.state.TotalProbes > 0 {
		successRate = float64(cp.state.TotalProbes-cp.state.TotalFailures) / float64(cp.state.TotalProbes) * 100
	}

	return ProbeMetrics{
		TotalProbes:          cp.state.TotalProbes,
		TotalFailures:        cp.state.TotalFailures,
		ConsecutiveFailures:  cp.state.ConsecutiveFailures,
		ConsecutiveSuccesses: cp.state.ConsecutiveSuccesses,
		SuccessRate:          successRate,
		Uptime:               uptime,
		Downtime:             downtime,
		LastProbeTime:        cp.state.LastProbeTime,
		LastSuccessTime:      cp.state.LastSuccessTime,
		LastFailureTime:      cp.state.LastFailureTime,
		IsReconnecting:       cp.state.IsReconnecting,
		ReconnectAttempts:    cp.state.ReconnectAttempts,
	}
}

// ProbeMetrics contains probe performance metrics
type ProbeMetrics struct {
	TotalProbes          int64         `json:"total_probes"`
	TotalFailures        int64         `json:"total_failures"`
	ConsecutiveFailures  int           `json:"consecutive_failures"`
	ConsecutiveSuccesses int           `json:"consecutive_successes"`
	SuccessRate          float64       `json:"success_rate"`
	Uptime               time.Duration `json:"uptime"`
	Downtime             time.Duration `json:"downtime"`
	LastProbeTime        time.Time     `json:"last_probe_time"`
	LastSuccessTime      time.Time     `json:"last_success_time"`
	LastFailureTime      time.Time     `json:"last_failure_time"`
	IsReconnecting       bool          `json:"is_reconnecting"`
	ReconnectAttempts    int           `json:"reconnect_attempts"`
}
