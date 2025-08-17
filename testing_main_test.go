package ygggo_mysql

import (
	"context"
	"os"
	"testing"
	"time"

	gge "github.com/yggai/ygggo_env"
)

// TestHelper provides a simple interface for tests to get a database pool
// that connects to the test MySQL container managed by TestMain
type TestHelper struct {
	pool   *Pool
	config Config
}

// NewTestHelper creates a new test helper that connects to the test MySQL container
// This replaces the old NewDockerTestHelper approach with a simpler environment-based approach
func NewTestHelper(ctx context.Context) (*TestHelper, error) {
	// Create config with environment variables applied
	config := Config{}
	applyEnv(&config)
	if config.Driver == "" {
		config.Driver = "mysql"
	}

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		return nil, err
	}
	return &TestHelper{pool: pool, config: config}, nil
}

// Pool returns the database pool for testing
func (h *TestHelper) Pool() *Pool {
	return h.pool
}

// Config returns the configuration used by this test helper
func (h *TestHelper) Config() Config {
	return h.config
}

// Close closes the test helper and its resources
func (h *TestHelper) Close() error {
	if h.pool != nil {
		return h.pool.Close()
	}
	return nil
}

// NewDockerTestHelper is an alias for NewTestHelper to maintain backward compatibility
// This removes the docker-in-test dependency while keeping the same interface
func NewDockerTestHelper(ctx context.Context) (*TestHelper, error) {
	return NewTestHelper(ctx)
}

// TestMain sets up Docker MySQL (auto) before tests and tears down after, within a strict timeout.
func TestMain(m *testing.M) {
	// Load .env
	gge.LoadEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ensure MySQL container is up (auto)
	if !IsMySQL(ctx) {
		// If container name not set, default
		if os.Getenv(EnvDockerContainerName) == "" {
			_ = os.Setenv(EnvDockerContainerName, "ygggo-mysql-test")
		}
		if err := NewMySQL(ctx); err != nil {
			// Print and exit with failure
			println("[TestMain] NewMySQL error:", err.Error())
			os.Exit(1)
		}
	}

	// Run all tests
	exitCode := m.Run()
	os.Exit(exitCode)
}
