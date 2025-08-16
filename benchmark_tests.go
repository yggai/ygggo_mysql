package ygggo_mysql

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// SelectBenchmarkTest tests SELECT query performance
type SelectBenchmarkTest struct {
	TableName string
	DataSize  int
}

func NewSelectBenchmarkTest(dataSize int) *SelectBenchmarkTest {
	return &SelectBenchmarkTest{
		TableName: "benchmark_select_test",
		DataSize:  dataSize,
	}
}

func (t *SelectBenchmarkTest) Name() string {
	return fmt.Sprintf("SELECT Benchmark (%d rows)", t.DataSize)
}

func (t *SelectBenchmarkTest) Setup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Create table (MySQL syntax)
		_, err := conn.Exec(ctx, fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				email VARCHAR(255),
				age INT,
				score DECIMAL(10,2),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`, t.TableName))
		if err != nil {
			return err
		}
		
		// Insert test data
		for i := 0; i < t.DataSize; i++ {
			_, err := conn.Exec(ctx, 
				fmt.Sprintf("INSERT INTO %s (name, email, age, score) VALUES (?, ?, ?, ?)", t.TableName),
				fmt.Sprintf("user_%d", i),
				fmt.Sprintf("user_%d@example.com", i),
				20+rand.Intn(60),
				rand.Float64()*100)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (t *SelectBenchmarkTest) Run(ctx context.Context, pool DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Random queries to avoid caching effects
		queryType := rand.Intn(4)
		
		switch queryType {
		case 0:
			// Simple point query
			id := rand.Intn(t.DataSize) + 1
			rows, err := conn.Query(ctx, 
				fmt.Sprintf("SELECT id, name, email FROM %s WHERE id = ?", t.TableName), id)
			if err != nil {
				return err
			}
			defer rows.Close()
			
		case 1:
			// Range query
			minAge := 20 + rand.Intn(40)
			rows, err := conn.Query(ctx, 
				fmt.Sprintf("SELECT id, name, age FROM %s WHERE age >= ? LIMIT 10", t.TableName), minAge)
			if err != nil {
				return err
			}
			defer rows.Close()
			
		case 2:
			// Aggregation query
			rows, err := conn.Query(ctx, 
				fmt.Sprintf("SELECT COUNT(*), AVG(age), MAX(score) FROM %s", t.TableName))
			if err != nil {
				return err
			}
			defer rows.Close()
			
		case 3:
			// Pattern matching
			pattern := fmt.Sprintf("user_%d%%", rand.Intn(100))
			rows, err := conn.Query(ctx, 
				fmt.Sprintf("SELECT id, name FROM %s WHERE name LIKE ? LIMIT 5", t.TableName), pattern)
			if err != nil {
				return err
			}
			defer rows.Close()
		}
		
		return nil
	})
}

func (t *SelectBenchmarkTest) Cleanup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))
		return err
	})
}

// InsertBenchmarkTest tests INSERT performance
type InsertPerformanceBenchmarkTest struct {
	TableName string
	BatchSize int
}

func NewInsertPerformanceBenchmarkTest(batchSize int) *InsertPerformanceBenchmarkTest {
	return &InsertPerformanceBenchmarkTest{
		TableName: "benchmark_insert_perf_test",
		BatchSize: batchSize,
	}
}

func (t *InsertPerformanceBenchmarkTest) Name() string {
	return fmt.Sprintf("INSERT Performance Benchmark (batch size: %d)", t.BatchSize)
}

func (t *InsertPerformanceBenchmarkTest) Setup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INT AUTO_INCREMENT PRIMARY KEY,
				data TEXT,
				value INT,
				timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`, t.TableName))
		return err
	})
}

func (t *InsertPerformanceBenchmarkTest) Run(ctx context.Context, pool DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		if t.BatchSize <= 1 {
			// Single insert
			_, err := conn.Exec(ctx, 
				fmt.Sprintf("INSERT INTO %s (data, value) VALUES (?, ?)", t.TableName),
				fmt.Sprintf("data_%d_%d", workerID, time.Now().UnixNano()),
				rand.Intn(1000))
			return err
		} else {
			// Batch insert using transaction
			return pool.WithinTx(ctx, func(tx DatabaseTx) error {
				for i := 0; i < t.BatchSize; i++ {
					_, err := tx.Exec(ctx, 
						fmt.Sprintf("INSERT INTO %s (data, value) VALUES (?, ?)", t.TableName),
						fmt.Sprintf("data_%d_%d_%d", workerID, time.Now().UnixNano(), i),
						rand.Intn(1000))
					if err != nil {
						return err
					}
				}
				return nil
			})
		}
	})
}

func (t *InsertPerformanceBenchmarkTest) Cleanup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))
		return err
	})
}

// UpdateBenchmarkTest tests UPDATE performance
type UpdateBenchmarkTest struct {
	TableName string
	DataSize  int
}

func NewUpdateBenchmarkTest(dataSize int) *UpdateBenchmarkTest {
	return &UpdateBenchmarkTest{
		TableName: "benchmark_update_test",
		DataSize:  dataSize,
	}
}

func (t *UpdateBenchmarkTest) Name() string {
	return fmt.Sprintf("UPDATE Benchmark (%d rows)", t.DataSize)
}

func (t *UpdateBenchmarkTest) Setup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Drop table first to ensure clean state
		_, _ = conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))

		// Create table (MySQL syntax)
		_, err := conn.Exec(ctx, fmt.Sprintf(`
			CREATE TABLE %s (
				id INT AUTO_INCREMENT PRIMARY KEY,
				counter INT DEFAULT 0,
				last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`, t.TableName))
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", t.TableName, err)
		}

		// Insert initial data
		for i := 0; i < t.DataSize; i++ {
			_, err := conn.Exec(ctx,
				fmt.Sprintf("INSERT INTO %s (counter) VALUES (?)", t.TableName), 0)
			if err != nil {
				return fmt.Errorf("failed to insert initial data: %w", err)
			}
		}

		// Verify table and data
		rows, err := conn.Query(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", t.TableName))
		if err != nil {
			return fmt.Errorf("failed to verify data: %w", err)
		}
		defer rows.Close()

		if rows.Next() {
			var count int
			if err := rows.Scan(&count); err != nil {
				return fmt.Errorf("failed to scan count: %w", err)
			}
			if count != t.DataSize {
				return fmt.Errorf("expected %d rows, got %d", t.DataSize, count)
			}
		}

		return nil
	})
}

func (t *UpdateBenchmarkTest) Run(ctx context.Context, pool DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Random update
		id := rand.Intn(t.DataSize) + 1
		_, err := conn.Exec(ctx, 
			fmt.Sprintf("UPDATE %s SET counter = counter + 1, last_updated = CURRENT_TIMESTAMP WHERE id = ?", t.TableName),
			id)
		return err
	})
}

func (t *UpdateBenchmarkTest) Cleanup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))
		return err
	})
}

// BulkOperationBenchmarkTest tests bulk operations
type BulkOperationBenchmarkTest struct {
	TableName string
	BatchSize int
}

func NewBulkOperationBenchmarkTest(batchSize int) *BulkOperationBenchmarkTest {
	return &BulkOperationBenchmarkTest{
		TableName: "benchmark_bulk_test",
		BatchSize: batchSize,
	}
}

func (t *BulkOperationBenchmarkTest) Name() string {
	return fmt.Sprintf("Bulk Operation Benchmark (batch size: %d)", t.BatchSize)
}

func (t *BulkOperationBenchmarkTest) Setup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Drop table first to ensure clean state
		_, _ = conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))

		// Create table (MySQL syntax)
		_, err := conn.Exec(ctx, fmt.Sprintf(`
			CREATE TABLE %s (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255),
				value INT
			)
		`, t.TableName))
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", t.TableName, err)
		}

		// Verify table exists (MySQL syntax)
		rows, err := conn.Query(ctx, fmt.Sprintf("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s'", t.TableName))
		if err != nil {
			return fmt.Errorf("failed to verify table creation: %w", err)
		}
		defer rows.Close()

		if !rows.Next() {
			return fmt.Errorf("table %s was not created", t.TableName)
		}

		return nil
	})
}

func (t *BulkOperationBenchmarkTest) Run(ctx context.Context, pool DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Verify table exists before attempting bulk insert
		rows, err := conn.Query(ctx, fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", t.TableName))
		if err != nil {
			return fmt.Errorf("failed to check table existence: %w", err)
		}
		defer rows.Close()

		if !rows.Next() {
			return fmt.Errorf("table %s does not exist", t.TableName)
		}

		// Prepare bulk data
		columns := []string{"name", "value"}
		bulkRows := make([][]any, t.BatchSize)

		for i := 0; i < t.BatchSize; i++ {
			bulkRows[i] = []any{
				fmt.Sprintf("bulk_name_%d_%d", workerID, i),
				rand.Intn(1000),
			}
		}

		// Use BulkInsert
		_, err = conn.BulkInsert(ctx, t.TableName, columns, bulkRows)
		if err != nil {
			return fmt.Errorf("bulk insert failed: %w", err)
		}
		return nil
	})
}

func (t *BulkOperationBenchmarkTest) Cleanup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))
		return err
	})
}

// MixedWorkloadBenchmarkTest tests mixed read/write workload
type MixedWorkloadBenchmarkTest struct {
	TableName   string
	DataSize    int
	ReadRatio   float64 // 0.0 = all writes, 1.0 = all reads
}

func NewMixedWorkloadBenchmarkTest(dataSize int, readRatio float64) *MixedWorkloadBenchmarkTest {
	return &MixedWorkloadBenchmarkTest{
		TableName: "benchmark_mixed_test",
		DataSize:  dataSize,
		ReadRatio: readRatio,
	}
}

func (t *MixedWorkloadBenchmarkTest) Name() string {
	return fmt.Sprintf("Mixed Workload Benchmark (%d rows, %.0f%% reads)", t.DataSize, t.ReadRatio*100)
}

func (t *MixedWorkloadBenchmarkTest) Setup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Drop table first to ensure clean state
		_, _ = conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))

		// Create table (MySQL syntax)
		_, err := conn.Exec(ctx, fmt.Sprintf(`
			CREATE TABLE %s (
				id INT AUTO_INCREMENT PRIMARY KEY,
				data TEXT,
				counter INT DEFAULT 0
			)
		`, t.TableName))
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", t.TableName, err)
		}

		// Insert initial data
		for i := 0; i < t.DataSize; i++ {
			_, err := conn.Exec(ctx,
				fmt.Sprintf("INSERT INTO %s (data, counter) VALUES (?, ?)", t.TableName),
				fmt.Sprintf("initial_data_%d", i), 0)
			if err != nil {
				return fmt.Errorf("failed to insert initial data: %w", err)
			}
		}

		// Verify table and data
		rows, err := conn.Query(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", t.TableName))
		if err != nil {
			return fmt.Errorf("failed to verify data: %w", err)
		}
		defer rows.Close()

		if rows.Next() {
			var count int
			if err := rows.Scan(&count); err != nil {
				return fmt.Errorf("failed to scan count: %w", err)
			}
			if count != t.DataSize {
				return fmt.Errorf("expected %d rows, got %d", t.DataSize, count)
			}
		}

		return nil
	})
}

func (t *MixedWorkloadBenchmarkTest) Run(ctx context.Context, pool DatabasePool, workerID int) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		if rand.Float64() < t.ReadRatio {
			// Read operation
			id := rand.Intn(t.DataSize) + 1
			rows, err := conn.Query(ctx, 
				fmt.Sprintf("SELECT id, data, counter FROM %s WHERE id = ?", t.TableName), id)
			if err != nil {
				return err
			}
			defer rows.Close()
			
			for rows.Next() {
				var id int
				var data string
				var counter int
				if err := rows.Scan(&id, &data, &counter); err != nil {
					return err
				}
			}
			return rows.Err()
		} else {
			// Write operation
			id := rand.Intn(t.DataSize) + 1
			_, err := conn.Exec(ctx, 
				fmt.Sprintf("UPDATE %s SET counter = counter + 1 WHERE id = ?", t.TableName), id)
			return err
		}
	})
}

func (t *MixedWorkloadBenchmarkTest) Cleanup(ctx context.Context, pool DatabasePool) error {
	return pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", t.TableName))
		return err
	})
}
