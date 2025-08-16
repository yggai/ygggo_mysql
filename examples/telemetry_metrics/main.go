package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yggai/ygggo_mysql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	// Setup OpenTelemetry Tracing
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil { log.Fatalf("failed to create trace exporter: %v", err) }

	tp := trace.NewTracerProvider(
		trace.WithSyncer(traceExporter),
	)
	otel.SetTracerProvider(tp)

	// Setup OpenTelemetry Metrics
	metricsReader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(metricsReader))

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

	// Enable both telemetry and metrics
	pool.EnableTelemetry(true)
	pool.EnableMetrics(true)
	pool.SetMeterProvider(mp)

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

		_, err = c.Exec(ctx, "INSERT INTO users (name) VALUES ('Alice'), ('Bob')")
		if err != nil { return err }

		return nil
	})
	if err != nil { log.Fatalf("Setup: %v", err) }

	fmt.Println("=== Executing Database Operations ===")

	// Use direct connection to avoid WithConn deadlock issues
	conn, err := pool.Acquire(ctx)
	if err != nil { log.Fatalf("Acquire: %v", err) }

	// Execute query (will create both trace span and metrics)
	rs, err := conn.Query(ctx, "SELECT id, name FROM users")
	if err != nil { log.Fatalf("Query: %v", err) }
	defer rs.Close()

	count := 0
	for rs.Next() { count++ }
	fmt.Printf("Query returned %d rows\n", count)

	// Execute insert (will create both trace span and metrics)
	result, err := conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Charlie")
	if err != nil { log.Fatalf("Exec: %v", err) }
	
	affected, _ := result.RowsAffected()
	fmt.Printf("Insert affected %d rows\n", affected)

	err = conn.Close()
	if err != nil { log.Fatalf("Close: %v", err) }

	// Collect and display metrics
	rm := metricdata.ResourceMetrics{}
	err = metricsReader.Collect(ctx, &rm)
	if err != nil { log.Fatalf("Collect: %v", err) }

	fmt.Println("\n=== Metrics Summary ===")
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			fmt.Printf("📊 %s: %s\n", m.Name, m.Description)
			
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					attrs := ""
					for _, attr := range dp.Attributes.ToSlice() {
						if attrs != "" { attrs += ", " }
						attrs += fmt.Sprintf("%s=%s", attr.Key, attr.Value.AsString())
					}
					if attrs != "" {
						fmt.Printf("   Value: %d [%s]\n", dp.Value, attrs)
					} else {
						fmt.Printf("   Value: %d\n", dp.Value)
					}
				}
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					attrs := ""
					for _, attr := range dp.Attributes.ToSlice() {
						if attrs != "" { attrs += ", " }
						attrs += fmt.Sprintf("%s=%s", attr.Key, attr.Value.AsString())
					}
					if attrs != "" {
						fmt.Printf("   Count: %d, Avg: %.3fms [%s]\n", dp.Count, dp.Sum*1000/float64(dp.Count), attrs)
					} else {
						fmt.Printf("   Count: %d, Avg: %.3fms\n", dp.Count, dp.Sum*1000/float64(dp.Count))
					}
				}
			}
		}
	}

	fmt.Printf("\n🎉 ygggo_mysql: Combined telemetry & metrics integration working! %s\n", ygggo_mysql.Version())
	fmt.Println("✅ OpenTelemetry traces are shown above")
	fmt.Println("✅ Metrics summary is shown above")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
