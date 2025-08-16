package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yggai/ygggo_mysql"
)

type User struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

func main() {
	ctx := context.Background()

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

	// 设置测试数据
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// 创建测试表
		_, err := c.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
			id INT PRIMARY KEY,
			name VARCHAR(100),
			active BOOLEAN DEFAULT TRUE
		)`)
		if err != nil { return err }

		// 清理数据
		_, err = c.Exec(ctx, "DELETE FROM users")
		if err != nil { return err }

		return nil
	})
	if err != nil { log.Fatalf("Setup: %v", err) }

	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// Single struct
		_, err := c.NamedExec(ctx, "INSERT INTO users (id,name) VALUES (:id,:name)", User{ID: 1, Name: "Alice"})
		if err != nil { return err }

		// Slice of structs
		users := []User{{ID: 2, Name: "Bob"}, {ID: 3, Name: "Charlie"}}
		_, err = c.NamedExec(ctx, "INSERT INTO users (id,name) VALUES (:id,:name)", users)
		if err != nil { return err }

		// Map query
		rs, err := c.NamedQuery(ctx, "SELECT * FROM users WHERE id=:id", map[string]any{"id": 1})
		if err != nil { return err }
		defer rs.Close()
		for rs.Next() {
			var id int
			var name string
			if err := rs.Scan(&id, &name); err != nil { return err }
			fmt.Printf("Found user: %d, %s\n", id, name)
		}

		// BuildIn helper
		query, args, err := ygggo_mysql.BuildIn("SELECT * FROM users WHERE id IN (?) AND active=?", []int{1, 2, 3}, true)
		if err != nil { return err }
		rs, err = c.Query(ctx, query, args...)
		if err != nil { return err }
		defer rs.Close()
		count := 0
		for rs.Next() { count++ }
		fmt.Printf("BuildIn found %d users\n", count)

		return nil
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	fmt.Println("ygggo_mysql example: named parameters & BuildIn", ygggo_mysql.Version())
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
