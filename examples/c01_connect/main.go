package main

import (
	"context"
	"fmt"
	"log"

	ggm "github.com/yggai/ygggo_mysql"
)

func main() {
	fmt.Println("开始连接数据库...")

	// 数据库配置
	config := ggm.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "zhangdapeng520",
		Database: "mysql",
		Driver:   "mysql",
	}

	// 创建连接
	ctx := context.Background()
	pool, err := ggm.NewPool(ctx, config)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer pool.Close()

	// 测试连接
	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("Ping失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功!")
	fmt.Printf("连接信息: %s@%s:%d/%s\n", config.Username, config.Host, config.Port, config.Database)
}




