# ygggo_mysql
Go语言操作MySQL的底层核心框架

## 特性

- 🚀 **高性能连接池管理** - 智能连接复用和生命周期管理
- 📊 **全面的性能监控** - 内置指标收集和慢查询分析
- 🔧 **灵活的配置选项** - 支持多种连接配置和优化参数
- 🧪 **完整的测试支持** - 基于Docker的集成测试框架
- 📈 **性能基准测试** - 内置基准测试工具和报告生成
- 🔍 **慢查询记录** - 自动检测和分析慢查询
- 🛡️ **生产就绪** - 经过充分测试，适用于生产环境

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "log"
    "github.com/yggai/ygggo_mysql"
)

func main() {
    ctx := context.Background()

    // 创建连接池
    pool, err := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{
        Host:     "localhost",
        Port:     3306,
        Database: "mydb",
        Username: "user",
        Password: "password",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // 使用连接执行查询
    err = pool.WithConn(ctx, func(conn ygggo_mysql.DatabaseConn) error {
        rows, err := conn.Query(ctx, "SELECT id, name FROM users WHERE age > ?", 18)
        if err != nil {
            return err
        }
        defer rows.Close()

        for rows.Next() {
            var id int
            var name string
            if err := rows.Scan(&id, &name); err != nil {
                return err
            }
            log.Printf("User: %d - %s", id, name)
        }
        return rows.Err()
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## 测试

本项目使用Docker-in-test进行集成测试，确保与真实MySQL环境的兼容性。

### 运行测试

```bash
# 运行所有测试（需要Docker）
go test

# 运行快速测试（跳过Docker测试）
go test -short

# 运行特定测试
go test -run TestPoolBasic

# 运行基准测试
go test -bench=.
```

### 测试要求

- Docker环境（用于集成测试）
- Go 1.21+

## 文档

- [性能基准测试](docs/性能基准测试.md)
- [慢查询记录](docs/慢查询记录.md)

## 许可证

MIT License
