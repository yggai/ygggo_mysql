package ygggo_mysql

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestExportUser 测试用的用户实体
type TestExportUser struct {
	ID        int    `ggm:"id,primary_key,auto_increment"`
	Name      string `ggm:"name,not_null"`
	Email     string `ggm:"email,unique"`
	Age       int    `ggm:"age"`
	Status    int    `ggm:"status,default:1"`
	tableName string // 动态表名
}

// TableName 实现表名接口
func (u TestExportUser) TableName() string {
	if u.tableName != "" {
		return u.tableName
	}
	return "test_export_users"
}

func TestNewExportImportManager(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 测试创建导入导出管理器
	manager := NewExportImportManager(helper.Pool())
	if manager == nil {
		t.Fatal("manager should not be nil")
	}
}

func TestExportImportManager_ExportTable_SQL(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建测试表
	tableName := "export_users_sql_test"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// 创建表并插入测试数据
	err = setupTestTable(helper.Pool(), ctx, tableName)
	if err != nil {
		t.Fatalf("Setup test table failed: %v", err)
	}

	// 创建管理器
	manager := NewExportImportManager(helper.Pool())

	// 测试导出为SQL格式
	var buf bytes.Buffer
	options := ExportOptions{
		Format: FormatSQL,
		Output: &buf,
	}

	err = manager.ExportTable(ctx, tableName, options)
	if err != nil {
		t.Fatalf("ExportTable failed: %v", err)
	}

	// 验证导出内容
	output := buf.String()
	if !strings.Contains(output, "CREATE TABLE") {
		t.Error("Expected CREATE TABLE statement in SQL export")
	}
	if !strings.Contains(output, "INSERT INTO") {
		t.Error("Expected INSERT INTO statement in SQL export")
	}
}

func TestExportImportManager_ExportTable_CSV(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建测试表
	tableName := "export_users_csv_test"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// 创建表并插入测试数据
	err = setupTestTable(helper.Pool(), ctx, tableName)
	if err != nil {
		t.Fatalf("Setup test table failed: %v", err)
	}

	// 创建管理器
	manager := NewExportImportManager(helper.Pool())

	// 测试导出为CSV格式
	var buf bytes.Buffer
	options := ExportOptions{
		Format: FormatCSV,
		Output: &buf,
	}

	err = manager.ExportTable(ctx, tableName, options)
	if err != nil {
		t.Fatalf("ExportTable failed: %v", err)
	}

	// 验证导出内容
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Error("Expected at least header and one data row in CSV export")
	}

	// 检查CSV头部
	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "name") {
		t.Error("Expected CSV header with column names")
	}
}

func TestExportImportManager_ExportTable_JSON(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建测试表
	tableName := "export_users_json_test"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// 创建表并插入测试数据
	err = setupTestTable(helper.Pool(), ctx, tableName)
	if err != nil {
		t.Fatalf("Setup test table failed: %v", err)
	}

	// 创建管理器
	manager := NewExportImportManager(helper.Pool())

	// 测试导出为JSON格式
	var buf bytes.Buffer
	options := ExportOptions{
		Format: FormatJSON,
		Output: &buf,
	}

	err = manager.ExportTable(ctx, tableName, options)
	if err != nil {
		t.Fatalf("ExportTable failed: %v", err)
	}

	// 验证导出内容
	output := buf.String()
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Error("Expected JSON format in export")
	}
	if !strings.Contains(output, "\"id\"") || !strings.Contains(output, "\"name\"") {
		t.Error("Expected JSON fields in export")
	}
}

func TestExportImportManager_ImportTable_CSV(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建测试表
	tableName := "import_users_csv_test"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// 创建空表
	err = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
		_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
		_, err := c.Exec(ctx, `CREATE TABLE `+tableName+` (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT,
			status INT DEFAULT 1
		)`)
		return err
	})
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	// 创建管理器
	manager := NewExportImportManager(helper.Pool())

	// 准备CSV数据
	csvData := `id,name,email,age,status
1,张三,zhangsan@example.com,25,1
2,李四,lisi@example.com,30,1
3,王五,wangwu@example.com,28,0`

	// 测试导入CSV数据
	options := ImportOptions{
		Format:        FormatCSV,
		Input:         strings.NewReader(csvData),
		TruncateFirst: true,
	}

	err = manager.ImportTable(ctx, tableName, options)
	if err != nil {
		t.Fatalf("ImportTable failed: %v", err)
	}

	// 验证导入的数据
	var count int
	err = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
		row := c.QueryRow(ctx, "SELECT COUNT(*) FROM "+tableName)
		return row.Scan(&count)
	})
	if err != nil {
		t.Fatalf("Count query failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}
}

func TestExportImportManager_ExportTables(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建两个测试表
	table1 := "export_users_multi_test1"
	table2 := "export_users_multi_test2"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+table1)
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+table2)
			return nil
		})
	}()

	// 创建表并插入测试数据
	err = setupTestTable(helper.Pool(), ctx, table1)
	if err != nil {
		t.Fatalf("Setup test table1 failed: %v", err)
	}

	err = setupTestTable(helper.Pool(), ctx, table2)
	if err != nil {
		t.Fatalf("Setup test table2 failed: %v", err)
	}

	// 创建管理器
	manager := NewExportImportManager(helper.Pool())

	// 测试导出多个表为JSON格式
	var buf bytes.Buffer
	options := ExportOptions{
		Format: FormatJSON,
		Output: &buf,
	}

	err = manager.ExportTables(ctx, []string{table1, table2}, options)
	if err != nil {
		t.Fatalf("ExportTables failed: %v", err)
	}

	// 验证导出内容
	output := buf.String()
	if !strings.Contains(output, table1) || !strings.Contains(output, table2) {
		t.Error("Expected both table names in JSON export")
	}
}

// setupTestTable 创建测试表并插入数据
func setupTestTable(pool DatabasePool, ctx context.Context, tableName string) error {
	return pool.WithConn(ctx, func(c DatabaseConn) error {
		// 删除已存在的表
		_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)

		// 创建表
		_, err := c.Exec(ctx, `CREATE TABLE `+tableName+` (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT,
			status INT DEFAULT 1
		)`)
		if err != nil {
			return err
		}

		// 插入测试数据
		_, err = c.Exec(ctx, `INSERT INTO `+tableName+` (name, email, age, status) VALUES 
			('张三', 'zhangsan@example.com', 25, 1),
			('李四', 'lisi@example.com', 30, 1),
			('王五', 'wangwu@example.com', 28, 0)`)

		return err
	})
}
