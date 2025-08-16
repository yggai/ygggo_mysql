# ygggo_mysql

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-red.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-available-brightgreen.svg)](docs/)

A comprehensive, production-ready MySQL database access library for Go, designed for enterprise applications requiring high performance, reliability, and comprehensive observability.

## 🚀 Features

### 🔗 Connection Management
- **Smart Connection Pooling**: Configurable connection limits and lifecycle management
- **Connection Leak Detection**: Automatic detection and reporting of long-held connections
- **Health Monitoring**: Real-time monitoring of pool status and database health
- **Auto-Reconnection**: Automatic reconnection on network failures

### � Transaction Support
- **Automatic Transaction Management**: Auto-commit/rollback based on function return values
- **Retry Policies**: Intelligent retry mechanisms for deadlocks and timeouts
- **ACID Guarantees**: Complete ACID transaction support
- **Nested Transactions**: Savepoint-based nested transaction support

### ⚡ Performance Optimization
- **Prepared Statement Caching**: LRU cache for improved repeated query performance
- **Bulk Operations**: Efficient bulk insert and update operations
- **Query Streaming**: Streaming processing for large result sets
- **Connection Reuse**: Efficient connection resource utilization

### � Observability
- **OpenTelemetry Integration**: Distributed tracing support
- **Prometheus Metrics**: Prometheus-compatible metrics collection
- **Structured Logging**: Configurable structured logging
- **Slow Query Analysis**: Automatic slow query detection and analysis

### � Developer Experience
- **Type Safety**: Strongly-typed query builders
- **Named Parameters**: Named parameter query binding support
- **Error Classification**: Detailed error classification and handling
- **Testing Support**: Complete mocking and testing utilities

## 📦 Installation

```bash
go get github.com/yggai/ygggo_mysql
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    ggm "github.com/yggai/ygggo_mysql"
)

func main() {
    // Configure database connection
    config := ggm.Config{
        Host:     "localhost",
        Port:     3306,
        Username: "user",
        Password: "password",
        Database: "mydb",
        Driver:   "mysql",
    }

    // Create connection pool
    ctx := context.Background()
    pool, err := ggm.NewPool(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Execute queries safely
    err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
        result, err := conn.Exec(ctx,
            "INSERT INTO users (name, age) VALUES (?, ?)",
            "Alice", 30)
        if err != nil {
            return err
        }

        id, err := result.LastInsertId()
        if err != nil {
            return err
        }

        log.Printf("Created user with ID: %d", id)
        return nil
    })

    if err != nil {
        log.Printf("Operation failed: %v", err)
    }
}
```

### Transaction Example

```go
// Money transfer with ACID guarantees
err := pool.WithinTx(ctx, func(tx ggm.DatabaseTx) error {
    // Debit source account
    result, err := tx.Exec(ctx,
        "UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?",
        amount, fromID, amount)
    if err != nil {
        return err
    }

    affected, _ := result.RowsAffected()
    if affected == 0 {
        return errors.New("insufficient funds")
    }

    // Credit destination account
    _, err = tx.Exec(ctx,
        "UPDATE accounts SET balance = balance + ? WHERE id = ?",
        amount, toID)
    return err
})
```

### Bulk Operations

```go
// High-performance bulk insert
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
    log.Printf("Inserted %d users", affected)
    return nil
})
```

## ⚙️ Configuration

### Programmatic Configuration

```go
config := ggm.Config{
    // Connection settings
    Host:     "localhost",
    Port:     3306,
    Username: "user",
    Password: "password",
    Database: "mydb",

    // Pool configuration
    Pool: ggm.PoolConfig{
        MaxOpen:         25,
        MaxIdle:         10,
        ConnMaxLifetime: 5 * time.Minute,
        ConnMaxIdleTime: 2 * time.Minute,
    },

    // Performance settings
    SlowQueryThreshold: 100 * time.Millisecond,

    // Retry policy
    Retry: ggm.RetryPolicy{
        MaxAttempts: 3,
        BaseBackoff: 10 * time.Millisecond,
    },
}
```

### Environment Variables

All configuration can be overridden using environment variables:

```bash
export YGGGO_MYSQL_HOST=localhost
export YGGGO_MYSQL_PORT=3306
export YGGGO_MYSQL_USERNAME=user
export YGGGO_MYSQL_PASSWORD=secret
export YGGGO_MYSQL_DATABASE=mydb
```

## 🔧 Advanced Features

### Connection Leak Detection

```go
pool.SetBorrowWarnThreshold(30 * time.Second)
pool.SetLeakHandler(func(leak ggm.BorrowLeak) {
    log.Printf("WARN: Connection held for %v", leak.HeldFor)
    // Send alert, increment metrics, etc.
})
```

### Prepared Statement Caching

```go
err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
    conn.EnableStmtCache(100) // Cache up to 100 prepared statements

    // Subsequent calls will reuse prepared statements
    for i := 0; i < 1000; i++ {
        _, err := conn.ExecCached(ctx,
            "INSERT INTO logs (message) VALUES (?)",
            fmt.Sprintf("Log entry %d", i))
        if err != nil {
            return err
        }
    }
    return nil
})
```

### Query Streaming

```go
err = pool.WithConn(ctx, func(conn ggm.DatabaseConn) error {
    return conn.QueryStream(ctx, "SELECT * FROM large_table",
        func(row []any) error {
            // Process each row without loading entire result set
            log.Printf("Processing row: %v", row)
            return nil
        })
})
```

## 📊 Performance

### Benchmark Results

| Operation Type | QPS | Avg Latency | P99 Latency |
|---------------|-----|-------------|-------------|
| Simple Query | 50,000+ | 0.2ms | 1ms |
| Transaction | 25,000+ | 0.5ms | 2ms |
| Bulk Insert | 100,000+ | 0.1ms | 0.5ms |

### Performance Optimizations

- **Connection Pool Optimization**: Smart connection count management
- **Prepared Statement Cache**: Reduces SQL parsing overhead
- **Bulk Operations**: Reduces network round trips
- **Async Logging**: Non-blocking log recording

## 🧪 Testing

The library provides interfaces that make testing easy:

```go
func TestUserService(t *testing.T) {
    // Use mock implementation for testing
    mockPool := &MockDatabasePool{}
    service := NewUserService(mockPool)

    // Test business logic without real database
    err := service.CreateUser(ctx, "Alice", "alice@example.com")
    assert.NoError(t, err)
}
```

### Running Tests

```bash
# Run all tests (requires Docker)
go test

# Run quick tests (skip Docker tests)
go test -short

# Run specific tests
go test -run TestPoolBasic

# Run benchmarks
go test -bench=.
```

## 📚 Documentation

- [API Documentation](docs/API文档.md) (Chinese)
- [User Manual](docs/使用手册.md) (Chinese)
- [Official Tutorial](docs/官方教程.md) (Chinese)
- [Project Introduction](docs/项目介绍.md) (Chinese)
- [Commercial License Guide](docs/商业许可证申请指南.md) (Chinese)

### Examples

Check out the [examples/](examples/) directory for complete working examples:

- [Basic Connection](examples/c01_connect/main.go)
- [Database Operations](examples/c02_database/main.go)
- [Table Operations](examples/c03_table/main.go)
- [Table Data Operations](examples/c03_table_data/main.go)
- [Transaction Examples](examples/c04_tx/main.go)

## 🌟 Use Cases

### Suitable Scenarios

- **Web Application Backends**: High-concurrency web services
- **Microservice Architecture**: Data access layer in distributed systems
- **Data Processing Services**: Batch data processing and ETL tasks
- **API Gateways**: API services requiring database access

### Industry Applications

- **E-commerce Platforms**: Order processing, inventory management
- **Financial Services**: Transaction processing, account management
- **Content Management**: User data, content storage
- **IoT**: Device data collection and analysis

## 📄 License

This project is licensed under the **PolyForm Noncommercial License 1.0.0**.

### 🆓 Free Use
- ✅ Personal learning and research
- ✅ Educational institution teaching
- ✅ Non-profit organization use
- ✅ Open source project contributions

### 💼 Commercial Use
- ❌ Commercial use requires a separate commercial license
- 📞 Contact us for commercial licensing
- 🏢 Enterprise support and services available

For detailed information, see:
- [License (English)](LICENSE)
- [License (Chinese)](LICENSE-zh.md)
- [Commercial License Guide](docs/商业许可证申请指南.md)

## 🤝 Contributing

We welcome community contributions! Please see:
- [Contributing Guidelines](CONTRIBUTING.md)
- [Code Style Guide](CODE_STYLE.md)
- [Issue Templates](ISSUE_TEMPLATE.md)

## 📞 Support

### Getting Help

- **Documentation**: Check [complete documentation](docs/)
- **Examples**: Refer to [examples/](examples/) directory
- **Issues**: Submit GitHub Issues
- **Discussions**: GitHub Discussions

### Commercial Support

For commercial licensing and enterprise support:

**Contact**: 1156956636@qq.com
**Website**: zhangdapeng.com
**Maintainer**: 源滚滚

## 🗺 Roadmap

### Current Version (v0.x)
- ✅ Core functionality implementation
- ✅ Basic observability features
- ✅ Documentation and examples

### Next Version (v1.0)
- 🔄 API stabilization
- 🔄 Performance optimizations
- 🔄 Additional database support

### Future Plans
- 📋 Read/write splitting support
- 📋 Database sharding support
- 📋 Enhanced observability features
- 📋 Cloud-native integrations

## 🙏 Acknowledgments

Thanks to all developers and community members who have contributed to this project!

---

**Start using ygggo_mysql to build high-performance database applications!**

For more information, visit our [documentation](docs/) or check out the [examples](examples/).
