package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()

	// Setup structured logging with JSON format
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create pool with mock
	pool, mock, err := ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{}, true)
	if err != nil {
		logger.Error("failed to create pool", "error", err)
		return
	}
	defer pool.Close()

	// Enable structured logging
	pool.EnableLogging(true)
	pool.SetLogger(logger)
	pool.SetSlowQueryThreshold(100 * time.Millisecond) // Queries > 100ms are considered slow

	logger.Info("ygggo_mysql structured logging example started")

	// Setup mock expectations
	if mock != nil {
		// Fast query
		rows := ygggo_mysql.NewRows([]string{"id", "name"})
		rows = ygggo_mysql.AddRow(rows, 1, "Alice")
		rows = ygggo_mysql.AddRow(rows, 2, "Bob")
		mock.ExpectQuery(`SELECT id, name FROM users WHERE active = \?`).WithArgs(true).WillReturnRows(rows)
		
		// Slow query (will be logged as warning)
		slowRows := ygggo_mysql.NewRows([]string{"count"})
		slowRows = ygggo_mysql.AddRow(slowRows, 1000)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM large_table`).WillReturnRows(slowRows)
		
		// Insert operation
		mock.ExpectExec(`INSERT INTO users \(name, email\) VALUES \(\?, \?\)`).
			WithArgs("Charlie", "charlie@example.com").
			WillReturnResult(ygggo_mysql.NewResult(3, 1))
		
		// Transaction
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE users SET last_login = NOW\(\) WHERE id = \?`).WithArgs(1).
			WillReturnResult(ygggo_mysql.NewResult(0, 1))
		mock.ExpectCommit()
	}

	logger.Info("executing database operations with structured logging")

	// Use direct connection to avoid WithConn deadlock issues
	conn, err := pool.Acquire(ctx)
	if err != nil {
		logger.Error("failed to acquire connection", "error", err)
		return
	}

	// 1. Fast query - will be logged at INFO level
	logger.Info("executing fast query")
	rs, err := conn.Query(ctx, "SELECT id, name FROM users WHERE active = ?", true)
	if err != nil {
		logger.Error("fast query failed", "error", err)
		return
	}
	defer rs.Close()

	count := 0
	for rs.Next() { count++ }
	logger.Info("fast query completed", "rows_returned", count)

	// 2. Slow query - will be logged at WARN level due to threshold
	logger.Info("executing slow query")
	rs2, err := conn.Query(ctx, "SELECT COUNT(*) FROM large_table")
	if err != nil {
		logger.Error("slow query failed", "error", err)
		return
	}
	defer rs2.Close()

	var totalCount int
	if rs2.Next() {
		rs2.Scan(&totalCount)
	}
	logger.Info("slow query completed", "total_count", totalCount)

	// 3. Insert operation
	logger.Info("executing insert operation")
	result, err := conn.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Charlie", "charlie@example.com")
	if err != nil {
		logger.Error("insert failed", "error", err)
		return
	}
	
	insertID, _ := result.LastInsertId()
	affected, _ := result.RowsAffected()
	logger.Info("insert completed", "insert_id", insertID, "rows_affected", affected)

	err = conn.Close()
	if err != nil {
		logger.Error("failed to close connection", "error", err)
		return
	}

	// 4. Transaction with logging
	logger.Info("executing transaction")
	err = pool.WithinTx(ctx, func(tx ygggo_mysql.DatabaseTx) error {
		_, err := tx.Exec(ctx, "UPDATE users SET last_login = NOW() WHERE id = ?", 1)
		return err
	})
	if err != nil {
		logger.Error("transaction failed", "error", err)
		return
	}
	logger.Info("transaction completed successfully")

	// 5. Log connection pool statistics
	stats := pool.GetPoolStats()
	logger.Info("connection pool statistics",
		"active_connections", stats.ActiveConnections,
		"idle_connections", stats.IdleConnections,
		"total_connections", stats.TotalConnections,
		"max_open", stats.MaxOpen,
		"max_idle", stats.MaxIdle,
	)

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			logger.Error("mock expectations not met", "error", err)
			return
		}
	}

	logger.Info("ygggo_mysql structured logging example completed successfully")
	logger.Info("check the JSON logs above for detailed operation information")
}
