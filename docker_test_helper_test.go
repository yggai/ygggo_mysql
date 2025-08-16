package ygggo_mysql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerTestHelper_BasicFunctionality tests basic Docker test helper functionality
func TestDockerTestHelper_BasicFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	// Create Docker test helper
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	// Test pool is available
	pool := helper.Pool()
	require.NotNil(t, pool)
	
	// Test connection
	err = pool.Ping(ctx)
	require.NoError(t, err)
	
	// Test DSN is valid
	dsn := helper.DSN()
	assert.NotEmpty(t, dsn)
	assert.Contains(t, dsn, "testuser")
	assert.Contains(t, dsn, "testdb")
	
	// Test config
	config := helper.Config()
	assert.NotEmpty(t, config.DSN)
}

// TestDockerTestHelper_CustomConfig tests Docker helper with custom configuration
func TestDockerTestHelper_CustomConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	// Custom configuration
	config := DockerTestConfig{
		MySQLVersion: "8.0",
		Database:     "customdb",
		Username:     "customuser",
		Password:     "custompass",
		RootPassword: "customroot",
		StartTimeout: 90 * time.Second,
	}
	
	helper, err := NewDockerTestHelperWithConfig(ctx, config)
	require.NoError(t, err)
	defer helper.Close()
	
	// Verify custom configuration
	dsn := helper.DSN()
	assert.Contains(t, dsn, "customuser")
	assert.Contains(t, dsn, "customdb")
	
	// Test connection works
	err = helper.Pool().Ping(ctx)
	require.NoError(t, err)
}

// TestDockerTestHelper_DatabaseOperations tests basic database operations
func TestDockerTestHelper_DatabaseOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	// Create a test table
	ddl := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	
	err = helper.CreateTable(ctx, ddl)
	require.NoError(t, err)
	
	// Insert test data
	err = helper.ExecSQL(ctx, "INSERT INTO test_users (name, email) VALUES (?, ?)", "John Doe", "john@example.com")
	require.NoError(t, err)
	
	err = helper.ExecSQL(ctx, "INSERT INTO test_users (name, email) VALUES (?, ?)", "Jane Smith", "jane@example.com")
	require.NoError(t, err)
	
	// Query data
	rows, err := helper.QuerySQL(ctx, "SELECT id, name, email FROM test_users ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()
	
	var users []struct {
		ID    int
		Name  string
		Email string
	}
	
	for rows.Next() {
		var user struct {
			ID    int
			Name  string
			Email string
		}
		err := rows.Scan(&user.ID, &user.Name, &user.Email)
		require.NoError(t, err)
		users = append(users, user)
	}
	
	require.NoError(t, rows.Err())
	require.Len(t, users, 2)
	assert.Equal(t, "John Doe", users[0].Name)
	assert.Equal(t, "Jane Smith", users[1].Name)
}

// TestDockerTestHelper_Reset tests database reset functionality
func TestDockerTestHelper_Reset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	// Create tables and data
	err = helper.CreateTable(ctx, "CREATE TABLE table1 (id INT PRIMARY KEY, name VARCHAR(50))")
	require.NoError(t, err)
	
	err = helper.CreateTable(ctx, "CREATE TABLE table2 (id INT PRIMARY KEY, value VARCHAR(50))")
	require.NoError(t, err)
	
	err = helper.ExecSQL(ctx, "INSERT INTO table1 (id, name) VALUES (1, 'test')")
	require.NoError(t, err)
	
	// Verify tables exist
	rows, err := helper.QuerySQL(ctx, "SHOW TABLES")
	require.NoError(t, err)
	
	var tableCount int
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		tableCount++
	}
	rows.Close()
	
	assert.Equal(t, 2, tableCount)
	
	// Reset database
	err = helper.Reset(ctx)
	require.NoError(t, err)
	
	// Verify tables are gone
	rows, err = helper.QuerySQL(ctx, "SHOW TABLES")
	require.NoError(t, err)
	
	tableCount = 0
	for rows.Next() {
		tableCount++
	}
	rows.Close()
	
	assert.Equal(t, 0, tableCount)
}

// TestDockerTestHelper_PoolIntegration tests integration with Pool functionality
func TestDockerTestHelper_PoolIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	pool := helper.Pool()
	
	// Test WithConn
	err = pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, "CREATE TABLE test_pool (id INT PRIMARY KEY, data VARCHAR(100))")
		return err
	})
	require.NoError(t, err)
	
	// Test WithinTx
	err = pool.WithinTx(ctx, func(tx DatabaseTx) error {
		_, err := tx.Exec(ctx, "INSERT INTO test_pool (id, data) VALUES (1, 'transaction test')")
		return err
	})
	require.NoError(t, err)
	
	// Verify data
	var count int
	err = pool.WithConn(ctx, func(conn DatabaseConn) error {
		rows, err := conn.Query(ctx, "SELECT COUNT(*) FROM test_pool")
		if err != nil {
			return err
		}
		defer rows.Close()
		
		if rows.Next() {
			return rows.Scan(&count)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestDockerTestHelper_ConcurrentAccess tests concurrent access to the database
func TestDockerTestHelper_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	// Create test table
	err = helper.CreateTable(ctx, "CREATE TABLE concurrent_test (id INT AUTO_INCREMENT PRIMARY KEY, worker_id INT)")
	require.NoError(t, err)
	
	pool := helper.Pool()
	
	// Run concurrent operations
	const numWorkers = 5
	const opsPerWorker = 10
	
	done := make(chan error, numWorkers)
	
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < opsPerWorker; j++ {
				err := pool.WithConn(ctx, func(conn DatabaseConn) error {
					_, err := conn.Exec(ctx, "INSERT INTO concurrent_test (worker_id) VALUES (?)", workerID)
					return err
				})
				if err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}(i)
	}
	
	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		err := <-done
		require.NoError(t, err)
	}
	
	// Verify total count
	var totalCount int
	err = pool.WithConn(ctx, func(conn DatabaseConn) error {
		rows, err := conn.Query(ctx, "SELECT COUNT(*) FROM concurrent_test")
		if err != nil {
			return err
		}
		defer rows.Close()
		
		if rows.Next() {
			return rows.Scan(&totalCount)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, numWorkers*opsPerWorker, totalCount)
}

// TestDockerTestHelper_ConnectionInfo tests connection info retrieval
func TestDockerTestHelper_ConnectionInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	info, err := helper.GetConnectionInfo(ctx)
	require.NoError(t, err)
	
	assert.NotEmpty(t, info["host"])
	assert.NotEmpty(t, info["port"])
	assert.NotEmpty(t, info["dsn"])
	
	// Host should be localhost or similar
	assert.Contains(t, []string{"localhost", "127.0.0.1"}, info["host"])
	
	// Port should be a number
	assert.Regexp(t, `^\d+$`, info["port"])
}

// TestDockerTestHelper_WaitForReady tests waiting for database readiness
func TestDockerTestHelper_WaitForReady(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}
	
	ctx := context.Background()
	
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()
	
	// Should be ready immediately since container is already started
	err = helper.WaitForReady(ctx, 5*time.Second)
	require.NoError(t, err)
	
	// Test with very short timeout (should still succeed since it's ready)
	err = helper.WaitForReady(ctx, 100*time.Millisecond)
	require.NoError(t, err)
}
