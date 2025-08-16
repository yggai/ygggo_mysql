// Package ygggo_mysql provides a comprehensive, production-ready MySQL database access library for Go.
//
// # Overview
//
// ygggo_mysql is designed for enterprise applications that require high performance,
// reliability, and comprehensive observability. It builds upon Go's standard database/sql
// package while adding essential production features.
//
// # Key Features
//
// ## Connection Management
//   - Advanced connection pooling with configurable limits and timeouts
//   - Connection leak detection and monitoring
//   - Health checks and automatic recovery
//   - Graceful shutdown and resource cleanup
//
// ## Transaction Support
//   - Automatic transaction management with commit/rollback
//   - Configurable retry policies for transient failures
//   - Deadlock detection and automatic retry
//   - Nested transaction support (savepoints)
//
// ## Performance Optimization
//   - Prepared statement caching with LRU eviction
//   - Bulk insert operations for high-throughput scenarios
//   - Query streaming for large result sets
//   - Connection reuse and efficient resource utilization
//
// ## Observability
//   - OpenTelemetry integration for distributed tracing
//   - Prometheus-compatible metrics collection
//   - Structured logging with configurable levels
//   - Slow query detection and analysis
//   - Performance monitoring and alerting
//
// ## Developer Experience
//   - Type-safe query builders and named parameters
//   - Comprehensive error handling and classification
//   - Extensive documentation and examples
//   - Testing utilities and mock support
//
// # Quick Start
//
// ## Basic Usage
//
//	package main
//
//	import (
//		"context"
//		"log"
//		ggm "github.com/yggai/ygggo_mysql"
//	)
//
//	func main() {
//		// Configure database connection
//		config := ggm.Config{
//			Host:     "localhost",
//			Port:     3306,
//			Username: "user",
//			Password: "password",
//			Database: "mydb",
//			Driver:   "mysql",
//		}
//
//		// Create connection pool
//		ctx := context.Background()
//		pool, err := ggm.NewPool(ctx, config)
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer pool.Close()
//
//		// Execute queries safely
//		err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
//			result, err := conn.Exec(ctx, "INSERT INTO users (name, age) VALUES (?, ?)", "Alice", 30)
//			if err != nil {
//				return err
//			}
//			
//			id, err := result.LastInsertId()
//			if err != nil {
//				return err
//			}
//			
//			log.Printf("Created user with ID: %d", id)
//			return nil
//		})
//		
//		if err != nil {
//			log.Printf("Operation failed: %v", err)
//		}
//	}
//
// ## Transaction Example
//
//	// Money transfer with ACID guarantees
//	err := pool.WithinTx(ctx, func(tx ggm.DatabaseTx) error {
//		// Debit source account
//		result, err := tx.Exec(ctx, 
//			"UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?",
//			amount, fromID, amount)
//		if err != nil {
//			return err
//		}
//		
//		affected, _ := result.RowsAffected()
//		if affected == 0 {
//			return errors.New("insufficient funds")
//		}
//		
//		// Credit destination account
//		_, err = tx.Exec(ctx,
//			"UPDATE accounts SET balance = balance + ? WHERE id = ?",
//			amount, toID)
//		return err
//	})
//
// ## Bulk Operations
//
//	// High-performance bulk insert
//	columns := []string{"name", "email", "age"}
//	rows := [][]any{
//		{"Alice", "alice@example.com", 30},
//		{"Bob", "bob@example.com", 25},
//		{"Charlie", "charlie@example.com", 35},
//	}
//	
//	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
//		result, err := conn.BulkInsert(ctx, "users", columns, rows)
//		if err != nil {
//			return err
//		}
//		
//		affected, _ := result.RowsAffected()
//		log.Printf("Inserted %d users", affected)
//		return nil
//	})
//
// # Configuration
//
// ## Programmatic Configuration
//
//	config := ggm.Config{
//		// Connection settings
//		Host:     "localhost",
//		Port:     3306,
//		Username: "user",
//		Password: "password",
//		Database: "mydb",
//		
//		// Pool configuration
//		Pool: ggm.PoolConfig{
//			MaxOpen:         25,
//			MaxIdle:         10,
//			ConnMaxLifetime: 5 * time.Minute,
//			ConnMaxIdleTime: 2 * time.Minute,
//		},
//		
//		// Performance settings
//		SlowQueryThreshold: 100 * time.Millisecond,
//		
//		// Retry policy
//		Retry: ggm.RetryPolicy{
//			MaxAttempts: 3,
//			BaseBackoff: 10 * time.Millisecond,
//		},
//	}
//
// ## Environment Variables
//
// All configuration can be overridden using environment variables:
//
//	export YGGGO_MYSQL_HOST=localhost
//	export YGGGO_MYSQL_PORT=3306
//	export YGGGO_MYSQL_USERNAME=user
//	export YGGGO_MYSQL_PASSWORD=secret
//	export YGGGO_MYSQL_DATABASE=mydb
//
// # Advanced Features
//
// ## Connection Leak Detection
//
//	pool.SetBorrowWarnThreshold(30 * time.Second)
//	pool.SetLeakHandler(func(leak ggm.BorrowLeak) {
//		log.Printf("WARN: Connection held for %v", leak.HeldFor)
//		// Send alert, increment metrics, etc.
//	})
//
// ## Prepared Statement Caching
//
//	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
//		conn.EnableStmtCache(100) // Cache up to 100 prepared statements
//		
//		// Subsequent calls will reuse prepared statements
//		for i := 0; i < 1000; i++ {
//			_, err := conn.ExecCached(ctx, "INSERT INTO logs (message) VALUES (?)", 
//				fmt.Sprintf("Log entry %d", i))
//			if err != nil {
//				return err
//			}
//		}
//		return nil
//	})
//
// ## Query Streaming
//
//	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
//		return conn.QueryStream(ctx, "SELECT * FROM large_table", 
//			func(row []any) error {
//				// Process each row without loading entire result set
//				log.Printf("Processing row: %v", row)
//				return nil
//			})
//	})
//
// # Error Handling
//
// The library provides comprehensive error classification and handling:
//
//	err := pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
//		_, err := conn.Exec(ctx, "INSERT INTO users (email) VALUES (?)", "duplicate@example.com")
//		return err
//	})
//	
//	if err != nil {
//		// Check for specific error types
//		if isDuplicateKeyError(err) {
//			log.Printf("Duplicate key: %v", err)
//		} else if isConnectionError(err) {
//			log.Printf("Connection issue: %v", err)
//		} else {
//			log.Printf("Other error: %v", err)
//		}
//	}
//
// # Testing
//
// The library provides interfaces that make testing easy:
//
//	func TestUserService(t *testing.T) {
//		// Use mock implementation for testing
//		mockPool := &MockDatabasePool{}
//		service := NewUserService(mockPool)
//		
//		// Test business logic without real database
//		err := service.CreateUser(ctx, "Alice", "alice@example.com")
//		assert.NoError(t, err)
//	}
//
// # Performance Considerations
//
//   - Use connection pooling (WithConn) instead of acquiring individual connections
//   - Enable prepared statement caching for frequently executed queries
//   - Use bulk operations for inserting multiple rows
//   - Configure appropriate pool sizes based on your workload
//   - Monitor slow queries and optimize them
//   - Use transactions judiciously to balance consistency and performance
//
// # Best Practices
//
//   - Always use context.Context for cancellation and timeouts
//   - Use placeholder parameters (?) to prevent SQL injection
//   - Handle errors appropriately and implement retry logic for transient failures
//   - Monitor connection pool metrics and adjust configuration as needed
//   - Use structured logging for better observability
//   - Implement health checks for your database connections
//   - Test your database interactions thoroughly, including error scenarios
//
// For more examples and detailed documentation, see the examples/ directory
// in the repository and the individual type and method documentation.
package ygggo_mysql
