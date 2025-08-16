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
		_, err := c.Exec(ctx, `CREATE TABLE IF NOT EXISTS t (
			id INT AUTO_INCREMENT PRIMARY KEY,
			a INT,
			name VARCHAR(100)
		)`)
		if err != nil { return err }

		// 清理数据
		_, err = c.Exec(ctx, "DELETE FROM t")
		if err != nil { return err }

		// 插入一些测试数据
		_, err = c.Exec(ctx, "INSERT INTO t (a, name) VALUES (1, 'a'), (2, 'b')")
		if err != nil { return err }

		return nil
	})
	if err != nil { log.Fatalf("Setup: %v", err) }

	// Use a single connection and enable LRU stmt cache (capacity 2)
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		c.EnableStmtCache(2)
		if _, err := c.ExecCached(ctx, "INSERT INTO t(a) VALUES(?)", 1); err != nil { return err }
		if _, err := c.ExecCached(ctx, "INSERT INTO t(a) VALUES(?)", 2); err != nil { return err }

		rs, err := c.QueryCached(ctx, "SELECT id,name FROM t")
		if err != nil { return err }
		defer rs.Close()
		count := 0
		for rs.Next() { count++ }
		fmt.Println("query rows:", count)
		return rs.Err()
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	fmt.Println("ygggo_mysql example: stmt_cache (prepared once, reused)", ygggo_mysql.Version())
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

