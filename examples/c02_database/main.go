package main

import (
	"context"
	"fmt"
	"log"

	ggm "github.com/yggai/ygggo_mysql"
)

// 数据库信息结构体
type DatabaseInfo struct {
	Name string `json:"name"`
}

func main() {
	fmt.Println("🚀 开始数据库增删改查示例...")

	// 数据库配置 - 连接到 mysql 系统数据库进行数据库管理操作
	config := ggm.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "zhangdapeng520",
		Database: "mysql", // 连接到系统数据库
		Driver:   "mysql",
	}

	// 创建连接池
	ctx := context.Background()
	pool, err := ggm.NewPool(ctx, config)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer pool.Close()

	// 测试连接
	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("❌ Ping失败: %v", err)
	}
	fmt.Println("✅ 数据库连接成功!")

	// 使用连接进行数据库增删改查操作
	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
		// 1. 查询现有数据库 (查)
		fmt.Println("\n� 查询现有数据库...")
		err := showDatabases(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据库失败: %v", err)
		}

		// 2. 创建测试数据库 (增)
		fmt.Println("\n➕ 创建测试数据库...")
		err = createDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("创建数据库失败: %v", err)
		}
		fmt.Println("✅ 数据库创建成功!")

		// 3. 再次查询验证创建
		fmt.Println("\n🔍 验证数据库创建结果...")
		err = showDatabases(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据库失败: %v", err)
		}

		// 4. 查询特定数据库信息 (查)
		fmt.Println("\n🔍 查询测试数据库详细信息...")
		err = showDatabaseInfo(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据库信息失败: %v", err)
		}

		// 5. 修改数据库字符集 (改)
		fmt.Println("\n✏️ 修改数据库字符集...")
		err = alterDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("修改数据库失败: %v", err)
		}
		fmt.Println("✅ 数据库修改成功!")

		// 6. 再次查询验证修改
		fmt.Println("\n� 验证数据库修改结果...")
		err = showDatabaseInfo(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据库信息失败: %v", err)
		}

		// 7. 删除测试数据库 (删)
		fmt.Println("\n�️ 删除测试数据库...")
		err = dropDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("删除数据库失败: %v", err)
		}
		fmt.Println("✅ 数据库删除成功!")

		// 8. 最终查询验证删除
		fmt.Println("\n🔍 验证数据库删除结果...")
		err = showDatabases(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据库失败: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("❌ 操作失败: %v", err)
	}

	fmt.Println("\n🎉 数据库增删改查示例完成!")
}

// 查询所有数据库
func showDatabases(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "SHOW DATABASES"
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("📊 数据库列表:")
	fmt.Println("数据库名称")
	fmt.Println("--------")

	for rows.Next() {
		var dbName string
		err := rows.Scan(&dbName)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", dbName)
	}

	return rows.Err()
}

// 创建测试数据库
func createDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "CREATE DATABASE IF NOT EXISTS test_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
	_, err := conn.Exec(ctx, query)
	return err
}

// 查询特定数据库信息
func showDatabaseInfo(ctx context.Context, conn ggm.DatabaseConn) error {
	query := `SELECT
		SCHEMA_NAME as database_name,
		DEFAULT_CHARACTER_SET_NAME as charset,
		DEFAULT_COLLATION_NAME as collation
	FROM information_schema.SCHEMATA
	WHERE SCHEMA_NAME = 'test_db'`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("📊 test_db 数据库详细信息:")
	fmt.Println("数据库名\t字符集\t\t排序规则")
	fmt.Println("--------\t------\t\t--------")

	for rows.Next() {
		var dbName, charset, collation string
		err := rows.Scan(&dbName, &charset, &collation)
		if err != nil {
			return err
		}
		fmt.Printf("%s\t\t%s\t%s\n", dbName, charset, collation)
	}

	return rows.Err()
}

// 修改数据库字符集
func alterDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "ALTER DATABASE test_db CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"
	_, err := conn.Exec(ctx, query)
	return err
}

// 删除测试数据库
func dropDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DROP DATABASE IF EXISTS test_db"
	_, err := conn.Exec(ctx, query)
	return err
}
