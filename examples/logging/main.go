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

	// 从环境变量获取数据库配置，或使用默认值
	config := ygggo_mysql.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     3306,
		Database: getEnv("DB_NAME", "test"),
		Username: getEnv("DB_USER", "root"),
		Password: getEnv("DB_PASSWORD", "password"),
	}

	// 创建连接池
	pool, err := ygggo_mysql.NewPool(ctx, config)
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

	// 设置测试数据
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// 创建测试表
		_, err := c.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100),
			active BOOLEAN DEFAULT TRUE,
			last_login TIMESTAMP NULL
		)`)
		if err != nil { return err }

		// 清理并插入测试数据
		_, err = c.Exec(ctx, "DELETE FROM users")
		if err != nil { return err }

		_, err = c.Exec(ctx, "INSERT INTO users (name, email, active) VALUES (?, ?, ?)", "Alice", "alice@example.com", true)
		if err != nil { return err }

		_, err = c.Exec(ctx, "INSERT INTO users (name, email, active) VALUES (?, ?, ?)", "Bob", "bob@example.com", true)
		if err != nil { return err }

		return nil
	})
	if err != nil {
		logger.Error("failed to setup test data", "error", err)
		return
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

	// 2. 模拟慢查询 - 使用SLEEP函数
	logger.Info("executing slow query")
	rs2, err := conn.Query(ctx, "SELECT SLEEP(0.2), COUNT(*) FROM users")
	if err != nil {
		logger.Error("slow query failed", "error", err)
		return
	}
	defer rs2.Close()

	var sleepResult, totalCount int
	if rs2.Next() {
		rs2.Scan(&sleepResult, &totalCount)
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

	logger.Info("ygggo_mysql structured logging example completed successfully", "version", ygggo_mysql.Version())
	logger.Info("check the JSON logs above for detailed operation information")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
