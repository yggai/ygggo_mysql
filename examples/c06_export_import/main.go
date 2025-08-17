package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	ygggo_mysql "github.com/yggai/ygggo_mysql"
	gge "github.com/yggai/ygggo_env"
)

// User 用户实体
type User struct {
	ID     int    `ggm:"id,primary_key,auto_increment"`
	Name   string `ggm:"name,not_null"`
	Email  string `ggm:"email,unique"`
	Age    int    `ggm:"age"`
	Status int    `ggm:"status,default:1"`
}

func (u User) TableName() string {
	return "demo_users"
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
	return "demo_products"
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

	fmt.Println("=== 数据导入导出管理器示例 ===")

	// 创建导入导出管理器
	exportImportManager := ygggo_mysql.NewExportImportManager(pool)

	// 创建表格数据管理器（用于准备测试数据）
	userManager, err := ygggo_mysql.NewTableDataManager(pool, User{})
	if err != nil {
		log.Fatalf("Failed to create user manager: %v", err)
	}

	productManager, err := ygggo_mysql.NewTableDataManager(pool, Product{})
	if err != nil {
		log.Fatalf("Failed to create product manager: %v", err)
	}

	// 创建表并准备测试数据
	err = setupDemoData(pool, userManager, productManager, ctx)
	if err != nil {
		log.Fatalf("Failed to setup demo data: %v", err)
	}

	// 演示导出功能
	fmt.Println("\n--- 导出功能演示 ---")
	demonstrateExport(exportImportManager, ctx)

	// 演示导入功能
	fmt.Println("\n--- 导入功能演示 ---")
	demonstrateImport(exportImportManager, ctx)

	fmt.Println("\n=== 示例完成 ===")
}

func setupDemoData(pool ygggo_mysql.DatabasePool, userManager, productManager ygggo_mysql.TableDataManager, ctx context.Context) error {
	// 创建表
	err := pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		// 创建用户表
		_, err := c.Exec(ctx, `CREATE TABLE IF NOT EXISTS demo_users (
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
		_, err = c.Exec(ctx, `CREATE TABLE IF NOT EXISTS demo_products (
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
	if err != nil {
		return err
	}

	// 清空现有数据
	pool.WithConn(ctx, func(c ygggo_mysql.DatabaseConn) error {
		c.Exec(ctx, "DELETE FROM demo_users")
		c.Exec(ctx, "DELETE FROM demo_products")
		return nil
	})

	// 添加用户数据
	users := []User{
		{Name: "张三", Email: "zhangsan@example.com", Age: 25, Status: 1},
		{Name: "李四", Email: "lisi@example.com", Age: 30, Status: 1},
		{Name: "王五", Email: "wangwu@example.com", Age: 28, Status: 0},
		{Name: "赵六", Email: "zhaoliu@example.com", Age: 35, Status: 1},
	}

	err = userManager.AddMany(ctx, users)
	if err != nil {
		return fmt.Errorf("failed to add users: %v", err)
	}

	// 添加产品数据
	products := []Product{
		{Name: "iPhone 15", Price: 999.99, Description: "最新款iPhone", CategoryID: 1},
		{Name: "MacBook Pro", Price: 1999.99, Description: "专业笔记本", CategoryID: 2},
		{Name: "iPad Air", Price: 599.99, Description: "平板电脑", CategoryID: 3},
		{Name: "Apple Watch", Price: 399.99, Description: "智能手表", CategoryID: 4},
	}

	err = productManager.AddMany(ctx, products)
	if err != nil {
		return fmt.Errorf("failed to add products: %v", err)
	}

	fmt.Println("演示数据准备完成")
	return nil
}

func demonstrateExport(manager ygggo_mysql.ExportImportManager, ctx context.Context) {
	// 1. 导出单个表为SQL格式
	fmt.Println("\n1. 导出用户表为SQL格式")
	var sqlBuf bytes.Buffer
	sqlOptions := ygggo_mysql.ExportOptions{
		Format: ygggo_mysql.FormatSQL,
		Output: &sqlBuf,
	}

	err := manager.ExportTable(ctx, "demo_users", sqlOptions)
	if err != nil {
		log.Printf("导出SQL失败: %v", err)
		return
	}

	fmt.Println("SQL导出内容预览:")
	sqlContent := sqlBuf.String()
	lines := strings.Split(sqlContent, "\n")
	for i, line := range lines {
		if i < 10 { // 只显示前10行
			fmt.Printf("  %s\n", line)
		} else if i == 10 {
			fmt.Println("  ...")
			break
		}
	}

	// 保存到文件
	err = os.WriteFile("demo_users.sql", sqlBuf.Bytes(), 0644)
	if err != nil {
		log.Printf("保存SQL文件失败: %v", err)
	} else {
		fmt.Println("SQL文件已保存为: demo_users.sql")
	}

	// 2. 导出单个表为CSV格式
	fmt.Println("\n2. 导出产品表为CSV格式")
	var csvBuf bytes.Buffer
	csvOptions := ygggo_mysql.ExportOptions{
		Format: ygggo_mysql.FormatCSV,
		Output: &csvBuf,
	}

	err = manager.ExportTable(ctx, "demo_products", csvOptions)
	if err != nil {
		log.Printf("导出CSV失败: %v", err)
		return
	}

	fmt.Println("CSV导出内容:")
	fmt.Println(csvBuf.String())

	// 保存到文件
	err = os.WriteFile("demo_products.csv", csvBuf.Bytes(), 0644)
	if err != nil {
		log.Printf("保存CSV文件失败: %v", err)
	} else {
		fmt.Println("CSV文件已保存为: demo_products.csv")
	}

	// 3. 导出多个表为JSON格式
	fmt.Println("\n3. 导出多个表为JSON格式")
	var jsonBuf bytes.Buffer
	jsonOptions := ygggo_mysql.ExportOptions{
		Format: ygggo_mysql.FormatJSON,
		Output: &jsonBuf,
	}

	err = manager.ExportTables(ctx, []string{"demo_users", "demo_products"}, jsonOptions)
	if err != nil {
		log.Printf("导出JSON失败: %v", err)
		return
	}

	fmt.Println("JSON导出内容预览:")
	jsonContent := jsonBuf.String()
	lines = strings.Split(jsonContent, "\n")
	for i, line := range lines {
		if i < 15 { // 只显示前15行
			fmt.Printf("  %s\n", line)
		} else if i == 15 {
			fmt.Println("  ...")
			break
		}
	}

	// 保存到文件
	err = os.WriteFile("demo_database.json", jsonBuf.Bytes(), 0644)
	if err != nil {
		log.Printf("保存JSON文件失败: %v", err)
	} else {
		fmt.Println("JSON文件已保存为: demo_database.json")
	}

	// 4. 导出整个数据库
	fmt.Println("\n4. 导出整个数据库为SQL格式")
	var dbBuf bytes.Buffer
	dbOptions := ygggo_mysql.ExportOptions{
		Format: ygggo_mysql.FormatSQL,
		Output: &dbBuf,
	}

	err = manager.Export(ctx, dbOptions)
	if err != nil {
		log.Printf("导出数据库失败: %v", err)
		return
	}

	// 保存到文件
	err = os.WriteFile("full_database.sql", dbBuf.Bytes(), 0644)
	if err != nil {
		log.Printf("保存数据库SQL文件失败: %v", err)
	} else {
		fmt.Println("完整数据库已导出为: full_database.sql")
	}
}

func demonstrateImport(manager ygggo_mysql.ExportImportManager, ctx context.Context) {
	// 1. 从CSV导入数据
	fmt.Println("\n1. 从CSV导入新用户数据")
	
	// 准备CSV数据
	csvData := `id,name,email,age,status
100,测试用户1,test1@example.com,22,1
101,测试用户2,test2@example.com,24,1
102,测试用户3,test3@example.com,26,0`

	csvOptions := ygggo_mysql.ImportOptions{
		Format:        ygggo_mysql.FormatCSV,
		Input:         strings.NewReader(csvData),
		TruncateFirst: false, // 不清空表，追加数据
	}

	err := manager.ImportTable(ctx, "demo_users", csvOptions)
	if err != nil {
		log.Printf("CSV导入失败: %v", err)
		return
	}

	fmt.Println("CSV数据导入成功")

	// 2. 验证导入结果
	fmt.Println("\n2. 验证导入结果")
	var exportBuf bytes.Buffer
	exportOptions := ygggo_mysql.ExportOptions{
		Format:      ygggo_mysql.FormatCSV,
		Output:      &exportBuf,
		WhereClause: "id >= 100", // 只查看新导入的数据
	}

	err = manager.ExportTable(ctx, "demo_users", exportOptions)
	if err != nil {
		log.Printf("验证导出失败: %v", err)
		return
	}

	fmt.Println("新导入的用户数据:")
	fmt.Println(exportBuf.String())
}
