package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
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

	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("ygggo_mysql SQLite example started")

	// Create SQLite pool (in-memory database)
	pool, err := ygggo_mysql.NewSQLiteTestPool(ctx)
	if err != nil {
		logger.Error("failed to create SQLite pool", "error", err)
		return
	}
	defer pool.Close()

	// Enable all observability features
	pool.EnableTelemetry(true)
	pool.EnableMetrics(true)
	pool.SetMeterProvider(mp)
	pool.EnableLogging(true)
	pool.SetLogger(logger)

	logger.Info("created SQLite pool with observability enabled")

	// Create schema
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		_, err := c.Exec(ctx, `
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				email TEXT UNIQUE NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil { return err }

		_, err = c.Exec(ctx, `
			CREATE TABLE posts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				title TEXT NOT NULL,
				content TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (user_id) REFERENCES users (id)
			)
		`)
		return err
	})
	if err != nil {
		logger.Error("failed to create schema", "error", err)
		return
	}

	logger.Info("created database schema")

	// Insert test data
	var userIDs []int64
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		users := []struct{ name, email string }{
			{"Alice", "alice@example.com"},
			{"Bob", "bob@example.com"},
			{"Charlie", "charlie@example.com"},
		}

		for _, user := range users {
			result, err := c.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", user.name, user.email)
			if err != nil { return err }
			
			id, err := result.LastInsertId()
			if err != nil { return err }
			userIDs = append(userIDs, id)
		}
		return nil
	})
	if err != nil {
		logger.Error("failed to insert users", "error", err)
		return
	}

	logger.Info("inserted test users", "count", len(userIDs))

	// Insert posts using transaction
	err = pool.WithinTx(ctx, func(tx ygggo_mysql.DatabaseTx) error {
		posts := []struct{ userID int64; title, content string }{
			{userIDs[0], "Hello World", "This is my first post!"},
			{userIDs[1], "SQLite is Great", "No CGO dependencies needed."},
			{userIDs[2], "Observability", "Tracing, metrics, and logging all work!"},
		}

		for _, post := range posts {
			_, err := tx.Exec(ctx, "INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)", 
				post.userID, post.title, post.content)
			if err != nil { return err }
		}
		return nil
	})
	if err != nil {
		logger.Error("failed to insert posts", "error", err)
		return
	}

	logger.Info("inserted test posts in transaction")

	// Query data with joins
	type UserPost struct {
		UserName string
		Email    string
		Title    string
		Content  string
	}

	var userPosts []UserPost
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		rs, err := c.Query(ctx, `
			SELECT u.name, u.email, p.title, p.content
			FROM users u
			JOIN posts p ON u.id = p.user_id
			ORDER BY u.name, p.title
		`)
		if err != nil { return err }
		defer rs.Close()

		for rs.Next() {
			var up UserPost
			err := rs.Scan(&up.UserName, &up.Email, &up.Title, &up.Content)
			if err != nil { return err }
			userPosts = append(userPosts, up)
		}
		return rs.Err()
	})
	if err != nil {
		logger.Error("failed to query user posts", "error", err)
		return
	}

	logger.Info("queried user posts", "count", len(userPosts))

	// Display results
	fmt.Println("\n=== User Posts ===")
	for _, up := range userPosts {
		fmt.Printf("👤 %s (%s)\n", up.UserName, up.Email)
		fmt.Printf("📝 %s: %s\n\n", up.Title, up.Content)
	}

	// Collect and display metrics
	rm := metricdata.ResourceMetrics{}
	err = metricsReader.Collect(ctx, &rm)
	if err != nil {
		logger.Error("failed to collect metrics", "error", err)
		return
	}

	fmt.Println("=== Metrics Summary ===")
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

	// Display connection pool stats
	stats := pool.GetPoolStats()
	fmt.Println("\n=== Connection Pool Stats ===")
	fmt.Printf("Active: %d, Idle: %d, Total: %d\n", 
		stats.ActiveConnections, stats.IdleConnections, stats.TotalConnections)
	fmt.Printf("Max Open: %d, Max Idle: %d\n", stats.MaxOpen, stats.MaxIdle)

	logger.Info("ygggo_mysql SQLite example completed successfully")
	fmt.Println("\n🎉 SQLite integration working perfectly!")
	fmt.Println("✅ No CGO dependencies")
	fmt.Println("✅ No deadlock issues")
	fmt.Println("✅ Full observability support")
	fmt.Println("✅ OpenTelemetry traces shown above")
	fmt.Println("✅ Metrics and logging working")
}
