package ygggo_mysql

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Enhanced PoolConfig with comprehensive connection pool settings
type EnhancedPoolConfig struct {
	// Basic pool settings (existing)
	MaxOpen         int           `json:"max_open"`
	MaxIdle         int           `json:"max_idle"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`

	// Advanced settings
	MinOpen         int           `json:"min_open"`          // Minimum open connections
	ConnectTimeout  time.Duration `json:"connect_timeout"`   // Connection establishment timeout
	PingTimeout     time.Duration `json:"ping_timeout"`      // Connection ping timeout

	// Health and monitoring
	HealthCheckInterval time.Duration `json:"health_check_interval"` // Health check interval
	EnableHealthCheck   bool          `json:"enable_health_check"`   // Enable automatic health checks

	// Connection validation
	ValidateOnBorrow bool `json:"validate_on_borrow"` // Validate connections when borrowed
	ValidateOnReturn bool `json:"validate_on_return"` // Validate connections when returned

	// Leak detection
	LeakDetectionThreshold time.Duration `json:"leak_detection_threshold"` // Connection leak detection threshold
	EnableLeakDetection    bool          `json:"enable_leak_detection"`    // Enable connection leak detection

	// Performance tuning
	PreparedStatementCacheSize int  `json:"prepared_statement_cache_size"` // Prepared statement cache size
	EnablePreparedStatements   bool `json:"enable_prepared_statements"`    // Enable prepared statement caching
}

// DetailedPoolStats represents comprehensive real-time pool statistics
type DetailedPoolStats struct {
	// Basic statistics from sql.DBStats
	OpenConnections   int           `json:"open_connections"`
	InUse            int           `json:"in_use"`
	Idle             int           `json:"idle"`
	WaitCount        int64         `json:"wait_count"`
	WaitDuration     time.Duration `json:"wait_duration"`
	MaxIdleClosed    int64         `json:"max_idle_closed"`
	MaxLifetimeClosed int64        `json:"max_lifetime_closed"`
	MaxIdleTimeClosed int64        `json:"max_idle_time_closed"`

	// Enhanced statistics
	TotalConnections  int64 `json:"total_connections"`  // Total connections created
	FailedConnections int64 `json:"failed_connections"` // Failed connection attempts
	LeakedConnections int64 `json:"leaked_connections"` // Detected connection leaks

	// Performance metrics
	AverageWaitTime   time.Duration `json:"average_wait_time"`   // Average wait time for connections
	ConnectionUtilization float64   `json:"connection_utilization"` // Connection utilization percentage
}

// PoolHealthStatus represents the health status of the connection pool
type PoolHealthStatus struct {
	Healthy      bool          `json:"healthy"`
	LastChecked  time.Time     `json:"last_checked"`
	ResponseTime time.Duration `json:"response_time"`
	Issues       []string      `json:"issues,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// PoolManager provides comprehensive connection pool management
type PoolManager struct {
	pool   *Pool
	config EnhancedPoolConfig
	mutex  sync.RWMutex

	// Statistics tracking
	totalConnections  int64
	failedConnections int64
	leakedConnections int64
}

// NewPoolManager creates a new pool manager for the given pool
func NewPoolManager(pool *Pool) *PoolManager {
	return &PoolManager{
		pool:   pool,
		config: convertToEnhancedConfig(DefaultPoolConfig()),
	}
}

// ValidatePoolConfig validates a pool configuration
func ValidatePoolConfig(config PoolConfig) error {
	if config.MaxOpen <= 0 {
		return fmt.Errorf("MaxOpen must be positive, got %d", config.MaxOpen)
	}
	
	if config.MaxIdle < 0 {
		return fmt.Errorf("MaxIdle must be non-negative, got %d", config.MaxIdle)
	}
	
	if config.MaxIdle > config.MaxOpen {
		return fmt.Errorf("MaxIdle cannot be greater than MaxOpen (MaxIdle: %d, MaxOpen: %d)", 
			config.MaxIdle, config.MaxOpen)
	}
	
	if config.ConnMaxLifetime < 0 {
		return fmt.Errorf("ConnMaxLifetime must be positive, got %v", config.ConnMaxLifetime)
	}
	
	if config.ConnMaxIdleTime < 0 {
		return fmt.Errorf("ConnMaxIdleTime must be positive, got %v", config.ConnMaxIdleTime)
	}
	
	return nil
}

// DefaultPoolConfig returns a default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpen:         25,
		MaxIdle:         10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// DevelopmentPoolConfig returns a pool configuration optimized for development
func DevelopmentPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpen:         5,
		MaxIdle:         2,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// ProductionPoolConfig returns a pool configuration optimized for production
func ProductionPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpen:         50,
		MaxIdle:         20,
		ConnMaxLifetime: 60 * time.Minute,
		ConnMaxIdleTime: 30 * time.Minute,
	}
}

// TestingPoolConfig returns a pool configuration optimized for testing
func TestingPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpen:         3,
		MaxIdle:         1,
		ConnMaxLifetime: 2 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// HighPerformancePoolConfig returns a pool configuration optimized for high performance
func HighPerformancePoolConfig() PoolConfig {
	config := PoolConfig{
		MaxOpen:         100,
		MaxIdle:         50,
		ConnMaxLifetime: 120 * time.Minute,
		ConnMaxIdleTime: 60 * time.Minute,
	}
	
	// Note: Enhanced features would be set in EnhancedPoolConfig
	return config
}

// GetConfig returns the current pool configuration
func (pm *PoolManager) GetConfig() PoolConfig {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	return convertFromEnhancedConfig(pm.config)
}

// UpdateConfig updates the pool configuration
func (pm *PoolManager) UpdateConfig(config PoolConfig) error {
	if err := ValidatePoolConfig(config); err != nil {
		return fmt.Errorf("invalid pool configuration: %w", err)
	}
	
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	// Apply configuration to the underlying database
	if pm.pool != nil && pm.pool.db != nil {
		pm.pool.db.SetMaxOpenConns(config.MaxOpen)
		pm.pool.db.SetMaxIdleConns(config.MaxIdle)
		pm.pool.db.SetConnMaxLifetime(config.ConnMaxLifetime)
		pm.pool.db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}
	
	// Update internal configuration
	pm.config = convertToEnhancedConfig(config)
	
	return nil
}

// Stats returns current pool statistics
func (pm *PoolManager) Stats() DetailedPoolStats {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	if pm.pool == nil || pm.pool.db == nil {
		return DetailedPoolStats{}
	}

	dbStats := pm.pool.db.Stats()

	stats := DetailedPoolStats{
		OpenConnections:   dbStats.OpenConnections,
		InUse:            dbStats.InUse,
		Idle:             dbStats.Idle,
		WaitCount:        dbStats.WaitCount,
		WaitDuration:     dbStats.WaitDuration,
		MaxIdleClosed:    dbStats.MaxIdleClosed,
		MaxLifetimeClosed: dbStats.MaxLifetimeClosed,
		MaxIdleTimeClosed: dbStats.MaxIdleTimeClosed,
		TotalConnections:  pm.totalConnections,
		FailedConnections: pm.failedConnections,
		LeakedConnections: pm.leakedConnections,
	}
	
	// Calculate derived metrics
	if dbStats.WaitCount > 0 {
		stats.AverageWaitTime = time.Duration(dbStats.WaitDuration.Nanoseconds() / dbStats.WaitCount)
	}
	
	if pm.config.MaxOpen > 0 {
		stats.ConnectionUtilization = float64(dbStats.InUse) / float64(pm.config.MaxOpen) * 100
	}
	
	return stats
}

// HealthCheck performs a comprehensive health check of the pool
func (pm *PoolManager) HealthCheck(ctx context.Context) (*PoolHealthStatus, error) {
	start := time.Now()
	
	health := &PoolHealthStatus{
		LastChecked: start,
		Healthy:     true,
		Issues:      make([]string, 0),
		Details:     make(map[string]interface{}),
	}
	
	// Basic connectivity check
	if pm.pool != nil {
		if err := pm.pool.Ping(ctx); err != nil {
			health.Healthy = false
			health.Issues = append(health.Issues, fmt.Sprintf("Ping failed: %v", err))
		}
	} else {
		health.Healthy = false
		health.Issues = append(health.Issues, "Pool is nil")
		return health, fmt.Errorf("pool is nil")
	}
	
	// Check pool statistics for issues
	stats := pm.Stats()
	health.Details["stats"] = stats
	
	// Check for potential issues
	if stats.ConnectionUtilization > 90 {
		health.Issues = append(health.Issues, "High connection utilization (>90%)")
	}
	
	if stats.WaitCount > 0 && stats.AverageWaitTime > 100*time.Millisecond {
		health.Issues = append(health.Issues, "High average wait time for connections")
	}
	
	if stats.FailedConnections > 0 {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed connections detected: %d", stats.FailedConnections))
	}
	
	health.ResponseTime = time.Since(start)
	
	return health, nil
}

// WarmUp pre-creates connections up to the minimum required
func (pm *PoolManager) WarmUp(ctx context.Context) error {
	if pm.pool == nil {
		return fmt.Errorf("pool is nil")
	}
	
	// For basic warm-up, we'll acquire and immediately release connections
	// This forces the pool to create connections up to MaxIdle
	config := pm.GetConfig()
	connections := make([]DatabaseConn, 0, config.MaxIdle)
	
	// Acquire connections
	for i := 0; i < config.MaxIdle; i++ {
		conn, err := pm.pool.Acquire(ctx)
		if err != nil {
			// Release any connections we've acquired
			for _, c := range connections {
				c.Close()
			}
			return fmt.Errorf("failed to warm up pool: %w", err)
		}
		connections = append(connections, conn)
	}
	
	// Release all connections back to the pool
	for _, conn := range connections {
		conn.Close()
	}
	
	return nil
}

// ScaleUp increases the maximum number of open connections
func (pm *PoolManager) ScaleUp(additionalConnections int) error {
	if additionalConnections <= 0 {
		return fmt.Errorf("additionalConnections must be positive, got %d", additionalConnections)
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	newMaxOpen := pm.config.MaxOpen + additionalConnections

	if pm.pool != nil && pm.pool.db != nil {
		pm.pool.db.SetMaxOpenConns(newMaxOpen)
		pm.config.MaxOpen = newMaxOpen
	}

	return nil
}

// ScaleDown decreases the maximum number of open connections
func (pm *PoolManager) ScaleDown(connectionsToRemove int) error {
	if connectionsToRemove <= 0 {
		return fmt.Errorf("connectionsToRemove must be positive, got %d", connectionsToRemove)
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	newMaxOpen := pm.config.MaxOpen - connectionsToRemove
	if newMaxOpen <= 0 {
		return fmt.Errorf("cannot scale down to %d or fewer connections", newMaxOpen)
	}

	// Ensure MaxIdle doesn't exceed new MaxOpen
	newMaxIdle := pm.config.MaxIdle
	if newMaxIdle > newMaxOpen {
		newMaxIdle = newMaxOpen
	}

	if pm.pool != nil && pm.pool.db != nil {
		pm.pool.db.SetMaxOpenConns(newMaxOpen)
		pm.pool.db.SetMaxIdleConns(newMaxIdle)
		pm.config.MaxOpen = newMaxOpen
		pm.config.MaxIdle = newMaxIdle
	}

	return nil
}

// Resize sets new maximum open and idle connection limits
func (pm *PoolManager) Resize(newMaxOpen, newMaxIdle int) error {
	if newMaxOpen <= 0 {
		return fmt.Errorf("newMaxOpen must be positive, got %d", newMaxOpen)
	}
	if newMaxIdle < 0 {
		return fmt.Errorf("newMaxIdle must be non-negative, got %d", newMaxIdle)
	}
	if newMaxIdle > newMaxOpen {
		return fmt.Errorf("newMaxIdle cannot exceed newMaxOpen (%d > %d)", newMaxIdle, newMaxOpen)
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.pool != nil && pm.pool.db != nil {
		pm.pool.db.SetMaxOpenConns(newMaxOpen)
		pm.pool.db.SetMaxIdleConns(newMaxIdle)
		pm.config.MaxOpen = newMaxOpen
		pm.config.MaxIdle = newMaxIdle
	}

	return nil
}

// DrainConnections gracefully closes idle connections
func (pm *PoolManager) DrainConnections(ctx context.Context) error {
	if pm.pool == nil || pm.pool.db == nil {
		return fmt.Errorf("pool is not initialized")
	}

	pm.mutex.Lock()
	originalMaxIdle := pm.config.MaxIdle
	pm.mutex.Unlock()

	// Set MaxIdle to 0 to prevent new idle connections
	pm.pool.db.SetMaxIdleConns(0)

	// Wait a bit for connections to be released
	select {
	case <-ctx.Done():
		// Restore original MaxIdle if context was cancelled
		pm.pool.db.SetMaxIdleConns(originalMaxIdle)
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Continue with drain
	}

	// Restore original MaxIdle
	pm.pool.db.SetMaxIdleConns(originalMaxIdle)

	return nil
}

// IsHealthy returns true if the pool is in a healthy state
func (pm *PoolManager) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := pm.HealthCheck(ctx)
	if err != nil {
		return false
	}

	return health.Healthy
}

// GetConnectionUtilization returns the current connection utilization percentage
func (pm *PoolManager) GetConnectionUtilization() float64 {
	stats := pm.Stats()
	return stats.ConnectionUtilization
}

// Helper functions to convert between PoolConfig and EnhancedPoolConfig
func convertToEnhancedConfig(config PoolConfig) EnhancedPoolConfig {
	return EnhancedPoolConfig{
		MaxOpen:                    config.MaxOpen,
		MaxIdle:                    config.MaxIdle,
		ConnMaxLifetime:            config.ConnMaxLifetime,
		ConnMaxIdleTime:            config.ConnMaxIdleTime,
		MinOpen:                    0,
		ConnectTimeout:             30 * time.Second,
		PingTimeout:                5 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		EnableHealthCheck:          false,
		ValidateOnBorrow:           false,
		ValidateOnReturn:           false,
		LeakDetectionThreshold:     5 * time.Minute,
		EnableLeakDetection:        false,
		PreparedStatementCacheSize: 100,
		EnablePreparedStatements:   false,
	}
}

func convertFromEnhancedConfig(config EnhancedPoolConfig) PoolConfig {
	return PoolConfig{
		MaxOpen:         config.MaxOpen,
		MaxIdle:         config.MaxIdle,
		ConnMaxLifetime: config.ConnMaxLifetime,
		ConnMaxIdleTime: config.ConnMaxIdleTime,
	}
}
