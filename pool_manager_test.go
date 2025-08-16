package ygggo_mysql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  PoolConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: PoolConfig{
				MaxOpen:         10,
				MaxIdle:         5,
				ConnMaxLifetime: 30 * time.Minute,
				ConnMaxIdleTime: 10 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "invalid max open - negative",
			config: PoolConfig{
				MaxOpen: -1,
				MaxIdle: 5,
			},
			wantErr: true,
			errMsg:  "MaxOpen must be positive",
		},
		{
			name: "invalid max idle - greater than max open",
			config: PoolConfig{
				MaxOpen: 5,
				MaxIdle: 10,
			},
			wantErr: true,
			errMsg:  "MaxIdle cannot be greater than MaxOpen",
		},
		{
			name: "invalid connection lifetime - negative",
			config: PoolConfig{
				MaxOpen:         10,
				MaxIdle:         5,
				ConnMaxLifetime: -1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "ConnMaxLifetime must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePoolConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPoolConfig_Defaults(t *testing.T) {
	config := DefaultPoolConfig()

	assert.Greater(t, config.MaxOpen, 0, "MaxOpen should have a positive default")
	assert.Greater(t, config.MaxIdle, 0, "MaxIdle should have a positive default")
	assert.Greater(t, config.ConnMaxLifetime, time.Duration(0), "ConnMaxLifetime should have a positive default")
	assert.Greater(t, config.ConnMaxIdleTime, time.Duration(0), "ConnMaxIdleTime should have a positive default")
	assert.LessOrEqual(t, config.MaxIdle, config.MaxOpen, "MaxIdle should not exceed MaxOpen")
}

func TestPoolConfig_Presets(t *testing.T) {
	t.Run("development preset", func(t *testing.T) {
		config := DevelopmentPoolConfig()
		assert.NoError(t, ValidatePoolConfig(config))
		assert.LessOrEqual(t, config.MaxOpen, 10, "Development should have limited connections")
	})

	t.Run("production preset", func(t *testing.T) {
		config := ProductionPoolConfig()
		assert.NoError(t, ValidatePoolConfig(config))
		assert.Greater(t, config.MaxOpen, 10, "Production should support more connections")
		assert.Greater(t, config.ConnMaxLifetime, 10*time.Minute, "Production should have longer connection lifetime")
	})

	t.Run("testing preset", func(t *testing.T) {
		config := TestingPoolConfig()
		assert.NoError(t, ValidatePoolConfig(config))
		assert.LessOrEqual(t, config.MaxOpen, 5, "Testing should have minimal connections")
		assert.Less(t, config.ConnMaxLifetime, 5*time.Minute, "Testing should have short connection lifetime")
	})

	t.Run("high performance preset", func(t *testing.T) {
		config := HighPerformancePoolConfig()
		assert.NoError(t, ValidatePoolConfig(config))
		assert.Greater(t, config.MaxOpen, 20, "High performance should support many connections")
		// Note: EnablePreparedStatements would be in EnhancedPoolConfig
	})
}

func TestPoolManager_BasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	// Get configuration from helper and update pool config
	config := helper.Config()
	config.Pool = DevelopmentPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	// Test pool manager creation
	manager := NewPoolManager(pool)
	require.NotNil(t, manager)

	// Test getting configuration
	poolConfig := manager.GetConfig()
	assert.Greater(t, poolConfig.MaxOpen, 0)
	assert.Greater(t, poolConfig.MaxIdle, 0)

	// Test getting statistics
	stats := manager.Stats()
	assert.GreaterOrEqual(t, stats.OpenConnections, 0)
	assert.GreaterOrEqual(t, stats.InUse, 0)
	assert.GreaterOrEqual(t, stats.Idle, 0)
}

func TestPoolManager_Statistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = TestingPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Get initial statistics
	initialStats := manager.Stats()
	assert.GreaterOrEqual(t, initialStats.OpenConnections, 0)

	// Acquire a connection to change statistics
	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	// Check statistics after acquiring connection
	activeStats := manager.Stats()
	assert.GreaterOrEqual(t, activeStats.InUse, 1)

	// Release connection
	conn.Close()

	// Check statistics after releasing connection
	finalStats := manager.Stats()
	assert.Equal(t, initialStats.InUse, finalStats.InUse)
}

func TestPoolManager_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = DevelopmentPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Test health check
	health, err := manager.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, health)

	assert.True(t, health.Healthy, "Pool should be healthy")
	assert.False(t, health.LastChecked.IsZero(), "LastChecked should be set")
	assert.Greater(t, health.ResponseTime, time.Duration(0), "ResponseTime should be positive")
}

func TestPoolManager_ConfigUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = TestingPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Get initial configuration
	initialConfig := manager.GetConfig()
	originalMaxOpen := initialConfig.MaxOpen

	// Update configuration
	newConfig := initialConfig
	newConfig.MaxOpen = originalMaxOpen + 5

	err = manager.UpdateConfig(newConfig)
	assert.NoError(t, err)

	// Verify configuration was updated
	updatedConfig := manager.GetConfig()
	assert.Equal(t, originalMaxOpen+5, updatedConfig.MaxOpen)
}

func TestPoolManager_WarmUp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = DevelopmentPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Test warm up
	err = manager.WarmUp(ctx)
	assert.NoError(t, err)

	// Check that connections were created
	stats := manager.Stats()
	assert.Greater(t, stats.OpenConnections, 0, "WarmUp should create connections")
}

func TestPoolManager_Scaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = TestingPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Get initial configuration
	initialConfig := manager.GetConfig()
	originalMaxOpen := initialConfig.MaxOpen

	// Test scale up
	err = manager.ScaleUp(5)
	assert.NoError(t, err)

	updatedConfig := manager.GetConfig()
	assert.Equal(t, originalMaxOpen+5, updatedConfig.MaxOpen)

	// Test scale down
	err = manager.ScaleDown(3)
	assert.NoError(t, err)

	finalConfig := manager.GetConfig()
	assert.Equal(t, originalMaxOpen+2, finalConfig.MaxOpen)

	// Test invalid scale operations
	err = manager.ScaleUp(-1)
	assert.Error(t, err)

	err = manager.ScaleDown(-1)
	assert.Error(t, err)

	err = manager.ScaleDown(1000) // Too many to remove
	assert.Error(t, err)
}

func TestPoolManager_Resize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = DevelopmentPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Test resize
	err = manager.Resize(20, 10)
	assert.NoError(t, err)

	updatedConfig := manager.GetConfig()
	assert.Equal(t, 20, updatedConfig.MaxOpen)
	assert.Equal(t, 10, updatedConfig.MaxIdle)

	// Test invalid resize operations
	err = manager.Resize(-1, 5)
	assert.Error(t, err)

	err = manager.Resize(10, -1)
	assert.Error(t, err)

	err = manager.Resize(5, 10) // MaxIdle > MaxOpen
	assert.Error(t, err)
}

func TestPoolManager_DrainConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = DevelopmentPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Test drain connections (without warm up to avoid potential issues)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = manager.DrainConnections(ctx)
	assert.NoError(t, err)

	// Test with already cancelled context
	cancelledCtx, cancel2 := context.WithCancel(context.Background())
	cancel2()

	err = manager.DrainConnections(cancelledCtx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestPoolManager_HealthAndUtilization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	config := helper.Config()
	config.Pool = TestingPoolConfig()

	ctx := context.Background()
	pool, err := NewPool(ctx, config)
	require.NoError(t, err)
	defer pool.Close()

	manager := NewPoolManager(pool)

	// Test health check
	isHealthy := manager.IsHealthy()
	assert.True(t, isHealthy, "Pool should be healthy")

	// Test connection utilization
	utilization := manager.GetConnectionUtilization()
	assert.GreaterOrEqual(t, utilization, 0.0)
	assert.LessOrEqual(t, utilization, 100.0)
}
