package main

import (
	"context"
	"fmt"
	"log"

	ggm "github.com/yggai/ygggo_mysql"
)

// 账户结构体
type Account struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

func main() {
	fmt.Println("🚀 开始事务转账示例...")

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

	// 使用连接进行事务操作
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

		// 3. 创建账户表
		fmt.Println("\n➕ 创建账户表...")
		err = createAccountTable(ctx, conn)
		if err != nil {
			return fmt.Errorf("创建表失败: %v", err)
		}
		fmt.Println("✅ 账户表创建成功!")

		// 4. 初始化账户数据
		fmt.Println("\n➕ 初始化账户数据...")
		err = initAccountData(ctx, conn)
		if err != nil {
			return fmt.Errorf("初始化数据失败: %v", err)
		}
		fmt.Println("✅ 账户数据初始化成功!")

		// 5. 查看初始账户余额
		fmt.Println("\n🔍 查看初始账户余额...")
		err = queryAccounts(ctx, conn)
		if err != nil {
			return fmt.Errorf("查询账户失败: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("❌ 初始化失败: %v", err)
	}

	// 6. 演示成功的转账事务
	fmt.Println("\n💰 演示成功转账: 张三给李四转账100元...")
	err = transferMoney(ctx, pool, "张三", "李四", 100.0)
	if err != nil {
		fmt.Printf("❌ 转账失败: %v\n", err)
	} else {
		fmt.Println("✅ 转账成功!")
	}

	// 7. 查看转账后的余额
	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
		err := useTestDatabase(ctx, conn)
		if err != nil {
			return err
		}
		fmt.Println("\n🔍 查看转账后的账户余额...")
		return queryAccounts(ctx, conn)
	})
	if err != nil {
		log.Printf("❌ 查询失败: %v", err)
	}

	// 8. 演示失败的转账事务（余额不足）
	fmt.Println("\n💸 演示失败转账: 张三给李四转账2000元（余额不足）...")
	err = transferMoney(ctx, pool, "张三", "李四", 2000.0)
	if err != nil {
		fmt.Printf("❌ 转账失败: %v\n", err)
	} else {
		fmt.Println("✅ 转账成功!")
	}

	// 9. 查看失败转账后的余额（应该没有变化）
	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
		err := useTestDatabase(ctx, conn)
		if err != nil {
			return err
		}
		fmt.Println("\n🔍 查看失败转账后的账户余额（应该没有变化）...")
		return queryAccounts(ctx, conn)
	})
	if err != nil {
		log.Printf("❌ 查询失败: %v", err)
	}

	// 10. 清理 test 数据库
	err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
		fmt.Println("\n🧹 清理 test 数据库...")
		err := dropTestDatabase(ctx, conn)
		if err != nil {
			return fmt.Errorf("删除数据库失败: %v", err)
		}
		fmt.Println("✅ test 数据库清理完成!")
		return nil
	})
	if err != nil {
		log.Printf("❌ 清理失败: %v", err)
	}

	fmt.Println("\n🎉 事务转账示例完成!")
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

// 创建账户表
func createAccountTable(ctx context.Context, conn ggm.DatabaseConn) error {
	query := `
	CREATE TABLE IF NOT EXISTS account (
		id INT AUTO_INCREMENT PRIMARY KEY COMMENT '账户ID',
		name VARCHAR(100) NOT NULL UNIQUE COMMENT '账户名称',
		balance DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '账户余额',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='账户表'`

	_, err := conn.Exec(ctx, query)
	return err
}

// 初始化账户数据
func initAccountData(ctx context.Context, conn ggm.DatabaseConn) error {
	// 清空现有数据
	_, err := conn.Exec(ctx, "DELETE FROM account")
	if err != nil {
		return err
	}

	// 插入初始账户
	columns := []string{"name", "balance"}
	rows := [][]any{
		{"张三", 1000.00},
		{"李四", 500.00},
		{"王五", 800.00},
	}
	_, err = conn.BulkInsert(ctx, "account", columns, rows)
	return err
}

// 查询所有账户
func queryAccounts(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "SELECT id, name, balance FROM account ORDER BY id"
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("📊 账户余额:")
	fmt.Println("ID\t姓名\t余额")
	fmt.Println("--\t----\t----")

	for rows.Next() {
		var account Account
		err := rows.Scan(&account.ID, &account.Name, &account.Balance)
		if err != nil {
			return err
		}
		fmt.Printf("%d\t%s\t%.2f元\n", account.ID, account.Name, account.Balance)
	}

	return rows.Err()
}

// 转账操作（使用事务）
func transferMoney(ctx context.Context, pool *ggm.Pool, fromName, toName string, amount float64) error {
	return pool.WithinTx(ctx, func(tx ggm.DatabaseTx) error {
		// 切换到 test 数据库
		_, err := tx.Exec(ctx, "USE test")
		if err != nil {
			return fmt.Errorf("切换数据库失败: %v", err)
		}

		// 1. 直接尝试从转出账户扣款（包含余额检查）
		result, err := tx.Exec(ctx, "UPDATE account SET balance = balance - ? WHERE name = ? AND balance >= ?", amount, fromName, amount)
		if err != nil {
			return fmt.Errorf("扣款失败: %v", err)
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("转账失败: 账户 %s 不存在或余额不足（需要 %.2f元）", fromName, amount)
		}

		// 2. 向转入账户加款
		result, err = tx.Exec(ctx, "UPDATE account SET balance = balance + ? WHERE name = ?", amount, toName)
		if err != nil {
			return fmt.Errorf("加款失败: %v", err)
		}
		affected, _ = result.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("转账失败: 转入账户 %s 不存在", toName)
		}

		fmt.Printf("✅ 转账详情: %s -> %s, 金额: %.2f元\n", fromName, toName, amount)
		return nil
	})
}

// 删除 test 数据库
func dropTestDatabase(ctx context.Context, conn ggm.DatabaseConn) error {
	query := "DROP DATABASE IF EXISTS test"
	_, err := conn.Exec(ctx, query)
	return err
}
