package main

import (
	"context"
	"fmt"
	"log"

	gge "github.com/yggai/ygggo_env"
	ygggo_mysql "github.com/yggai/ygggo_mysql"
)

// User 用户实体
type User struct {
	ID     int    `ggm:"id,primary_key,auto_increment"`
	Name   string `ggm:"name,not_null"`
	Email  string `ggm:"email,unique"`
	Age    int    `ggm:"age"`
	Status int    `ggm:"status,default:1"`
}

// TableName 实现表名接口
func (u User) TableName() string {
	return "users"
}

// Product 产品实体
type Product struct {
	ID          int     `ggm:"id,primary_key,auto_increment"`
	Name        string  `ggm:"name,not_null"`
	Price       float64 `ggm:"price,not_null"`
	Description string  `ggm:"description"`
	CategoryID  int     `ggm:"category_id"`
}

func (p Product) TableName() string {
	return "products"
}

func main() {
	// 加载环境变量
	gge.LoadEnv()

	ctx := context.Background()

	// 创建数据库连接池
	pool, err := ygggo_mysql.NewPoolEnv(ctx)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	fmt.Println("=== 表格数据管理器示例 ===")

	// 创建表格数据管理器
	userManager, err := ygggo_mysql.NewTableDataManager(pool, User{})
	if err != nil {
		log.Fatalf("Failed to create user manager: %v", err)
	}

	productManager, err := ygggo_mysql.NewTableDataManager(pool, Product{})
	if err != nil {
		log.Fatalf("Failed to create product manager: %v", err)
	}

	// 创建表（如果不存在）
	err = createTables(pool, ctx)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// 演示用户管理
	fmt.Println("\n--- 用户管理示例 ---")
	demonstrateUserManagement(userManager, ctx)

	// 演示产品管理
	fmt.Println("\n--- 产品管理示例 ---")
	demonstrateProductManagement(productManager, ctx)

	fmt.Println("\n=== 示例完成 ===")
}

func createTables(pool ygggo_mysql.DatabasePool, ctx context.Context) error {
	return pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// 创建用户表
		_, err := c.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT,
			status INT DEFAULT 1
		)`)
		if err != nil {
			return fmt.Errorf("failed to create users table: %v", err)
		}

		// 创建产品表
		_, err = c.Exec(ctx, `CREATE TABLE IF NOT EXISTS products (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			description TEXT,
			category_id INT
		)`)
		if err != nil {
			return fmt.Errorf("failed to create products table: %v", err)
		}

		return nil
	})
}

func demonstrateUserManagement(manager ygggo_mysql.TableDataManager, ctx context.Context) {
	// 1. 添加用户
	fmt.Println("1. 添加用户")
	user1 := User{
		Name:  "张三",
		Email: "zhangsan@example.com",
		Age:   25,
	}

	err := manager.Add(ctx, &user1)
	if err != nil {
		log.Printf("添加用户失败: %v", err)
		return
	}
	fmt.Printf("添加用户成功，ID: %d\n", user1.ID)

	// 2. 批量添加用户
	fmt.Println("\n2. 批量添加用户")
	users := []User{
		{Name: "李四", Email: "lisi@example.com", Age: 30},
		{Name: "王五", Email: "wangwu@example.com", Age: 28},
		{Name: "赵六", Email: "zhaoliu@example.com", Age: 35},
	}

	err = manager.AddMany(ctx, users)
	if err != nil {
		log.Printf("批量添加用户失败: %v", err)
		return
	}
	fmt.Println("批量添加用户成功")

	// 3. 根据ID查询用户
	fmt.Println("\n3. 根据ID查询用户")
	var retrievedUser User
	err = manager.Get(ctx, user1.ID, &retrievedUser)
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return
	}
	fmt.Printf("查询到用户: %+v\n", retrievedUser)

	// 4. 根据条件查询用户
	fmt.Println("\n4. 根据条件查询用户")
	var userByAge User
	err = manager.GetBy(ctx, "age > ?", &userByAge, 25)
	if err != nil {
		log.Printf("条件查询用户失败: %v", err)
		return
	}
	fmt.Printf("年龄大于25的用户: %+v\n", userByAge)

	// 5. 查询所有用户
	fmt.Println("\n5. 查询所有用户")
	var allUsers []User
	err = manager.GetAll(ctx, &allUsers, "")
	if err != nil {
		log.Printf("查询所有用户失败: %v", err)
		return
	}
	fmt.Printf("所有用户数量: %d\n", len(allUsers))
	for _, u := range allUsers {
		fmt.Printf("  - %s (%s), 年龄: %d\n", u.Name, u.Email, u.Age)
	}

	// 6. 分页查询用户
	fmt.Println("\n6. 分页查询用户")
	var pageUsers []User
	err = manager.GetPage(ctx, 1, 2, &pageUsers, "")
	if err != nil {
		log.Printf("分页查询用户失败: %v", err)
		return
	}
	fmt.Printf("第1页用户数量: %d\n", len(pageUsers))

	// 7. 更新用户
	fmt.Println("\n7. 更新用户")
	retrievedUser.Age = 26
	err = manager.Update(ctx, &retrievedUser)
	if err != nil {
		log.Printf("更新用户失败: %v", err)
		return
	}
	fmt.Println("更新用户成功")

	// 8. 根据条件更新用户
	fmt.Println("\n8. 根据条件更新用户")
	updates := map[string]any{"status": 0}
	err = manager.UpdateBy(ctx, "age > ?", updates, 30)
	if err != nil {
		log.Printf("条件更新用户失败: %v", err)
		return
	}
	fmt.Println("条件更新用户成功")

	// 9. 根据条件删除用户
	fmt.Println("\n9. 根据条件删除用户")
	err = manager.DeleteBy(ctx, "status = ?", 0)
	if err != nil {
		log.Printf("条件删除用户失败: %v", err)
		return
	}
	fmt.Println("条件删除用户成功")
}

func demonstrateProductManagement(manager ygggo_mysql.TableDataManager, ctx context.Context) {
	// 1. 添加产品
	fmt.Println("1. 添加产品")
	product := Product{
		Name:        "iPhone 15",
		Price:       999.99,
		Description: "最新款iPhone",
		CategoryID:  1,
	}

	err := manager.Add(ctx, &product)
	if err != nil {
		log.Printf("添加产品失败: %v", err)
		return
	}
	fmt.Printf("添加产品成功，ID: %d\n", product.ID)

	// 2. 批量添加产品
	fmt.Println("\n2. 批量添加产品")
	products := []Product{
		{Name: "MacBook Pro", Price: 1999.99, Description: "专业笔记本", CategoryID: 2},
		{Name: "iPad Air", Price: 599.99, Description: "平板电脑", CategoryID: 3},
		{Name: "Apple Watch", Price: 399.99, Description: "智能手表", CategoryID: 4},
	}

	err = manager.AddMany(ctx, products)
	if err != nil {
		log.Printf("批量添加产品失败: %v", err)
		return
	}
	fmt.Println("批量添加产品成功")

	// 3. 查询所有产品
	fmt.Println("\n3. 查询所有产品")
	var allProducts []Product
	err = manager.GetAll(ctx, &allProducts, "")
	if err != nil {
		log.Printf("查询所有产品失败: %v", err)
		return
	}
	fmt.Printf("所有产品数量: %d\n", len(allProducts))
	for _, p := range allProducts {
		fmt.Printf("  - %s: $%.2f\n", p.Name, p.Price)
	}

	// 4. 根据价格范围查询产品
	fmt.Println("\n4. 根据价格范围查询产品")
	var expensiveProducts []Product
	err = manager.GetAll(ctx, &expensiveProducts, "price > ?", 500)
	if err != nil {
		log.Printf("查询高价产品失败: %v", err)
		return
	}
	fmt.Printf("价格大于$500的产品数量: %d\n", len(expensiveProducts))

	// 5. 更新产品价格
	fmt.Println("\n5. 更新产品价格")
	priceUpdates := map[string]any{"price": 899.99}
	err = manager.UpdateBy(ctx, "name = ?", priceUpdates, "iPhone 15")
	if err != nil {
		log.Printf("更新产品价格失败: %v", err)
		return
	}
	fmt.Println("更新产品价格成功")

	// 6. 删除产品
	fmt.Println("\n6. 删除产品")
	err = manager.Delete(ctx, product.ID)
	if err != nil {
		log.Printf("删除产品失败: %v", err)
		return
	}
	fmt.Println("删除产品成功")
}
