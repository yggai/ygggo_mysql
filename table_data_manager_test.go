package ygggo_mysql

import (
	"context"
	"testing"
	"time"
)

// 测试用的实体结构体
type User struct {
	ID       int       `ggm:"id,primary_key,auto_increment"`
	Name     string    `ggm:"name,not_null"`
	Email    string    `ggm:"email,unique"`
	Age      int       `ggm:"age"`
	Status   int       `ggm:"status,default:1"`
	CreateAt time.Time `ggm:"create_at,default:CURRENT_TIMESTAMP"`
	UpdateAt time.Time `ggm:"update_at,default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

// TableName 实现表名接口
func (u User) TableName() string {
	return "users"
}

// Product 另一个测试实体
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

func TestNewTableDataManager(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 测试创建表数据管理器
	manager, err := NewTableDataManager(helper.Pool(), User{})
	if err != nil {
		t.Fatalf("NewTableDataManager failed: %v", err)
	}

	if manager == nil {
		t.Fatal("manager should not be nil")
	}
}

// TestUser 测试专用的User结构体
type TestUser struct {
	ID        int    `ggm:"id,primary_key,auto_increment"`
	Name      string `ggm:"name,not_null"`
	Email     string `ggm:"email,unique"`
	Age       int    `ggm:"age"`
	Status    int    `ggm:"status,default:1"`
	tableName string // 动态表名
}

// TableName 实现表名接口
func (u TestUser) TableName() string {
	if u.tableName != "" {
		return u.tableName
	}
	return "test_users"
}

func TestTableDataManager_Add(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建测试表
	tableName := "users_add_test"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// 创建表
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

	// 创建测试用户实例，设置表名
	testUser := TestUser{tableName: tableName}

	// 创建管理器
	manager, err := NewTableDataManager(helper.Pool(), testUser)
	if err != nil {
		t.Fatalf("NewTableDataManager failed: %v", err)
	}

	// 测试添加用户
	user := TestUser{
		Name:      "John Doe",
		Email:     "john@example.com",
		Age:       30,
		tableName: tableName,
	}

	err = manager.Add(ctx, &user)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// 验证用户ID被设置
	if user.ID == 0 {
		t.Fatal("User ID should be set after insert")
	}
}

func TestTableDataManager_Get(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 创建测试表
	tableName := "users_get_test"
	ctx := context.Background()

	defer func() {
		_ = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
			_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			return nil
		})
	}()

	// 创建表
	err = helper.Pool().WithConn(ctx, func(c DatabaseConn) error {
		_, _ = c.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
		_, err := c.Exec(ctx, `CREATE TABLE `+tableName+` (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT,
			status INT DEFAULT 1,
			create_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`)
		return err
	})
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	// 创建管理器
	manager, err := NewTableDataManager(helper.Pool(), User{})
	if err != nil {
		t.Fatalf("NewTableDataManager failed: %v", err)
	}

	// 先添加一个用户
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	err = manager.Add(ctx, &user)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// 测试Get方法
	var retrievedUser User
	err = manager.Get(ctx, user.ID, &retrievedUser)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// 验证数据
	if retrievedUser.ID != user.ID {
		t.Errorf("Expected ID %d, got %d", user.ID, retrievedUser.ID)
	}
	if retrievedUser.Name != user.Name {
		t.Errorf("Expected Name %s, got %s", user.Name, retrievedUser.Name)
	}
	if retrievedUser.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, retrievedUser.Email)
	}
}

func TestTableDataManager_Update(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 这个测试将在实现Update方法后完成
	t.Skip("Update method not implemented yet")
}

func TestTableDataManager_Delete(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 这个测试将在实现Delete方法后完成
	t.Skip("Delete method not implemented yet")
}

func TestTableDataManager_GetPage(t *testing.T) {
	helper, err := NewTestHelper(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer helper.Close()

	// 这个测试将在实现GetPage方法后完成
	t.Skip("GetPage method not implemented yet")
}
