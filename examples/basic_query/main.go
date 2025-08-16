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

	// 执行查询
	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		rs, err := c.Query(ctx, "SELECT 1 as test_column")
		if err != nil { return err }
		defer rs.Close()

		var count int
		for rs.Next() {
			var value int
			if err := rs.Scan(&value); err != nil {
				return err
			}
			fmt.Printf("Query result: %d\n", value)
			count++
		}
		fmt.Printf("Total rows: %d\n", count)
		return rs.Err()
	})
	if err != nil {
		log.Fatalf("WithConn: %v", err)
	}

	fmt.Println("ygggo_mysql example: basic_query", ygggo_mysql.Version())
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

