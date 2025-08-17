package main

import (
	"context"
	"fmt"
	"log"
	"time"

	gge "github.com/yggai/ygggo_env"
	ggm "github.com/yggai/ygggo_mysql"
)

type User struct {
	ID   int    `json:"id" ggm:"id,pk,auto"`
	Name string `json:"name" ggm:"name,notnull,default=Anonymous"`
}

func main() {
	// 自动查找并加载环境变量
	gge.LoadEnv()

	// 使用显式可控的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := ggm.NewPoolEnv(ctx)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer pool.Close()

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Ping失败: %v", err)
	}
	fmt.Println("✅ 数据库连接成功!")

	// 获取数据库连接对象
	db, _ := pool.GetDB()

	// 创建表格
	db.AddTable(&User{})

	// 打印建表SQL（推导）
	fmt.Println("建表SQL (推导)：", db.GetCreateTableSQL(&User{}))
	// 打印实际建表DDL（SHOW CREATE TABLE）
	fmt.Println("实际建表DDL：", db.ShowCreateTable(&User{}))

	// 查看所有表格
	fmt.Println("查看所有表格：", db.GetAllTable())

	// 删除表格
	db.DeleteTable(&User{})
}

