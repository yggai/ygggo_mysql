package main

import (
	"context"
	"fmt"
	"log"

	ggm "github.com/yggai/ygggo_mysql"
)

// 用户结构体
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Age       int    `json:"age"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func main() {
	fmt.Println("🚀 开始表数据增删改查示例...")

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

	// 使用连接进行表数据操作
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

		// 3. 创建 user 表
		fmt.Println("\n➕ 创建 user 表...")
		err = createUserTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("创建表失败: %v", err)
		}
		fmt.Println("✅ user 表创建成功!")

		// 4. 查询表数据 (查) - 初始状态
		fmt.Println("\n🔍 查询初始表数据...")
		err = queryUsers(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据失败: %v", err)
		}

		// 5. 添加表数据 (增)
		fmt.Println("\n➕ 添加表数据...")
		err = insertUsers(ctx, conn)
		if err != nil {
			return fmt.Errorf("插入数据失败: %v", err)
		}
		fmt.Println("✅ 数据添加成功!")

		// 6. 查询表数据 (查) - 验证添加
		fmt.Println("\n🔍 验证数据添加结果...")
		err = queryUsers(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据失败: %v", err)
		}

		// 7. 修改表数据 (改)
		fmt.Println("\n✏️ 修改表数据...")
		err = updateUser(ctx, conn)
		if err != nil {
			return fmt.Errorf("修改数据失败: %v", err)
		}
		fmt.Println("✅ 数据修改成功!")

		// 8. 查询表数据 (查) - 验证修改
		fmt.Println("\n🔍 验证数据修改结果...")
		err = queryUsers(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据失败: %v", err)
		}

		// 9. 删除表数据 (删)
		fmt.Println("\n🗑️ 删除表数据...")
		err = deleteUser(ctx, conn)
		if err != nil {
			return fmt.Errorf("删除数据失败: %v", err)
		}
		fmt.Println("✅ 数据删除成功!")

		// 10. 查询表数据 (查) - 验证删除
		fmt.Println("\n🔍 验证数据删除结果...")
		err = queryUsers(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询数据失败: %v", err)
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

	fmt.Println("\n🎉 表数据增删改查示例完成!")
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

// 创建 user 表
func createUserTable(ctx context.Context, conn ggm.DatabaseConn) error {
	query := `
	CREATE TABLE IF NOT EXISTS user (
		id INT AUTO_INCREMENT PRIMARY KEY COMMENT '用户ID',
		name VARCHAR(100) NOT NULL COMMENT '用户姓名',
		age INT NOT NULL COMMENT '用户年龄',
		email VARCHAR(255) DEFAULT NULL COMMENT '用户邮箱',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表'`

	_, err := conn.Exec(ctx, query)
	return err
}

// 查询用户数据
func queryUsers(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "SELECT id, name, age, email, created_at FROM user ORDER BY id"
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("📊 用户数据:")
	fmt.Println("ID\t姓名\t年龄\t邮箱\t\t\t创建时间")
	fmt.Println("--\t----\t----\t----\t\t\t--------")

	hasData := false
	for rows.Next() {
		var user User
		var email *string // 处理可能的 NULL 值
		err := rows.Scan(&user.ID, &user.Name, &user.Age, &email, &user.CreatedAt)
		if err != nil {
			return err
		}
		
		// 处理邮箱 NULL 值
		if email != nil {
			user.Email = *email
		} else {
			user.Email = "无"
		}
		
		fmt.Printf("%d\t%s\t%d\t%s\t\t%s\n", 
			user.ID, user.Name, user.Age, user.Email, user.CreatedAt[:19])
		hasData = true
	}

	if !hasData {
		fmt.Println("(暂无数据)")
	}

	return rows.Err()
}

// 添加用户数据
func insertUsers(ctx context.Context, conn ggm.DatabaseConn) error {
	// 单条插入
	query := "INSERT INTO user (name, age, email) VALUES (?, ?, ?)"
	_, err := conn.Exec(ctx, query, "张三", 25, "zhangsan@example.com")
	if err != nil {
		return err
	}

	// 批量插入
	columns := []string{"name", "age", "email"}
	rows := [][]any{
		{"李四", 30, "lisi@example.com"},
		{"王五", 28, "wangwu@example.com"},
		{"赵六", 35, nil}, // NULL 邮箱
	}
	_, err = conn.BulkInsert(ctx, "user", columns, rows)
	return err
}

// 修改用户数据
func updateUser(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "UPDATE user SET age = ?, email = ? WHERE name = ?"
	result, err := conn.Exec(ctx, query, 26, "zhangsan_new@example.com", "张三")
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	fmt.Printf("✅ 更新了 %d 条记录 (张三的年龄改为26岁，邮箱已更新)\n", affected)
	return nil
}

// 删除用户数据
func deleteUser(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DELETE FROM user WHERE name = ?"
	result, err := conn.Exec(ctx, query, "王五")
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	fmt.Printf("✅ 删除了 %d 条记录 (删除了王五)\n", affected)
	return nil
}

// 删除 test 数据库
func dropTestDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DROP DATABASE IF EXISTS test"
	_, err := conn.Exec(ctx, query)
	return err
}
