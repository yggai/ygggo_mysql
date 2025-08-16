package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryBuilder_BasicSelect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) NOT NULL,
			age INT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(context.Background(), 
		"INSERT INTO test_users (name, email, age) VALUES (?, ?, ?), (?, ?, ?)",
		"John Doe", "john@example.com", 25,
		"Jane Smith", "jane@example.com", 30,
	)
	require.NoError(t, err)

	// Test basic SELECT query
	qb := NewQueryBuilder(conn)
	rows, err := qb.Select("id", "name", "email").
		From("test_users").
		Query(context.Background())
	
	require.NoError(t, err)
	defer rows.Close()

	// Verify we can read the data
	var users []struct {
		ID    int
		Name  string
		Email string
	}

	for rows.Next() {
		var user struct {
			ID    int
			Name  string
			Email string
		}
		err := rows.Scan(&user.ID, &user.Name, &user.Email)
		require.NoError(t, err)
		users = append(users, user)
	}

	assert.Len(t, users, 2)
	assert.Equal(t, "John Doe", users[0].Name)
	assert.Equal(t, "jane@example.com", users[1].Email)
}

func TestQueryBuilder_SelectWithWhere(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) NOT NULL,
			age INT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(context.Background(), 
		"INSERT INTO test_users (name, email, age) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)",
		"John Doe", "john@example.com", 25,
		"Jane Smith", "jane@example.com", 30,
		"Bob Johnson", "bob@example.com", 20,
	)
	require.NoError(t, err)

	// Test SELECT with WHERE clause
	qb := NewQueryBuilder(conn)
	rows, err := qb.Select("name", "age").
		From("test_users").
		Where("age > ?", 22).
		Query(context.Background())
	
	require.NoError(t, err)
	defer rows.Close()

	// Verify filtered results
	var users []struct {
		Name string
		Age  int
	}

	for rows.Next() {
		var user struct {
			Name string
			Age  int
		}
		err := rows.Scan(&user.Name, &user.Age)
		require.NoError(t, err)
		users = append(users, user)
	}

	assert.Len(t, users, 2) // Should exclude Bob (age 20)
	
	// Verify the correct users are returned
	names := make([]string, len(users))
	for i, user := range users {
		names[i] = user.Name
		assert.Greater(t, user.Age, 22)
	}
	assert.Contains(t, names, "John Doe")
	assert.Contains(t, names, "Jane Smith")
}

func TestQueryBuilder_SelectWithOrderByAndLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			age INT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(context.Background(), 
		"INSERT INTO test_users (name, age) VALUES (?, ?), (?, ?), (?, ?)",
		"Charlie", 35,
		"Alice", 25,
		"Bob", 30,
	)
	require.NoError(t, err)

	// Test SELECT with ORDER BY and LIMIT
	qb := NewQueryBuilder(conn)
	rows, err := qb.Select("name", "age").
		From("test_users").
		OrderBy("age ASC").
		Limit(2).
		Query(context.Background())
	
	require.NoError(t, err)
	defer rows.Close()

	// Verify ordered and limited results
	var users []struct {
		Name string
		Age  int
	}

	for rows.Next() {
		var user struct {
			Name string
			Age  int
		}
		err := rows.Scan(&user.Name, &user.Age)
		require.NoError(t, err)
		users = append(users, user)
	}

	assert.Len(t, users, 2)
	assert.Equal(t, "Alice", users[0].Name) // Youngest first
	assert.Equal(t, 25, users[0].Age)
	assert.Equal(t, "Bob", users[1].Name)   // Second youngest
	assert.Equal(t, 30, users[1].Age)
}

func TestQueryBuilder_Insert(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_products (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			category VARCHAR(50) NOT NULL
		)
	`)
	require.NoError(t, err)

	// Test INSERT with Values
	qb := NewQueryBuilder(conn)
	result, err := qb.Insert("test_products").
		Values(map[string]any{
			"name":     "Laptop",
			"price":    999.99,
			"category": "Electronics",
		}).
		Exec(context.Background())

	require.NoError(t, err)

	// Verify the insert worked
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	lastInsertID, err := result.LastInsertId()
	require.NoError(t, err)
	assert.Greater(t, lastInsertID, int64(0))

	// Verify data was inserted correctly
	rows, err := conn.Query(context.Background(),
		"SELECT name, price, category FROM test_products WHERE id = ?", lastInsertID)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	var name, category string
	var price float64
	err = rows.Scan(&name, &price, &category)
	require.NoError(t, err)

	assert.Equal(t, "Laptop", name)
	assert.Equal(t, 999.99, price)
	assert.Equal(t, "Electronics", category)
}

func TestQueryBuilder_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table and insert data
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_products (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			category VARCHAR(50) NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = conn.Exec(context.Background(),
		"INSERT INTO test_products (name, price, category) VALUES (?, ?, ?), (?, ?, ?)",
		"Laptop", 999.99, "Electronics",
		"Mouse", 29.99, "Electronics",
	)
	require.NoError(t, err)

	// Test UPDATE with WHERE
	qb := NewQueryBuilder(conn)
	result, err := qb.Update("test_products").
		Set("price", 899.99).
		Set("category", "Computers").
		Where("name = ?", "Laptop").
		Exec(context.Background())

	require.NoError(t, err)

	// Verify the update worked
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Verify data was updated correctly
	rows, err := conn.Query(context.Background(),
		"SELECT price, category FROM test_products WHERE name = ?", "Laptop")
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	var price float64
	var category string
	err = rows.Scan(&price, &category)
	require.NoError(t, err)

	assert.Equal(t, 899.99, price)
	assert.Equal(t, "Computers", category)
}

func TestQueryBuilder_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table and insert data
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_products (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			category VARCHAR(50) NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = conn.Exec(context.Background(),
		"INSERT INTO test_products (name, price, category) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)",
		"Laptop", 999.99, "Electronics",
		"Mouse", 29.99, "Electronics",
		"Book", 19.99, "Education",
	)
	require.NoError(t, err)

	// Test DELETE with WHERE
	qb := NewQueryBuilder(conn)
	result, err := qb.Delete("test_products").
		Where("category = ?", "Electronics").
		Exec(context.Background())

	require.NoError(t, err)

	// Verify the delete worked
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(2), rowsAffected) // Should delete Laptop and Mouse

	// Verify only the Book remains
	rows, err := conn.Query(context.Background(), "SELECT COUNT(*) FROM test_products")
	require.NoError(t, err)

	require.True(t, rows.Next())
	var count int
	err = rows.Scan(&count)
	require.NoError(t, err)
	rows.Close() // Close explicitly before next query
	assert.Equal(t, 1, count)

	// Verify it's the Book that remains
	rows2, err := conn.Query(context.Background(), "SELECT name FROM test_products")
	require.NoError(t, err)
	defer rows2.Close()

	require.True(t, rows2.Next())
	var name string
	err = rows2.Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Book", name)
}

func TestQueryBuilder_MultipleWhereConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test table
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE test_users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			age INT NOT NULL,
			status VARCHAR(20) NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(context.Background(),
		"INSERT INTO test_users (name, age, status) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)",
		"John", 25, "active",
		"Jane", 30, "active",
		"Bob", 20, "inactive",
		"Alice", 35, "active",
	)
	require.NoError(t, err)

	// Test multiple WHERE conditions
	qb := NewQueryBuilder(conn)
	rows, err := qb.Select("name", "age").
		From("test_users").
		Where("age > ?", 22).
		Where("status = ?", "active").
		OrderBy("age ASC").
		Query(context.Background())

	require.NoError(t, err)
	defer rows.Close()

	// Should return John (25) and Jane (30), but not Bob (inactive) or Alice (too old for this test)
	var users []struct {
		Name string
		Age  int
	}

	for rows.Next() {
		var user struct {
			Name string
			Age  int
		}
		err := rows.Scan(&user.Name, &user.Age)
		require.NoError(t, err)
		users = append(users, user)
	}

	assert.Len(t, users, 3) // John, Jane, Alice
	assert.Equal(t, "John", users[0].Name)
	assert.Equal(t, 25, users[0].Age)
	assert.Equal(t, "Jane", users[1].Name)
	assert.Equal(t, 30, users[1].Age)
	assert.Equal(t, "Alice", users[2].Name)
	assert.Equal(t, 35, users[2].Age)
}

func TestQueryBuilder_JoinOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	helper, err := NewDockerTestHelper(context.Background())
	require.NoError(t, err)
	defer helper.Close()

	conn, err := helper.Pool().Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	// Create test tables
	_, err = conn.Exec(context.Background(), `
		CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL,
			department_id INT
		)
	`)
	require.NoError(t, err)

	_, err = conn.Exec(context.Background(), `
		CREATE TABLE departments (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert test data
	_, err = conn.Exec(context.Background(),
		"INSERT INTO departments (name) VALUES (?), (?)",
		"Engineering", "Marketing",
	)
	require.NoError(t, err)

	_, err = conn.Exec(context.Background(),
		"INSERT INTO users (name, department_id) VALUES (?, ?), (?, ?), (?, ?)",
		"John", 1, "Jane", 1, "Bob", 2,
	)
	require.NoError(t, err)

	// Test JOIN operation
	qb := NewQueryBuilder(conn)
	rows, err := qb.Select("u.name", "d.name AS department").
		From("users u").
		Join("INNER JOIN departments d ON u.department_id = d.id").
		Where("d.name = ?", "Engineering").
		OrderBy("u.name ASC").
		Query(context.Background())

	require.NoError(t, err)
	defer rows.Close()

	var results []struct {
		UserName   string
		Department string
	}

	for rows.Next() {
		var result struct {
			UserName   string
			Department string
		}
		err := rows.Scan(&result.UserName, &result.Department)
		require.NoError(t, err)
		results = append(results, result)
	}

	assert.Len(t, results, 2) // John and Jane from Engineering
	assert.Equal(t, "Jane", results[0].UserName)
	assert.Equal(t, "Engineering", results[0].Department)
	assert.Equal(t, "John", results[1].UserName)
	assert.Equal(t, "Engineering", results[1].Department)
}
