# ygggo_mysql API 文档

## 📚 概述

本文档详细介绍了 ygggo_mysql 库的所有公开 API，包括接口、类型、方法和配置选项。

## 🔌 核心接口

### DatabasePool 接口

数据库连接池的核心接口，提供连接管理和事务支持。

```go
type DatabasePool interface {
    // 连接管理
    WithConn(ctx context.Context, fn func(DatabaseConn) error) error
    Acquire(ctx context.Context) (DatabaseConn, error)
    
    // 事务管理
    WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error
    
    // 健康检查和生命周期
    Ping(ctx context.Context) error
    SelfCheck(ctx context.Context) error
    Close() error
}
```

#### 方法详解

##### WithConn
```go
func WithConn(ctx context.Context, fn func(DatabaseConn) error) error
```
**功能**: 自动管理数据库连接的执行函数  
**参数**:
- `ctx`: 上下文，用于取消和超时控制
- `fn`: 要执行的函数，接收一个数据库连接

**返回值**: 错误信息（如果有）

**示例**:
```go
err := pool.WithConn(ctx, func(conn DatabaseConn) error {
    result, err := conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
    return err
})
```

##### WithinTx
```go
func WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error
```
**功能**: 在事务中执行函数，自动处理提交和回滚  
**参数**:
- `ctx`: 上下文
- `fn`: 事务函数
- `opts`: 可选的事务选项

**示例**:
```go
err := pool.WithinTx(ctx, func(tx DatabaseTx) error {
    _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - ? WHERE id = ?", 100, fromID)
    if err != nil {
        return err
    }
    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + ? WHERE id = ?", 100, toID)
    return err
})
```

### DatabaseConn 接口

数据库连接接口，提供所有数据库操作方法。

```go
type DatabaseConn interface {
    // 基本查询操作
    Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRow(ctx context.Context, query string, args ...any) *sql.Row
    Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryStream(ctx context.Context, query string, cb func([]any) error, args ...any) error
    
    // 缓存/预编译语句操作
    EnableStmtCache(capacity int)
    ExecCached(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryCached(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    
    // 命名参数操作
    NamedExec(ctx context.Context, query string, arg any) (sql.Result, error)
    NamedQuery(ctx context.Context, query string, arg any) (*sql.Rows, error)
    
    // 批量操作
    BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (sql.Result, error)
    InsertOnDuplicate(ctx context.Context, table string, columns []string, rows [][]any, updateCols []string) (sql.Result, error)
    
    // 生命周期
    Close() error
}
```

#### 核心方法

##### Query
```go
func Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
```
**功能**: 执行返回多行结果的查询  
**返回值**: sql.Rows 和错误信息

##### Exec
```go
func Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
```
**功能**: 执行不返回行的语句（INSERT、UPDATE、DELETE）  
**返回值**: sql.Result 和错误信息

##### BulkInsert
```go
func BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (sql.Result, error)
```
**功能**: 批量插入数据  
**参数**:
- `table`: 表名
- `columns`: 列名数组
- `rows`: 数据行数组

**示例**:
```go
columns := []string{"name", "age", "email"}
rows := [][]any{
    {"Alice", 25, "alice@example.com"},
    {"Bob", 30, "bob@example.com"},
}
result, err := conn.BulkInsert(ctx, "users", columns, rows)
```

### DatabaseTx 接口

事务接口，提供事务范围内的数据库操作。

```go
type DatabaseTx interface {
    Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
}
```

## 🏗 核心类型

### Pool

连接池的主要实现类型。

```go
type Pool struct {
    // 内部字段（不可直接访问）
}
```

#### 构造函数

##### NewPool
```go
func NewPool(ctx context.Context, cfg Config) (*Pool, error)
```
**功能**: 创建新的连接池  
**参数**:
- `ctx`: 上下文
- `cfg`: 配置对象

**示例**:
```go
config := Config{
    Host:     "localhost",
    Port:     3306,
    Username: "user",
    Password: "password",
    Database: "mydb",
}
pool, err := NewPool(ctx, config)
```

#### 方法

##### SetBorrowWarnThreshold
```go
func (p *Pool) SetBorrowWarnThreshold(d time.Duration)
```
**功能**: 设置连接持有时间警告阈值

##### SetLeakHandler
```go
func (p *Pool) SetLeakHandler(h func(BorrowLeak))
```
**功能**: 设置连接泄漏处理函数

### Config

配置结构体，包含所有连接和功能配置。

```go
type Config struct {
    // 连接配置
    Driver   string
    DSN      string
    Host     string
    Port     int
    Username string
    Password string
    Database string
    Params   map[string]string
    
    // 功能配置
    Pool               PoolConfig
    Retry              RetryPolicy
    Telemetry          TelemetryConfig
    SlowQueryThreshold time.Duration
}
```

#### 字段说明

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| Driver | string | 数据库驱动 | "mysql" |
| Host | string | 数据库主机 | "localhost" |
| Port | int | 数据库端口 | 3306 |
| Username | string | 用户名 | - |
| Password | string | 密码 | - |
| Database | string | 数据库名 | - |

### PoolConfig

连接池配置结构体。

```go
type PoolConfig struct {
    MaxOpen         int
    MaxIdle         int
    ConnMaxLifetime time.Duration
    ConnMaxIdleTime time.Duration
}
```

#### 推荐配置

```go
// 生产环境配置
poolConfig := PoolConfig{
    MaxOpen:         25,
    MaxIdle:         10,
    ConnMaxLifetime: 5 * time.Minute,
    ConnMaxIdleTime: 2 * time.Minute,
}

// 开发环境配置
poolConfig := PoolConfig{
    MaxOpen:         10,
    MaxIdle:         5,
    ConnMaxLifetime: 1 * time.Minute,
    ConnMaxIdleTime: 30 * time.Second,
}
```

## 🔧 工具函数

### BuildIn
```go
func BuildIn(query string, slice any, others ...any) (string, []any, error)
```
**功能**: 构建 IN 查询语句  
**示例**:
```go
ids := []int{1, 2, 3, 4}
query, args, err := BuildIn("SELECT * FROM users WHERE id IN (?)", ids)
// 结果: "SELECT * FROM users WHERE id IN (?,?,?,?)", [1,2,3,4]
```

### Get
```go
func Get[T any](ctx context.Context, c *Conn, dest *T, query string, args ...any) error
```
**功能**: 查询单行数据并扫描到结构体

### Select
```go
func Select[T any](ctx context.Context, c *Conn, dest *[]T, query string, args ...any) error
```
**功能**: 查询多行数据并扫描到结构体切片

## 📊 监控和指标

### 健康检查

```go
// 基本健康检查
status, err := pool.HealthCheck(ctx)

// 深度健康检查
status, err := pool.DeepHealthCheck(ctx)
```

### 连接池统计

```go
stats := pool.GetPoolStats()
fmt.Printf("活跃连接: %d, 空闲连接: %d", stats.ActiveConnections, stats.IdleConnections)
```

### 慢查询记录

```go
// 启用慢查询记录
config := SlowQueryConfig{
    Enabled:   true,
    Threshold: 100 * time.Millisecond,
}
pool.EnableSlowQueryRecording(config, storage)

// 获取慢查询记录
recorder := pool.GetSlowQueryRecorder()
records, err := recorder.GetRecords(ctx, filter)
```

## 🚨 错误处理

### 错误类型

ygggo_mysql 提供了详细的错误分类：

```go
// 连接错误
if isConnectionError(err) {
    // 处理连接问题
}

// 重试错误
if isRetryableError(err) {
    // 可以重试的错误
}

// 约束错误
if isDuplicateKeyError(err) {
    // 重复键错误
}
```

### 重试策略

```go
retryPolicy := RetryPolicy{
    MaxAttempts: 3,
    BaseBackoff: 10 * time.Millisecond,
    MaxBackoff:  1 * time.Second,
    Jitter:      true,
}
```

## 🔐 安全特性

### SQL 注入防护

所有查询方法都支持参数化查询：

```go
// ✅ 安全的方式
result, err := conn.Exec(ctx, "SELECT * FROM users WHERE id = ?", userID)

// ❌ 不安全的方式（避免）
query := fmt.Sprintf("SELECT * FROM users WHERE id = %d", userID)
```

### 连接加密

```go
config := Config{
    Params: map[string]string{
        "tls": "true",
    },
}
```

## 📈 性能优化

### 预编译语句缓存

```go
err := pool.WithConn(ctx, func(conn DatabaseConn) error {
    conn.EnableStmtCache(100) // 缓存100个预编译语句
    
    // 重复执行的查询会自动使用缓存
    for i := 0; i < 1000; i++ {
        _, err := conn.ExecCached(ctx, "INSERT INTO logs (message) VALUES (?)", 
            fmt.Sprintf("Log %d", i))
        if err != nil {
            return err
        }
    }
    return nil
})
```

### 批量操作

```go
// 批量插入比单条插入快10-100倍
columns := []string{"name", "email"}
rows := make([][]any, 1000)
for i := 0; i < 1000; i++ {
    rows[i] = []any{fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i)}
}
result, err := conn.BulkInsert(ctx, "users", columns, rows)
```

## 🧪 测试支持

### 模拟接口

```go
type MockPool struct {
    // 实现 DatabasePool 接口
}

func (m *MockPool) WithConn(ctx context.Context, fn func(DatabaseConn) error) error {
    // 模拟实现
    return fn(&MockConn{})
}
```

### 测试工具

```go
func TestUserService(t *testing.T) {
    mockPool := &MockPool{}
    service := NewUserService(mockPool)
    
    err := service.CreateUser(ctx, "Alice", "alice@example.com")
    assert.NoError(t, err)
}
```

---

**更多详细信息请参考 [Go 文档](https://pkg.go.dev/github.com/yggai/ygggo_mysql) 或使用 `go doc` 命令查看。**
