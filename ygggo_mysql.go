// Package ygggo_mysql provides a comprehensive, production-ready MySQL database access library for Go.
//
// # Overview
//
// ygggo_mysql is a high-performance MySQL client library that offers:
//   - Connection pooling with leak detection and health monitoring
//   - Transaction management with automatic retry and rollback
//   - Query optimization with prepared statement caching
//   - Comprehensive observability (metrics, logging, tracing)
//   - Bulk operations and streaming query support
//   - Named parameter binding and query building
//   - Slow query detection and analysis
//
// # Quick Start
//
// Basic usage example:
//
//	import ggm "github.com/yggai/ygggo_mysql"
//
//	// Configure database connection
//	config := ggm.Config{
//		Host:     "localhost",
//		Port:     3306,
//		Username: "user",
//		Password: "password",
//		Database: "mydb",
//		Driver:   "mysql",
//	}
//
//	// Create connection pool
//	ctx := context.Background()
//	pool, err := ggm.NewPool(ctx, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer pool.Close()
//
//	// Execute queries safely
//	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
//		result, err := conn.Exec(ctx, "INSERT INTO users (name, age) VALUES (?, ?)", "Alice", 30)
//		return err
//	})
//
// # Transaction Support
//
// Automatic transaction management with retry:
//
//	err = pool.WithinTx(ctx, func(tx ggm.DatabaseTx) error {
//		_, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - ? WHERE id = ?", 100, fromID)
//		if err != nil {
//			return err
//		}
//		_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + ? WHERE id = ?", 100, toID)
//		return err
//	})
//
// # Performance Features
//
//   - Prepared statement caching for repeated queries
//   - Bulk insert operations for high-throughput scenarios
//   - Connection leak detection and monitoring
//   - Configurable retry policies for transient failures
//   - Slow query detection and logging
//
// # Observability
//
//   - OpenTelemetry integration for distributed tracing
//   - Prometheus-compatible metrics
//   - Structured logging with configurable levels
//   - Health check endpoints
//
// # Configuration
//
// The library supports both programmatic configuration and environment variables.
// Environment variables use the prefix YGGGO_MYSQL_* (e.g., YGGGO_MYSQL_HOST).
//
// For detailed examples, see the examples/ directory in the repository.
package ygggo_mysql

// Version returns the current library version.
//
// This version follows semantic versioning (semver) principles.
// During development, it returns "v0.0.0-dev".
func Version() string { return "v0.0.0-dev" }

