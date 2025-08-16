package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DockerTestHelper manages MySQL containers for testing
type DockerTestHelper struct {
	container testcontainers.Container
	pool      *Pool
	config    Config
	dsn       string
}

// DockerTestConfig holds configuration for Docker test containers
type DockerTestConfig struct {
	MySQLVersion string        // MySQL version to use (default: "8.0")
	Database     string        // Database name (default: "testdb")
	Username     string        // Username (default: "testuser")
	Password     string        // Password (default: "testpass")
	RootPassword string        // Root password (default: "rootpass")
	StartTimeout time.Duration // Container start timeout (default: 60s)
	Port         string        // Exposed port (default: auto-assigned)
}

// DefaultDockerTestConfig returns default configuration for Docker tests
func DefaultDockerTestConfig() DockerTestConfig {
	return DockerTestConfig{
		MySQLVersion: "8.0",
		Database:     "testdb",
		Username:     "testuser",
		Password:     "testpass",
		RootPassword: "rootpass",
		StartTimeout: 60 * time.Second,
		Port:         "0", // Auto-assign port
	}
}

// NewDockerTestHelper creates a new Docker test helper with default configuration
func NewDockerTestHelper(ctx context.Context) (*DockerTestHelper, error) {
	return NewDockerTestHelperWithConfig(ctx, DefaultDockerTestConfig())
}

// NewDockerTestHelperWithConfig creates a new Docker test helper with custom configuration
func NewDockerTestHelperWithConfig(ctx context.Context, config DockerTestConfig) (*DockerTestHelper, error) {
	// Create MySQL container
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:"+config.MySQLVersion,
		mysql.WithDatabase(config.Database),
		mysql.WithUsername(config.Username),
		mysql.WithPassword(config.Password),
		testcontainers.WithEnv(map[string]string{
			"MYSQL_ROOT_PASSWORD": config.RootPassword,
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").
				WithOccurrence(1).
				WithStartupTimeout(config.StartTimeout),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start MySQL container: %w", err)
	}

	// Get connection details
	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		mysqlContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		mysqlContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	// Build DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		config.Username, config.Password, host, port.Port(), config.Database)

	// Parse port to int
	portInt, err := strconv.Atoi(port.Port())
	if err != nil {
		mysqlContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to parse port: %w", err)
	}

	// Create pool configuration
	poolConfig := Config{
		Host:     host,
		Port:     portInt,
		Database: config.Database,
		Username: config.Username,
		Password: config.Password,
		DSN:      dsn,
	}

	// Create connection pool
	pool, err := NewPool(ctx, poolConfig)
	if err != nil {
		mysqlContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		mysqlContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DockerTestHelper{
		container: mysqlContainer,
		pool:      pool,
		config:    poolConfig,
		dsn:       dsn,
	}, nil
}

// Pool returns the database connection pool
func (h *DockerTestHelper) Pool() *Pool {
	return h.pool
}

// Config returns the pool configuration
func (h *DockerTestHelper) Config() Config {
	return h.config
}

// DSN returns the database connection string
func (h *DockerTestHelper) DSN() string {
	return h.dsn
}

// DB returns the underlying sql.DB instance
func (h *DockerTestHelper) DB() *sql.DB {
	return h.pool.db
}

// Container returns the underlying testcontainer
func (h *DockerTestHelper) Container() testcontainers.Container {
	return h.container
}

// Close closes the connection pool and terminates the container
func (h *DockerTestHelper) Close() error {
	var err error
	
	// Close the pool first
	if h.pool != nil {
		if poolErr := h.pool.Close(); poolErr != nil {
			err = fmt.Errorf("failed to close pool: %w", poolErr)
		}
	}
	
	// Terminate the container
	if h.container != nil {
		ctx := context.Background()
		if containerErr := h.container.Terminate(ctx); containerErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to terminate container: %w", err, containerErr)
			} else {
				err = fmt.Errorf("failed to terminate container: %w", containerErr)
			}
		}
	}
	
	return err
}

// Reset clears all data from the test database
func (h *DockerTestHelper) Reset(ctx context.Context) error {
	if h.pool == nil {
		return fmt.Errorf("pool is not initialized")
	}
	
	return h.pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Get all table names
		rows, err := conn.Query(ctx, "SHOW TABLES")
		if err != nil {
			return fmt.Errorf("failed to get table list: %w", err)
		}
		defer rows.Close()
		
		var tables []string
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return fmt.Errorf("failed to scan table name: %w", err)
			}
			tables = append(tables, tableName)
		}
		
		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating table names: %w", err)
		}
		
		// Disable foreign key checks
		if _, err := conn.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
			return fmt.Errorf("failed to disable foreign key checks: %w", err)
		}
		
		// Drop all tables
		for _, table := range tables {
			if _, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)); err != nil {
				return fmt.Errorf("failed to drop table %s: %w", table, err)
			}
		}
		
		// Re-enable foreign key checks
		if _, err := conn.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
			return fmt.Errorf("failed to re-enable foreign key checks: %w", err)
		}
		
		return nil
	})
}

// CreateTable creates a table with the given DDL
func (h *DockerTestHelper) CreateTable(ctx context.Context, ddl string) error {
	if h.pool == nil {
		return fmt.Errorf("pool is not initialized")
	}
	
	return h.pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, ddl)
		return err
	})
}

// ExecSQL executes arbitrary SQL
func (h *DockerTestHelper) ExecSQL(ctx context.Context, query string, args ...any) error {
	if h.pool == nil {
		return fmt.Errorf("pool is not initialized")
	}
	
	return h.pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, query, args...)
		return err
	})
}

// QuerySQL executes a query and returns the result
func (h *DockerTestHelper) QuerySQL(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if h.pool == nil {
		return nil, fmt.Errorf("pool is not initialized")
	}
	
	var rows *sql.Rows
	err := h.pool.WithConn(ctx, func(conn DatabaseConn) error {
		var err error
		rows, err = conn.Query(ctx, query, args...)
		return err
	})
	
	return rows, err
}

// WaitForReady waits for the database to be ready for connections
func (h *DockerTestHelper) WaitForReady(ctx context.Context, timeout time.Duration) error {
	if h.pool == nil {
		return fmt.Errorf("pool is not initialized")
	}
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database to be ready: %w", ctx.Err())
		case <-ticker.C:
			if err := h.pool.Ping(ctx); err == nil {
				return nil
			}
		}
	}
}

// GetConnectionInfo returns connection information for debugging
func (h *DockerTestHelper) GetConnectionInfo(ctx context.Context) (map[string]string, error) {
	if h.container == nil {
		return nil, fmt.Errorf("container is not initialized")
	}
	
	host, err := h.container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	
	port, err := h.container.MappedPort(ctx, "3306")
	if err != nil {
		return nil, fmt.Errorf("failed to get port: %w", err)
	}
	
	return map[string]string{
		"host": host,
		"port": port.Port(),
		"dsn":  h.dsn,
	}, nil
}
