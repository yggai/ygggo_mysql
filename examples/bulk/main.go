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

	err = pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// 创建测试表
		_, err := c.Exec(ctx, "CREATE TABLE IF NOT EXISTS t (id INT AUTO_INCREMENT PRIMARY KEY, a INT, b TEXT, UNIQUE KEY(a))")
		if err != nil { return err }

		// 清理数据
		_, err = c.Exec(ctx, "DELETE FROM t")
		if err != nil { return err }

		// 批量插入
		rows := [][]any{{1, "x"}, {2, "y"}}
		res, err := c.BulkInsert(ctx, "t", []string{"a", "b"}, rows)
		if err != nil { return err }
		aff, _ := res.RowsAffected()
		fmt.Println("bulk affected:", aff)

		// 重复插入时使用ON DUPLICATE KEY UPDATE
		res, err = c.InsertOnDuplicate(ctx, "t", []string{"a", "b"}, rows, []string{"b"})
		if err != nil { return err }
		aff, _ = res.RowsAffected()
		fmt.Println("upsert affected:", aff)

		return nil
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	fmt.Println("ygggo_mysql example: bulk & upsert", ygggo_mysql.Version())
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

