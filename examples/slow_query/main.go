package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()

	// 注意：这个示例需要Docker环境来运行MySQL容器
	// 在生产环境中，你应该使用真实的MySQL连接配置

	// 创建Docker测试助手用于演示
	helper, err := ygggo_mysql.NewDockerTestHelper(ctx)
	if err != nil {
		log.Fatal("Failed to create Docker helper (需要Docker环境):", err)
	}
	defer helper.Close()

	pool := helper.Pool()

	// Example 1: Basic slow query recording with memory storage
	fmt.Println("=== Example 1: Basic Slow Query Recording ===")
	
	// Create memory storage for slow queries
	storage := ygggo_mysql.NewMemorySlowQueryStorage(1000)
	
	// Configure slow query recording
	config := ygggo_mysql.DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 10 * time.Millisecond // Very low threshold for demo
	config.SanitizeArgs = true
	config.IncludeStack = false
	
	// Enable slow query recording
	pool.EnableSlowQueryRecording(config, storage)
	defer pool.DisableSlowQueryRecording()
	
	// Create a test table
	err = pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
		_, err := conn.Exec(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
		return err
	})
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	
	// Execute some queries that might be slow
	err = pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
		// Insert some data
		for i := 1; i <= 5; i++ {
			_, err := conn.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", 
				fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i))
			if err != nil {
				return err
			}
			// Simulate slow operation
			time.Sleep(15 * time.Millisecond)
		}
		
		// Execute a potentially slow query
		rows, err := conn.Query(ctx, "SELECT * FROM users WHERE name LIKE ?", "%User%")
		if err != nil {
			return err
		}
		defer rows.Close()
		
		// Simulate processing time
		time.Sleep(20 * time.Millisecond)
		
		return nil
	})
	if err != nil {
		log.Fatal("Failed to execute queries:", err)
	}
	
	// Wait a bit for recording to complete
	time.Sleep(100 * time.Millisecond)
	
	// Get slow query recorder
	recorder := pool.GetSlowQueryRecorder()
	if recorder == nil {
		log.Fatal("Slow query recorder not found")
	}
	
	// Retrieve and display slow query records
	records, err := recorder.GetRecords(ctx, ygggo_mysql.SlowQueryFilter{})
	if err != nil {
		log.Fatal("Failed to get records:", err)
	}
	
	fmt.Printf("Found %d slow queries:\n", len(records))
	for i, record := range records {
		fmt.Printf("  %d. Query: %s\n", i+1, record.Query)
		fmt.Printf("     Duration: %v\n", record.Duration)
		fmt.Printf("     Timestamp: %v\n", record.Timestamp.Format(time.RFC3339))
		fmt.Printf("     Normalized: %s\n", record.NormalizedQuery)
		if record.Error != "" {
			fmt.Printf("     Error: %s\n", record.Error)
		}
		fmt.Println()
	}
	
	// Example 2: Query statistics and analysis
	fmt.Println("=== Example 2: Query Statistics and Analysis ===")
	
	// Get statistics
	stats, err := recorder.GetStats(ctx)
	if err != nil {
		log.Fatal("Failed to get stats:", err)
	}
	
	fmt.Printf("Total slow queries: %d\n", stats.TotalCount)
	fmt.Printf("Unique query patterns: %d\n", stats.UniqueQueries)
	fmt.Printf("Average duration: %v\n", stats.AverageDuration)
	fmt.Printf("Max duration: %v\n", stats.MaxDuration)
	fmt.Printf("Min duration: %v\n", stats.MinDuration)
	fmt.Println()
	
	// Get query patterns
	patterns, err := recorder.GetPatterns(ctx, 5)
	if err != nil {
		log.Fatal("Failed to get patterns:", err)
	}
	
	fmt.Printf("Top %d query patterns:\n", len(patterns))
	for i, pattern := range patterns {
		fmt.Printf("  %d. Pattern: %s\n", i+1, pattern.NormalizedQuery)
		fmt.Printf("     Count: %d\n", pattern.Count)
		fmt.Printf("     Average duration: %v\n", pattern.AverageDuration)
		fmt.Printf("     Max duration: %v\n", pattern.MaxDuration)
		fmt.Println()
	}
	
	// Example 3: Advanced analysis with recommendations
	fmt.Println("=== Example 3: Advanced Analysis ===")
	
	analyzer := ygggo_mysql.NewSlowQueryAnalyzer(storage)
	report, err := analyzer.GenerateReport(ctx, ygggo_mysql.SlowQueryFilter{})
	if err != nil {
		log.Fatal("Failed to generate report:", err)
	}
	
	fmt.Printf("Analysis Report (generated at %v):\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Printf("  Total queries: %d\n", report.Summary.TotalQueries)
	fmt.Printf("  Unique patterns: %d\n", report.Summary.UniquePatterns)
	fmt.Printf("  Average duration: %v\n", report.Summary.AverageDuration)
	fmt.Printf("  Median duration: %v\n", report.Summary.MedianDuration)
	fmt.Printf("  95th percentile: %v\n", report.Summary.P95Duration)
	fmt.Printf("  99th percentile: %v\n", report.Summary.P99Duration)
	fmt.Printf("  Slowest query: %v\n", report.Summary.SlowestQuery)
	fmt.Println()
	
	if len(report.Recommendations) > 0 {
		fmt.Println("Recommendations:")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
		fmt.Println()
	}
	
	// Example 4: File-based storage
	fmt.Println("=== Example 4: File-based Storage ===")
	
	// Create file storage
	fileStorage, err := ygggo_mysql.NewFileSlowQueryStorage("slow_queries.log", 1024*1024) // 1MB max
	if err != nil {
		log.Fatal("Failed to create file storage:", err)
	}
	defer fileStorage.Close()
	
	// Create a new recorder with file storage
	fileConfig := ygggo_mysql.DefaultSlowQueryConfig()
	fileConfig.Enabled = true
	fileConfig.Threshold = 5 * time.Millisecond
	
	fileRecorder := ygggo_mysql.NewSlowQueryRecorder(fileConfig, fileStorage)
	defer fileRecorder.Close()
	
	// Record some queries
	err = fileRecorder.Record(ctx, "SELECT COUNT(*) FROM users", nil, 10*time.Millisecond, nil)
	if err != nil {
		log.Fatal("Failed to record query:", err)
	}
	
	err = fileRecorder.Record(ctx, "SELECT * FROM users ORDER BY name", nil, 25*time.Millisecond, nil)
	if err != nil {
		log.Fatal("Failed to record query:", err)
	}
	
	// Retrieve records from file storage
	fileRecords, err := fileRecorder.GetRecords(ctx, ygggo_mysql.SlowQueryFilter{})
	if err != nil {
		log.Fatal("Failed to get file records:", err)
	}
	
	fmt.Printf("File storage contains %d slow queries\n", len(fileRecords))
	
	// Example 5: Configuration management
	fmt.Println("=== Example 5: Configuration Management ===")
	
	fmt.Printf("Current threshold: %v\n", recorder.GetThreshold())
	fmt.Printf("Recording enabled: %v\n", recorder.IsEnabled())
	
	// Change configuration
	recorder.SetThreshold(50 * time.Millisecond)
	fmt.Printf("New threshold: %v\n", recorder.GetThreshold())
	
	// Update entire configuration
	newConfig := recorder.GetConfig()
	newConfig.SanitizeArgs = false
	newConfig.IncludeStack = true
	recorder.UpdateConfig(newConfig)
	
	fmt.Printf("Updated config - SanitizeArgs: %v, IncludeStack: %v\n", 
		newConfig.SanitizeArgs, newConfig.IncludeStack)
	
	// Example 6: Filtering queries
	fmt.Println("=== Example 6: Filtering Queries ===")
	
	// Filter by duration
	minDuration := 15 * time.Millisecond
	filteredRecords, err := recorder.GetRecords(ctx, ygggo_mysql.SlowQueryFilter{
		MinDuration: &minDuration,
		Limit:       3,
	})
	if err != nil {
		log.Fatal("Failed to get filtered records:", err)
	}
	
	fmt.Printf("Found %d queries with duration >= %v:\n", len(filteredRecords), minDuration)
	for i, record := range filteredRecords {
		fmt.Printf("  %d. %s (duration: %v)\n", i+1, record.Query, record.Duration)
	}
	
	// Clear all records
	fmt.Println("\n=== Clearing Records ===")
	err = recorder.Clear(ctx)
	if err != nil {
		log.Fatal("Failed to clear records:", err)
	}
	
	finalRecords, err := recorder.GetRecords(ctx, ygggo_mysql.SlowQueryFilter{})
	if err != nil {
		log.Fatal("Failed to get final records:", err)
	}
	
	fmt.Printf("Records after clearing: %d\n", len(finalRecords))
	
	fmt.Println("\n=== Slow Query Recording Demo Complete ===")
}
