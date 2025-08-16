package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()

	// Create a SQLite test helper for demonstration
	helper, err := ygggo_mysql.NewSQLiteTestHelper(ctx)
	if err != nil {
		log.Fatal("Failed to create SQLite helper:", err)
	}
	defer helper.Close()

	pool := helper.Pool()

	fmt.Println("=== Database Performance Benchmark Demo ===")
	fmt.Println()

	// Example 1: Basic benchmark configuration
	fmt.Println("1. Basic Benchmark Configuration")
	config := ygggo_mysql.DefaultBenchmarkConfig()
	config.Duration = 5 * time.Second
	config.Concurrency = 4
	config.WarmupTime = 1 * time.Second
	config.ReportInterval = 2 * time.Second

	fmt.Printf("   Duration: %v\n", config.Duration)
	fmt.Printf("   Concurrency: %d\n", config.Concurrency)
	fmt.Printf("   Warmup Time: %v\n", config.WarmupTime)
	fmt.Println()

	// Example 2: Simple query benchmark
	fmt.Println("2. Simple Query Benchmark")
	runner := ygggo_mysql.NewBenchmarkRunner(config, pool)
	
	// Create a simple benchmark test
	simpleTest := &SimpleBenchmarkTest{}
	
	result, err := runner.RunBenchmark(ctx, simpleTest)
	if err != nil {
		log.Fatal("Simple benchmark failed:", err)
	}
	
	printBenchmarkResult("Simple Query", result)

	// Example 3: SELECT performance with different data sizes
	fmt.Println("3. SELECT Performance Benchmark")
	selectTest := ygggo_mysql.NewSelectBenchmarkTest(500) // 500 rows
	
	result, err = runner.RunBenchmark(ctx, selectTest)
	if err != nil {
		log.Fatal("SELECT benchmark failed:", err)
	}
	
	printBenchmarkResult("SELECT Queries", result)

	// Example 4: INSERT performance benchmark
	fmt.Println("4. INSERT Performance Benchmark")
	insertTest := ygggo_mysql.NewInsertPerformanceBenchmarkTest(1) // Single inserts
	
	result, err = runner.RunBenchmark(ctx, insertTest)
	if err != nil {
		log.Fatal("INSERT benchmark failed:", err)
	}
	
	printBenchmarkResult("INSERT Operations", result)

	// Example 5: Mixed workload benchmark
	fmt.Println("5. Mixed Workload Benchmark (70% reads, 30% writes)")
	mixedTest := ygggo_mysql.NewMixedWorkloadBenchmarkTest(200, 0.7)
	
	result, err = runner.RunBenchmark(ctx, mixedTest)
	if err != nil {
		log.Fatal("Mixed workload benchmark failed:", err)
	}
	
	printBenchmarkResult("Mixed Workload", result)

	// Example 6: Benchmark suite with multiple tests
	fmt.Println("6. Benchmark Suite with Multiple Tests")
	
	suiteConfig := ygggo_mysql.DefaultBenchmarkConfig()
	suiteConfig.Duration = 3 * time.Second
	suiteConfig.Concurrency = 2
	
	suite := ygggo_mysql.NewBenchmarkSuite(suiteConfig)
	
	// Add multiple tests to the suite
	suite.AddTest(&SimpleBenchmarkTest{})
	suite.AddTest(ygggo_mysql.NewSelectBenchmarkTest(100))
	suite.AddTest(ygggo_mysql.NewInsertPerformanceBenchmarkTest(1))
	suite.AddTest(ygggo_mysql.NewUpdateBenchmarkTest(50))
	
	results, err := suite.RunAll(ctx, pool)
	if err != nil {
		log.Fatal("Benchmark suite failed:", err)
	}
	
	fmt.Printf("   Suite completed with %d tests\n", len(results))
	for i, result := range results {
		fmt.Printf("   Test %d: %s - %.2f ops/sec\n", 
			i+1, result.TestName, result.ThroughputOPS)
	}
	fmt.Println()

	// Example 7: Generate comprehensive report
	fmt.Println("7. Comprehensive Performance Report")
	
	generator := ygggo_mysql.NewBenchmarkReportGenerator()
	generator.AddResults(results)
	
	// Generate text report
	fmt.Println("   Text Report:")
	fmt.Println("   " + strings.Repeat("-", 50))
	err = generator.WriteTextReport(os.Stdout)
	if err != nil {
		log.Fatal("Failed to generate text report:", err)
	}
	
	// Generate JSON report to file
	jsonFile, err := os.Create("benchmark_report.json")
	if err != nil {
		log.Fatal("Failed to create JSON report file:", err)
	}
	defer jsonFile.Close()
	
	err = generator.WriteJSONReport(jsonFile)
	if err != nil {
		log.Fatal("Failed to generate JSON report:", err)
	}
	fmt.Println("   JSON report saved to: benchmark_report.json")
	
	// Generate CSV report to file
	csvFile, err := os.Create("benchmark_report.csv")
	if err != nil {
		log.Fatal("Failed to create CSV report file:", err)
	}
	defer csvFile.Close()
	
	err = generator.WriteCSVReport(csvFile)
	if err != nil {
		log.Fatal("Failed to generate CSV report:", err)
	}
	fmt.Println("   CSV report saved to: benchmark_report.csv")
	fmt.Println()

	// Example 8: Performance analysis
	fmt.Println("8. Performance Analysis")
	
	topPerformers := generator.GetTopPerformers(3)
	fmt.Printf("   Top %d performers:\n", len(topPerformers))
	for i, result := range topPerformers {
		fmt.Printf("   %d. %s: %.2f ops/sec (avg latency: %v)\n", 
			i+1, result.TestName, result.ThroughputOPS, result.AvgLatency)
	}
	fmt.Println()

	// Example 9: Custom benchmark test
	fmt.Println("9. Custom Benchmark Test")
	
	customTest := &CustomBenchmarkTest{
		QueryComplexity: "medium",
		DataSize:        100,
	}
	
	result, err = runner.RunBenchmark(ctx, customTest)
	if err != nil {
		log.Fatal("Custom benchmark failed:", err)
	}
	
	printBenchmarkResult("Custom Test", result)

	fmt.Println("=== Benchmark Demo Complete ===")
	fmt.Println()
	fmt.Println("Key Takeaways:")
	fmt.Println("- Benchmark tests help identify performance bottlenecks")
	fmt.Println("- Different workloads have different performance characteristics")
	fmt.Println("- Concurrency level affects both throughput and latency")
	fmt.Println("- Regular benchmarking helps track performance over time")
}

// printBenchmarkResult prints a formatted benchmark result
func printBenchmarkResult(testName string, result *ygggo_mysql.BenchmarkResult) {
	fmt.Printf("   %s Results:\n", testName)
	fmt.Printf("   - Duration: %v\n", result.Duration)
	fmt.Printf("   - Total Operations: %d\n", result.TotalOps)
	fmt.Printf("   - Success Rate: %.2f%%\n", 
		float64(result.SuccessOps)/float64(result.TotalOps)*100)
	fmt.Printf("   - Throughput: %.2f ops/sec\n", result.ThroughputOPS)
	fmt.Printf("   - Average Latency: %v\n", result.AvgLatency)
	fmt.Printf("   - P95 Latency: %v\n", result.P95Latency)
	fmt.Printf("   - P99 Latency: %v\n", result.P99Latency)
	
	if result.ErrorOps > 0 {
		fmt.Printf("   - Errors: %d\n", result.ErrorOps)
	}
	fmt.Println()
}

// SimpleBenchmarkTest performs simple SELECT 1 operations
type SimpleBenchmarkTest struct{}

func (t *SimpleBenchmarkTest) Name() string {
	return "Simple Query Test"
}

func (t *SimpleBenchmarkTest) Setup(ctx context.Context, pool ygggo_mysql.DatabasePool) error {
	return nil
}

func (t *SimpleBenchmarkTest) Run(ctx context.Context, pool ygggo_mysql.DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
		rows, err := conn.Query(ctx, "SELECT 1")
		if err != nil {
			return err
		}
		defer rows.Close()
		
		for rows.Next() {
			var result int
			if err := rows.Scan(&result); err != nil {
				return err
			}
		}
		return rows.Err()
	})
}

func (t *SimpleBenchmarkTest) Cleanup(ctx context.Context, pool ygggo_mysql.DatabasePool) error {
	return nil
}

// CustomBenchmarkTest demonstrates a custom benchmark implementation
type CustomBenchmarkTest struct {
	QueryComplexity string
	DataSize        int
}

func (t *CustomBenchmarkTest) Name() string {
	return fmt.Sprintf("Custom Test (%s complexity, %d data size)", 
		t.QueryComplexity, t.DataSize)
}

func (t *CustomBenchmarkTest) Setup(ctx context.Context, pool ygggo_mysql.DatabasePool) error {
	return pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
		_, err := conn.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS custom_test (
				id INTEGER PRIMARY KEY,
				data TEXT,
				value INTEGER
			)
		`)
		if err != nil {
			return err
		}
		
		// Insert test data
		for i := 0; i < t.DataSize; i++ {
			_, err := conn.Exec(ctx, 
				"INSERT INTO custom_test (data, value) VALUES (?, ?)",
				fmt.Sprintf("test_data_%d", i), i)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (t *CustomBenchmarkTest) Run(ctx context.Context, pool ygggo_mysql.DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
		switch t.QueryComplexity {
		case "simple":
			rows, err := conn.Query(ctx, "SELECT COUNT(*) FROM custom_test")
			if err != nil {
				return err
			}
			defer rows.Close()
			
		case "medium":
			rows, err := conn.Query(ctx, 
				"SELECT id, data FROM custom_test WHERE value < ? LIMIT 10", 
				workerID*10+50)
			if err != nil {
				return err
			}
			defer rows.Close()
			
		case "complex":
			rows, err := conn.Query(ctx, 
				"SELECT id, data, value FROM custom_test WHERE value BETWEEN ? AND ? ORDER BY value LIMIT 5",
				workerID*5, workerID*5+20)
			if err != nil {
				return err
			}
			defer rows.Close()
		}
		
		return nil
	})
}

func (t *CustomBenchmarkTest) Cleanup(ctx context.Context, pool ygggo_mysql.DatabasePool) error {
	return pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
		_, err := conn.Exec(ctx, "DROP TABLE IF EXISTS custom_test")
		return err
	})
}
