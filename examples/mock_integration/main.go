package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()

	// Check if we should use mock or real DB
	isMock := os.Getenv("USE_MOCK") != "false" // default to mock

	var pool *ygggo_mysql.Pool
	var mock ygggo_mysql.MockExpectations
	var err error

	if isMock {
		fmt.Println("Using mock database")
		pool, mock, err = ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{}, true)
		if err != nil { log.Fatalf("NewPoolWithMock: %v", err) }

		// Set up mock expectations
		rows := ygggo_mysql.NewRows([]string{"result"})
		rows = ygggo_mysql.AddRow(rows, 1)
		mock.ExpectQuery(`SELECT 1`).WillReturnRows(rows)
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO users \(name\) VALUES \(\?\)`).WithArgs("Alice").WillReturnResult(ygggo_mysql.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectExec(`INSERT INTO users \(name,email\) VALUES \(\?,\?\),\(\?,\?\)`).
			WithArgs("Bob", "bob@example.com", "Charlie", "charlie@example.com").
			WillReturnResult(ygggo_mysql.NewResult(2, 2))
	} else {
		fmt.Println("Using real database")
		pool, mock, err = ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{
			Host: "localhost",
			Port: 3306,
			Username: "root",
			Password: "password",
			Database: "test",
		}, false)
		if err != nil { log.Fatalf("NewPoolWithMock: %v", err) }
	}
	defer pool.Close()
	
	// Test basic query
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT 1")
		if err != nil { return err }
		defer rs.Close()
		for rs.Next() {
			var result int
			if err := rs.Scan(&result); err != nil { return err }
			fmt.Println("Query result:", result)
		}
		return rs.Err()
	})
	if err != nil { log.Fatalf("Query: %v", err) }
	
	// Test transaction
	err = pool.WithinTx(ctx, func(tx ygggo_mysql.DatabaseTx) error {
		_, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
		return err
	})
	if err != nil { log.Fatalf("WithinTx: %v", err) }
	fmt.Println("Transaction completed")
	
	// Test bulk insert
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		rows := [][]any{{"Bob", "bob@example.com"}, {"Charlie", "charlie@example.com"}}
		_, err := c.BulkInsert(ctx, "users", []string{"name", "email"}, rows)
		return err
	})
	if err != nil { log.Fatalf("BulkInsert: %v", err) }
	fmt.Println("Bulk insert completed")
	
	// Verify mock expectations if using mock
	if isMock && mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil {
			log.Fatalf("Mock expectations not met: %v", err)
		}
		fmt.Println("All mock expectations met!")
	}
	
	fmt.Println("Example completed successfully")
}
