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

	// 使用显式可控的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 自动管理 Docker 中的 MySQL：如果没有就安装并启动
	if !ggm.IsDockerInstalled(ctx) {
		log.Fatalf("❌ 未检测到 Docker，请先安装 Docker 再运行该示例")
	}
	if err := ggm.NewMySQL(ctx); err != nil {
		log.Fatalf("安装/启动 MySQL 失败: %v", err)
	}

	// 等待数据库就绪片刻（简单演示；生产建议更稳健的健康检查）
	time.Sleep(3 * time.Second)

	// 读取环境变量创建数据库连接池
	pool, err := ggm.NewPoolEnv(ctx)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer pool.Close()

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Ping失败: %v", err)
	}
	fmt.Println("✅ 数据库连接成功! DSN:", ggm.GetDSN())
}
