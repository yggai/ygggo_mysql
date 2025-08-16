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
			name VARCHAR(100)
		)`)
		if err != nil { return err }

		// 清理并插入测试数据
		_, err = c.Exec(ctx, "DELETE FROM t")
		if err != nil { return err }

		_, err = c.Exec(ctx, "INSERT INTO t (name) VALUES ('a'), ('b')")
		if err != nil { return err }

		return nil
	})
	if err != nil { log.Fatalf("Setup: %v", err) }

	// 使用流式查询
	count := 0
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		return c.QueryStream(ctx, "SELECT id,name FROM t", func(row []any) error {
			count++
			fmt.Printf("Row %d: id=%v, name=%v\n", count, row[0], row[1])
			return nil
		})
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	fmt.Printf("ygggo_mysql example: stream_query - processed %d rows %s\n", count, ygggo_mysql.Version())
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

