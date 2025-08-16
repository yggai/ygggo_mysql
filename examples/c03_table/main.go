package main

import (
	"context"
	"fmt"
	"log"

	ggm "github.com/yggai/ygggo_mysql"
)

// 表信息结构体
type TableInfo struct {
	Name string `json:"name"`
}

func main() {
	fmt.Println("🚀 开始表增删改查示例...")

	// 数据库配置 - 先连接到 mysql 系统数据库
	config := ggm.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "zhangdapeng520",
		Database: "mysql", // 先连接到系统数据库
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

	// 使用连接进行表操作
	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
		// 1. 创建 test 数据库
		fmt.Println("\n➕ 创建 test 数据库...")
		err := createTestDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("创建数据库失败: %v", err)
		}
		fmt.Println("✅ test 数据库创建成功!")

		// 2. 切换到 test 数据库
		fmt.Println("\n🔄 切换到 test 数据库...")
		err = useTestDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("切换数据库失败: %v", err)
		}
		fmt.Println("✅ 已切换到 test 数据库!")

		// 3. 查看当前数据库中的所有表 (查)
		fmt.Println("\n🔍 查看 test 数据库中的所有表...")
		err = showTables(ctx, conn)
		if err != nil {
			return fmt.Errorf("查看表失败: %v", err)
		}

		// 4. 创建 user 表 (增)
		fmt.Println("\n➕ 创建 user 表...")
		err = createUserTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("创建表失败: %v", err)
		}
		fmt.Println("✅ user 表创建成功!")

		// 5. 再次查看所有表，验证创建
		fmt.Println("\n🔍 验证表创建结果...")
		err = showTables(ctx, conn)
		if err != nil {
			return fmt.Errorf("查看表失败: %v", err)
		}

		// 6. 查看 user 表结构 (查)
		fmt.Println("\n🔍 查看 user 表结构...")
		err = describeUserTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("查看表结构失败: %v", err)
		}

		// 7. 修改表结构 (改) - 添加一个字段
		fmt.Println("\n✏️ 修改 user 表结构 (添加 email 字段)...")
		err = alterUserTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("修改表失败: %v", err)
		}
		fmt.Println("✅ user 表修改成功!")

		// 8. 再次查看表结构，验证修改
		fmt.Println("\n🔍 验证表修改结果...")
		err = describeUserTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("查看表结构失败: %v", err)
		}

		// 9. 删除 user 表 (删)
		fmt.Println("\n🗑️ 删除 user 表...")
		err = dropUserTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("删除表失败: %v", err)
		}
		fmt.Println("✅ user 表删除成功!")

		// 10. 最终查看所有表，验证删除
		fmt.Println("\n🔍 验证表删除结果...")
		err = showTables(ctx, conn)
		if err != nil {
			return fmt.Errorf("查看表失败: %v", err)
		}

		// 11. 清理 test 数据库
		fmt.Println("\n🧹 清理 test 数据库...")
		err = dropTestDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("删除数据库失败: %v", err)
		}
		fmt.Println("✅ test 数据库清理完成!")

		return nil
	})

	if err != nil {
		log.Fatalf("❌ 操作失败: %v", err)
	}

	fmt.Println("\n🎉 表增删改查示例完成!")
}

// 创建 test 数据库
func createTestDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "CREATE DATABASE IF NOT EXISTS test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
	_, err := conn.Exec(ctx, query)
	return err
}

// 切换到 test 数据库
func useTestDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "USE test"
	_, err := conn.Exec(ctx, query)
	return err
}

// 查看所有表
func showTables(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "SHOW TABLES"
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("📊 表列表:")
	fmt.Println("表名称")
	fmt.Println("------")

	hasTable := false
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", tableName)
		hasTable = true
	}

	if !hasTable {
		fmt.Println("(暂无表)")
	}

	return rows.Err()
}

// 创建 user 表
func createUserTable(ctx context.Context, conn ggm.DatabaseConn) error {
	query := `
	CREATE TABLE IF NOT EXISTS user (
		id INT AUTO_INCREMENT PRIMARY KEY COMMENT '用户ID',
		name VARCHAR(100) NOT NULL COMMENT '用户姓名',
		age INT NOT NULL COMMENT '用户年龄',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表'`

	_, err := conn.Exec(ctx, query)
	return err
}

// 查看 user 表结构
func describeUserTable(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DESCRIBE user"
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("📊 user 表结构:")
	fmt.Println("字段名\t\t类型\t\t\t空值\t键\t默认值\t\t额外")
	fmt.Println("------\t\t----\t\t\t----\t--\t------\t\t----")

	for rows.Next() {
		var field, fieldType, null, key, extra string
		var defaultVal *string // 使用指针处理 NULL 值
		err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra)
		if err != nil {
			return err
		}

		// 处理 NULL 值显示
		defaultStr := "NULL"
		if defaultVal != nil {
			defaultStr = *defaultVal
		}

		fmt.Printf("%s\t\t%s\t\t%s\t%s\t%s\t\t%s\n",
			field, fieldType, null, key, defaultStr, extra)
	}

	return rows.Err()
}

// 修改 user 表结构 - 添加 email 字段
func alterUserTable(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "ALTER TABLE user ADD COLUMN email VARCHAR(255) DEFAULT NULL COMMENT '用户邮箱'"
	_, err := conn.Exec(ctx, query)
	return err
}

// 删除 user 表
func dropUserTable(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DROP TABLE IF EXISTS user"
	_, err := conn.Exec(ctx, query)
	return err
}

// 删除 test 数据库
func dropTestDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DROP DATABASE IF EXISTS test"
	_, err := conn.Exec(ctx, query)
	return err
}
