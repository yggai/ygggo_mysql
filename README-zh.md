# ygggo_mysql

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-red.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-available-brightgreen.svg)](docs/)
[![Test Status](https://img.shields.io/badge/tests-passing-brightgreen.svg)]()

中文文档 | [English](README.md)

一个功能完整、生产就绪的 MySQL 数据库访问库，专为企业级应用设计，提供高性能、高可靠性和全面的可观测性支持。基于 TDD 开发规范构建，确保代码质量和功能稳定性。

## ✨ 核心特性

### 🔗 连接管理
- **智能连接池**: 可配置的连接限制和生命周期管理
- **连接泄漏检测**: 自动检测和报告长时间持有的连接
- **健康监控**: 实时监控连接池状态和数据库健康状况
- **自动重连**: 网络故障时的自动重连机制
- **环境变量配置**: 支持从环境变量自动读取配置

### 🗄️ 数据库管理
- **自动创建数据库**: 检测并自动创建不存在的数据库
- **数据库操作**: 查询、创建、删除数据库的完整支持
- **Docker 集成**: 自动管理 Docker MySQL 容器

### 📊 表格管理
- **表结构管理**: 基于结构体标签的表创建、删除和查询
- **ggm 标签支持**: 通过标签声明数据库字段属性
- **自动表结构解析**: 从结构体自动生成表结构

### 📈 表格数据管理
- **完整 CRUD 操作**: 增删改查的全面支持
- **批量操作**: 高效的批量插入、更新和删除
- **条件查询**: 灵活的条件查询和分页支持
- **类型安全**: 基于结构体的类型安全操作

### 📤 数据导入导出
- **多格式支持**: SQL、CSV、JSON 三种格式
- **灵活导出**: 支持单表、多表、全库导出
- **批量导入**: 高性能的批量数据导入
- **条件过滤**: 支持 WHERE 条件的数据导出

### 🔄 事务支持
- **自动事务管理**: 基于函数返回值的自动提交/回滚
- **重试策略**: 死锁和超时的智能重试机制
- **ACID 保证**: 完整的 ACID 事务支持
- **嵌套事务**: 基于保存点的嵌套事务支持

### ⚡ 性能优化
- **预处理语句缓存**: LRU 缓存提升重复查询性能
- **批量操作**: 高效的批量插入和更新操作
- **查询流式处理**: 大结果集的流式处理
- **连接复用**: 高效的连接资源利用

### 📊 可观测性
- **OpenTelemetry 集成**: 分布式链路追踪支持
- **Prometheus 指标**: Prometheus 兼容的指标收集
- **结构化日志**: 可配置的结构化日志记录
- **慢查询分析**: 自动慢查询检测和分析

### 🛠️ 开发体验
- **类型安全**: 强类型查询构建器
- **命名参数**: 命名参数查询绑定支持
- **错误分类**: 详细的错误分类和处理
- **测试支持**: 完整的模拟和测试工具

## 📦 安装

```bash
go get github.com/yggai/ygggo_mysql
```

## 🚀 快速开始

### 基于环境变量的快速连接

```go
package main

import (
    "context"
    "fmt"
    "log"

    gge "github.com/yggai/ygggo_env"
    ggm "github.com/yggai/ygggo_mysql"
)

func main() {
    // 自动查找并加载环境变量
    gge.LoadEnv()

    // 自动读取环境变量里面的值创建数据库连接池对象
    ctx := context.Background()
    pool, err := ggm.NewPoolEnv(ctx)
    if err != nil {
        log.Fatalf("连接失败: %v", err)
    }
    defer pool.Close()

    // 测试连接
    err = pool.Ping(ctx)
    if err != nil {
        log.Fatalf("Ping失败: %v", err)
    }

    fmt.Println("✅ 数据库连接成功!")
    fmt.Println("数据库连接信息：", ggm.GetDSN())
}
```

### 环境变量配置

创建 `.env` 文件：

```bash
YGGGO_MYSQL_HOST=localhost
YGGGO_MYSQL_PORT=3306
YGGGO_MYSQL_USERNAME=root
YGGGO_MYSQL_PASSWORD=password
YGGGO_MYSQL_DATABASE=test
```

### 数据库管理

```go
// 获取数据库连接对象
db, _ := pool.GetDB()

// 查看所有数据库
fmt.Println("所有数据库：", db.GetAllDatabase())

// 添加新的数据库
db.AddDatabase("test2")
fmt.Println("所有数据库：", db.GetAllDatabase())

// 删除数据库
db.DeleteDatabase("test2")
fmt.Println("所有数据库：", db.GetAllDatabase())
```

### 表格管理

```go
// 定义用户表结构
type User struct {
    ID     int    `ggm:"id,primary_key,auto_increment"`
    Name   string `ggm:"name,not_null"`
    Email  string `ggm:"email,unique"`
    Age    int    `ggm:"age"`
    Status int    `ggm:"status,default:1"`
}

func (u User) TableName() string {
    return "users"
}

// 创建表格管理器
tableManager, err := ggm.NewTableManager(pool, User{})
if err != nil {
    log.Fatal(err)
}

// 创建表
err = tableManager.AddTable(ctx)
if err != nil {
    log.Printf("创建表失败: %v", err)
}

// 查看所有表
tables, err := tableManager.GetAllTable(ctx)
if err != nil {
    log.Printf("查询表失败: %v", err)
} else {
    fmt.Println("所有表:", tables)
}
```

### 表格数据管理

```go
// 创建表格数据管理器
userManager, err := ggm.NewTableDataManager(pool, User{})
if err != nil {
    log.Fatal(err)
}

// 添加用户
user := User{
    Name:  "张三",
    Email: "zhangsan@example.com",
    Age:   25,
}

err = userManager.Add(ctx, &user)
if err != nil {
    log.Printf("添加用户失败: %v", err)
} else {
    fmt.Printf("添加用户成功，ID: %d\n", user.ID)
}

// 批量添加用户
users := []User{
    {Name: "李四", Email: "lisi@example.com", Age: 30},
    {Name: "王五", Email: "wangwu@example.com", Age: 28},
}

err = userManager.AddMany(ctx, users)
if err != nil {
    log.Printf("批量添加用户失败: %v", err)
}

// 查询用户
var retrievedUser User
err = userManager.Get(ctx, user.ID, &retrievedUser)
if err != nil {
    log.Printf("查询用户失败: %v", err)
} else {
    fmt.Printf("查询到用户: %+v\n", retrievedUser)
}

// 分页查询
var allUsers []User
err = userManager.GetPage(ctx, 1, 10, &allUsers, "status = ?", 1)
if err != nil {
    log.Printf("分页查询失败: %v", err)
} else {
    fmt.Printf("查询到 %d 个用户\n", len(allUsers))
}
```

### 数据导入导出

```go
// 创建导入导出管理器
exportImportManager := ggm.NewExportImportManager(pool)

// 导出为 SQL 格式
var sqlBuf bytes.Buffer
sqlOptions := ggm.ExportOptions{
    Format: ggm.FormatSQL,
    Output: &sqlBuf,
}

err = exportImportManager.ExportTable(ctx, "users", sqlOptions)
if err != nil {
    log.Printf("导出SQL失败: %v", err)
} else {
    // 保存到文件
    os.WriteFile("users.sql", sqlBuf.Bytes(), 0644)
    fmt.Println("SQL文件已保存")
}

// 导出为 CSV 格式
var csvBuf bytes.Buffer
csvOptions := ggm.ExportOptions{
    Format: ggm.FormatCSV,
    Output: &csvBuf,
}

err = exportImportManager.ExportTable(ctx, "users", csvOptions)
if err != nil {
    log.Printf("导出CSV失败: %v", err)
} else {
    fmt.Println("CSV导出内容:")
    fmt.Println(csvBuf.String())
}

// 从 CSV 导入数据
csvData := `id,name,email,age,status
100,测试用户1,test1@example.com,22,1
101,测试用户2,test2@example.com,24,1`

importOptions := ggm.ImportOptions{
    Format:        ggm.FormatCSV,
    Input:         strings.NewReader(csvData),
    TruncateFirst: false, // 不清空表，追加数据
}

err = exportImportManager.ImportTable(ctx, "users", importOptions)
if err != nil {
    log.Printf("CSV导入失败: %v", err)
} else {
    fmt.Println("CSV数据导入成功")
}
```

## 🔧 高级特性

### Docker MySQL 管理

```go
// 检测 Docker 是否安装
if !ggm.IsDockerInstalled() {
    log.Fatal("Docker 未安装")
}

// 自动安装 MySQL 容器
err := ggm.NewMySQL()
if err != nil {
    log.Printf("安装 MySQL 失败: %v", err)
} else {
    fmt.Println("MySQL 容器安装成功")
}

// 检测 MySQL 是否运行
if ggm.IsMySQL() {
    fmt.Println("MySQL 容器正在运行")
} else {
    fmt.Println("MySQL 容器未运行")
}

// 删除 MySQL 容器
err = ggm.DeleteMySQL()
if err != nil {
    log.Printf("删除 MySQL 失败: %v", err)
} else {
    fmt.Println("MySQL 容器删除成功")
}
```

### 事务支持

```go
// 自动事务管理
err := pool.WithinTx(ctx, func(tx ggm.DatabaseTx) error {
    // 扣款
    result, err := tx.Exec(ctx,
        "UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?",
        amount, fromID, amount)
    if err != nil {
        return err
    }

    affected, _ := result.RowsAffected()
    if affected == 0 {
        return errors.New("余额不足")
    }

    // 入账
    _, err = tx.Exec(ctx,
        "UPDATE accounts SET balance = balance + ? WHERE id = ?",
        amount, toID)
    return err
})
```

### 批量操作

```go
// 高性能批量插入
columns := []string{"name", "email", "age"}
rows := [][]any{
    {"Alice", "alice@example.com", 30},
    {"Bob", "bob@example.com", 25},
    {"Charlie", "charlie@example.com", 35},
}

err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
    result, err := conn.BulkInsert(ctx, "users", columns, rows)
    if err != nil {
        return err
    }

    affected, _ := result.RowsAffected()
    log.Printf("插入了 %d 个用户", affected)
    return nil
})
```

## 📊 性能表现

### 基准测试结果

| 操作类型 | QPS | 平均延迟 | P99 延迟 |
|---------|-----|---------|---------|
| 简单查询 | 50,000+ | 0.2ms | 1ms |
| 事务操作 | 25,000+ | 0.5ms | 2ms |
| 批量插入 | 100,000+ | 0.1ms | 0.5ms |

### 性能优化

- **连接池优化**: 智能连接数量管理
- **预处理语句缓存**: 减少 SQL 解析开销
- **批量操作**: 减少网络往返次数
- **异步日志**: 非阻塞日志记录

## 🧪 测试

本库提供了完整的测试支持，基于 TDD 开发规范构建：

```go
func TestUserService(t *testing.T) {
    // 使用测试助手创建测试环境
    helper, err := ggm.NewTestHelper(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    defer helper.Close()

    // 测试业务逻辑
    service := NewUserService(helper.Pool())
    err = service.CreateUser(ctx, "Alice", "alice@example.com")
    assert.NoError(t, err)
}
```

### 运行测试

```bash
# 运行所有测试（自动管理 MySQL 容器）
go test -timeout 30s

# 运行特定测试
go test -timeout 30s -run TestTableDataManager

# 运行基准测试
go test -bench=. -timeout 30s

# 查看测试覆盖率
go test -cover -timeout 30s
```

### 测试特性

- **自动容器管理**: 自动检测和启动 MySQL 测试容器
- **测试独立性**: 每个测试用例独立运行，不相互影响
- **数据清理**: 自动清理测试数据，确保测试可重复运行
- **30秒超时**: 所有测试在30秒内完成，确保快速反馈

## 📚 文档

- [开发笔记](docs/开发笔记.md) - 详细的开发过程和功能说明
- [API 文档](docs/API文档.md) - 完整的 API 参考
- [使用手册](docs/使用手册.md) - 详细的使用指南
- [官方教程](docs/官方教程.md) - 从入门到精通
- [项目介绍](docs/项目介绍.md) - 项目背景和设计理念
- [商业许可证申请指南](docs/商业许可证申请指南.md) - 商业使用指南

### 示例代码

查看 [examples/](examples/) 目录获取完整的工作示例：

- [Docker MySQL 管理](examples/c01_mysql/main.go) - Docker 容器管理
- [环境变量连接](examples/c02_new_env/main.go) - 基于环境变量的连接
- [数据库管理](examples/c03_database/main.go) - 数据库的增删查操作
- [表格管理](examples/c04_table/main.go) - 表结构的管理操作
- [表格数据管理](examples/c05_table_data/main.go) - 完整的 CRUD 操作
- [数据导入导出](examples/c06_export_import/main.go) - 多格式数据导入导出

## 🌟 使用场景

### 适用场景

- **Web 应用后端**: 高并发 Web 服务
- **微服务架构**: 分布式系统中的数据访问层
- **数据处理服务**: 批量数据处理和 ETL 任务
- **API 网关**: 需要数据库访问的 API 服务
- **企业应用**: 需要可靠数据库访问的企业级应用

### 行业应用

- **电商平台**: 订单处理、库存管理
- **金融服务**: 交易处理、账户管理
- **内容管理**: 用户数据、内容存储
- **物联网**: 设备数据收集和分析
- **数据分析**: 大数据处理和分析平台

### 技术栈集成

- **Gin/Echo**: Web 框架集成
- **gRPC**: 微服务通信
- **Docker**: 容器化部署
- **Kubernetes**: 云原生部署
- **Prometheus**: 监控和指标收集

## 📄 许可证

本项目采用 **PolyForm Noncommercial License 1.0.0** 许可证。

### 🆓 免费使用
- ✅ 个人学习和研究
- ✅ 教育机构教学
- ✅ 非营利组织使用
- ✅ 开源项目贡献

### 💼 商业使用
- ❌ 商业使用需要单独的商业许可证
- 📞 联系我们获取商业许可
- 🏢 提供企业支持和服务

详细信息请查看：
- [许可证 (英文)](LICENSE)
- [许可证 (中文)](LICENSE-zh.md)
- [商业许可证申请指南](docs/商业许可证申请指南.md)

## 🤝 贡献

我们欢迎社区贡献！请查看：
- [贡献指南](CONTRIBUTING.md)
- [代码风格指南](CODE_STYLE.md)
- [问题模板](ISSUE_TEMPLATE.md)

### 开发规范

- **TDD 开发**: 基于测试驱动开发
- **代码覆盖率**: 保持高代码覆盖率
- **文档完整**: 完善的文档和示例
- **性能优化**: 持续的性能优化

## 📞 支持

### 获取帮助

- **文档**: 查看 [完整文档](docs/)
- **示例**: 参考 [examples/](examples/) 目录
- **问题**: 提交 GitHub Issues
- **讨论**: GitHub Discussions

### 商业支持

商业许可和企业支持：

**联系方式**: 1156956636@qq.com
**网站**: zhangdapeng.com
**维护者**: 源滚滚

## 🗺 发展路线

### 当前版本 (v1.0)
- ✅ 核心功能实现
- ✅ 环境变量配置支持
- ✅ Docker MySQL 管理
- ✅ 数据库管理功能
- ✅ 表格管理功能
- ✅ 表格数据管理（完整 CRUD）
- ✅ 数据导入导出（SQL/CSV/JSON）
- ✅ 基础可观测性功能
- ✅ 完整文档和示例
- ✅ 全面测试覆盖

### 下一版本 (v1.1)
- 🔄 API 稳定化
- 🔄 性能优化
- 🔄 更多数据库支持
- 🔄 增强的监控功能

### 未来计划
- 📋 读写分离支持
- 📋 数据库分片支持
- 📋 增强的可观测性功能
- 📋 云原生集成
- 📋 GraphQL 支持
- 📋 更多 ORM 特性

## 🏆 项目特色

### 基于 TDD 开发
- **测试驱动**: 所有功能都基于测试驱动开发
- **高质量代码**: 严格的代码质量标准
- **持续集成**: 自动化测试和部署

### 企业级特性
- **生产就绪**: 经过充分测试，可用于生产环境
- **高性能**: 优化的连接池和查询性能
- **可观测性**: 完整的监控和日志支持
- **易于使用**: 简洁的 API 设计

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者和社区成员！

---

**开始使用 ygggo_mysql 构建高性能数据库应用！**

更多信息请访问我们的 [文档](docs/) 或查看 [示例](examples/)。
