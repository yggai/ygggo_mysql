package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yggai/ygggo_mysql"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func main() {
	ctx := context.Background()

	// Setup OpenTelemetry Metrics
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))

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
		log.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	// Enable metrics
	pool.EnableMetrics(true)
	pool.SetMeterProvider(provider)

	// 设置测试数据
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// 创建测试表
		_, err := c.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100)
		)`)
		if err != nil { return err }

		// 清理并插入测试数据
		_, err = c.Exec(ctx, "DELETE FROM users")
		if err != nil { return err }

		_, err = c.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
		if err != nil { return err }

		_, err = c.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Bob")
		if err != nil { return err }

		return nil
	})
	if err != nil { log.Fatalf("Setup: %v", err) }

	// Use direct connection to avoid WithConn issues for now
	conn, err := pool.Acquire(ctx)
	if err != nil { log.Fatalf("Acquire: %v", err) }

	// Execute query
	rs, err := conn.Query(ctx, "SELECT id, name FROM users")
	if err != nil { log.Fatalf("Query: %v", err) }
	defer rs.Close()

	count := 0
	for rs.Next() { count++ }
	fmt.Printf("Query returned %d rows\n", count)

	// Execute insert
	result, err := conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Charlie")
	if err != nil { log.Fatalf("Exec: %v", err) }

	affected, _ := result.RowsAffected()
	fmt.Printf("Insert affected %d rows\n", affected)

	err = conn.Close()
	if err != nil { log.Fatalf("Close: %v", err) }

	// Collect and display metrics
	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(ctx, &rm)
	if err != nil { log.Fatalf("Collect: %v", err) }

	fmt.Println("\n=== Metrics ===")
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			fmt.Printf("Metric: %s\n", m.Name)
			fmt.Printf("  Description: %s\n", m.Description)
			
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					fmt.Printf("  Value: %d, Attributes: %v\n", dp.Value, dp.Attributes)
				}
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					fmt.Printf("  Count: %d, Sum: %f, Attributes: %v\n", dp.Count, dp.Sum, dp.Attributes)
				}
			}
			fmt.Println()
		}
	}

	fmt.Println("ygggo_mysql example: metrics integration", ygggo_mysql.Version())
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
