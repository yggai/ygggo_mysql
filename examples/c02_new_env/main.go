package main

import (
	"context"
	"fmt"
	"log"
	"time"

	gge "github.com/yggai/ygggo_env"
	ggm "github.com/yggai/ygggo_mysql"
)

func main() {
	// 自动查找并加载环境变量
	gge.LoadEnv()

	// 自动读取环境变量里面的值创建数据库连接池对象
	// 创建连接
	// 使用显式可控的上下文，避免默认背景上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := ggm.NewPoolEnv(ctx)
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
	fmt.Println("数据库连接信息：", ggm.GetDSN())
}
