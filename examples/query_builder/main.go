package main

import (
	"context"
	"fmt"
	"log"

	ygggo "github.com/yggai/ygggo_mysql"
)

func main() {
	// This example demonstrates the Query Builder functionality
	// Note: This requires a running MySQL instance
	
	config := ygggo.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
		Pool: ygggo.PoolConfig{
			MaxOpen: 10,
		},
	}

	ctx := context.Background()

	pool, err := ygggo.NewPool(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Get a connection
	conn, err := pool.Acquire(ctx)
	if err != nil {
		log.Fatalf("Failed to acquire connection: %v", err)
	}
	defer conn.Close()

	// Create example tables
	if err := createExampleTables(ctx, conn); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// Insert sample data
	if err := insertSampleData(ctx, conn); err != nil {
		log.Fatalf("Failed to insert data: %v", err)
	}

	// Demonstrate various query builder features
	fmt.Println("=== Query Builder Examples ===\n")

	// Example 1: Basic SELECT
	fmt.Println("1. Basic SELECT:")
	basicSelect(ctx, conn)

	// Example 2: SELECT with WHERE and ORDER BY
	fmt.Println("\n2. SELECT with WHERE and ORDER BY:")
	selectWithConditions(ctx, conn)

	// Example 3: JOIN operations
	fmt.Println("\n3. JOIN operations:")
	joinExample(ctx, conn)

	// Example 4: INSERT operation
	fmt.Println("\n4. INSERT operation:")
	insertExample(ctx, conn)

	// Example 5: UPDATE operation
	fmt.Println("\n5. UPDATE operation:")
	updateExample(ctx, conn)

	// Example 6: DELETE operation
	fmt.Println("\n6. DELETE operation:")
	deleteExample(ctx, conn)

	fmt.Println("\n=== Query Builder Examples Complete ===")
}

func createExampleTables(ctx context.Context, conn ygggo.DatabaseConn) error {
	// Create users table
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) NOT NULL,
			age INT NOT NULL,
			department_id INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create departments table
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS departments (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			budget DECIMAL(10,2) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create departments table: %w", err)
	}

	return nil
}

func insertSampleData(ctx context.Context, conn ygggo.DatabaseConn) error {
	// Clear existing data
	_, err := conn.Exec(ctx, "DELETE FROM users")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "DELETE FROM departments")
	if err != nil {
		return err
	}

	// Insert departments
	qb := ygggo.NewQueryBuilder(conn)
	_, err = qb.Insert("departments").
		Values(map[string]any{
			"name":   "Engineering",
			"budget": 500000.00,
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = qb.Insert("departments").
		Values(map[string]any{
			"name":   "Marketing",
			"budget": 200000.00,
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Insert users
	users := []map[string]any{
		{"name": "John Doe", "email": "john@example.com", "age": 30, "department_id": 1},
		{"name": "Jane Smith", "email": "jane@example.com", "age": 28, "department_id": 1},
		{"name": "Bob Johnson", "email": "bob@example.com", "age": 35, "department_id": 2},
		{"name": "Alice Brown", "email": "alice@example.com", "age": 32, "department_id": 1},
	}

	for _, user := range users {
		_, err = qb.Insert("users").Values(user).Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func basicSelect(ctx context.Context, conn ygggo.DatabaseConn) {
	qb := ygggo.NewQueryBuilder(conn)
	rows, err := qb.Select("name", "email", "age").
		From("users").
		Query(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("All users:")
	for rows.Next() {
		var name, email string
		var age int
		if err := rows.Scan(&name, &email, &age); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("  %s (%s) - Age: %d\n", name, email, age)
	}
}

func selectWithConditions(ctx context.Context, conn ygggo.DatabaseConn) {
	qb := ygggo.NewQueryBuilder(conn)
	rows, err := qb.Select("name", "age").
		From("users").
		Where("age > ?", 29).
		Where("department_id = ?", 1).
		OrderBy("age DESC").
		Limit(2).
		Query(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("Engineering users over 29, ordered by age (limit 2):")
	for rows.Next() {
		var name string
		var age int
		if err := rows.Scan(&name, &age); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("  %s - Age: %d\n", name, age)
	}
}

func joinExample(ctx context.Context, conn ygggo.DatabaseConn) {
	qb := ygggo.NewQueryBuilder(conn)
	rows, err := qb.Select("u.name", "u.age", "d.name AS department", "d.budget").
		From("users u").
		Join("INNER JOIN departments d ON u.department_id = d.id").
		Where("u.age > ?", 30).
		OrderBy("d.budget DESC").
		OrderBy("u.name ASC").
		Query(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("Users over 30 with their departments:")
	for rows.Next() {
		var name, department string
		var age int
		var budget float64
		if err := rows.Scan(&name, &age, &department, &budget); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("  %s (Age: %d) - %s (Budget: $%.2f)\n", name, age, department, budget)
	}
}

func insertExample(ctx context.Context, conn ygggo.DatabaseConn) {
	qb := ygggo.NewQueryBuilder(conn)
	result, err := qb.Insert("users").
		Values(map[string]any{
			"name":          "Charlie Wilson",
			"email":         "charlie@example.com",
			"age":           27,
			"department_id": 2,
		}).
		Exec(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	lastID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Inserted new user with ID: %d (rows affected: %d)\n", lastID, rowsAffected)
}

func updateExample(ctx context.Context, conn ygggo.DatabaseConn) {
	qb := ygggo.NewQueryBuilder(conn)
	result, err := qb.Update("users").
		Set("age", 29).
		Set("email", "jane.smith@example.com").
		Where("name = ?", "Jane Smith").
		Exec(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Updated user (rows affected: %d)\n", rowsAffected)
}

func deleteExample(ctx context.Context, conn ygggo.DatabaseConn) {
	qb := ygggo.NewQueryBuilder(conn)
	result, err := qb.Delete("users").
		Where("name = ?", "Charlie Wilson").
		Exec(ctx)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Deleted user (rows affected: %d)\n", rowsAffected)
}
