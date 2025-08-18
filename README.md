# ygggo_mysql

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-PolyForm%20Noncommercial-red.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-available-brightgreen.svg)](docs/)
[![Test Status](https://img.shields.io/badge/tests-passing-brightgreen.svg)]()

[中文文档](README-zh.md) | English

A comprehensive, production-ready MySQL database access library for Go, designed for enterprise applications requiring high performance, reliability, and ease of use. Built with TDD (Test-Driven Development) principles to ensure code quality and functional stability. Features deep integration with environment variables for zero-configuration deployment and Docker-based MySQL management.

## ✨ Core Features

### 🔗 Connection Management
- **Smart Connection Pooling**: Configurable connection limits and lifecycle management
- **Connection Leak Detection**: Automatic detection and reporting of long-held connections
- **Health Monitoring**: Real-time monitoring of pool status and database health
- **Auto-Reconnection**: Automatic reconnection on network failures
- **Zero-Configuration Setup**: Deep integration with ygggo_env for automatic environment variable configuration
- **Context-Free API**: Simplified API design with optional context support

### 🗄️ Database Management
- **Auto Database Creation**: Detect and automatically create non-existent databases during connection
- **Database Operations**: Complete support for querying, creating, and deleting databases
- **Docker Integration**: Full Docker MySQL container lifecycle management
- **Environment-Driven Setup**: Automatic MySQL container creation based on environment variables

### 📊 Table Management
- **Table Structure Management**: Create, delete, and query tables based on struct tags
- **ggm Tag Support**: Declare database field attributes through tags
- **Auto Schema Parsing**: Automatically generate table structure from structs

### 📈 Table Data Management
- **Complete CRUD Operations**: Comprehensive support for Create, Read, Update, Delete
- **Batch Operations**: Efficient bulk insert, update, and delete operations
- **Conditional Queries**: Flexible conditional queries and pagination support
- **Type Safety**: Type-safe operations based on structs

### 📤 Data Import/Export
- **Multi-Format Support**: SQL, CSV, JSON formats
- **Flexible Export**: Support for single table, multiple tables, and full database export
- **Batch Import**: High-performance bulk data import
- **Conditional Filtering**: Support for WHERE conditions in data export

### 🔄 Transaction Support
- **Automatic Transaction Management**: Auto-commit/rollback based on function return values
- **Retry Policies**: Intelligent retry mechanisms for deadlocks and timeouts
- **ACID Guarantees**: Complete ACID transaction support
- **Nested Transactions**: Savepoint-based nested transaction support

### ⚡ Performance Optimization
- **Prepared Statement Caching**: LRU cache for improved repeated query performance
- **Bulk Operations**: Efficient bulk insert and update operations
- **Query Streaming**: Streaming processing for large result sets
- **Connection Reuse**: Efficient connection resource utilization

### 📊 Observability
- **Integrated Logging**: Deep integration with ygggo_log for structured logging
- **Slow Query Analysis**: Automatic slow query detection and analysis
- **Performance Monitoring**: Built-in performance metrics and monitoring
- **Connection Health Tracking**: Real-time connection pool health monitoring

### 🛠️ Developer Experience
- **Type Safety**: Strongly-typed query builders
- **Named Parameters**: Named parameter query binding support
- **Error Classification**: Detailed error classification and handling
- **Testing Support**: Complete mocking and testing utilities

## 📦 Installation

```bash
go get github.com/yggai/ygggo_mysql
```

## 🚀 Quick Start

### Environment Variable Based Connection

```go
package main

import (
    "fmt"
    "log"

    gge "github.com/yggai/ygggo_env"
    ggm "github.com/yggai/ygggo_mysql"
)

func main() {
    // Automatically find and load environment variables
    gge.LoadEnv()

    // Create database connection pool from environment variables
    pool, err := ggm.NewPoolEnv()
    if err != nil {
        log.Fatalf("Connection failed: %v", err)
    }
    defer pool.Close()

    // Test connection
    err = pool.Ping()
    if err != nil {
        log.Fatalf("Ping failed: %v", err)
    }

    fmt.Println("✅ Database connection successful!")
    fmt.Println("Database connection info:", ggm.GetDSN())
}
```

### Environment Variable Configuration

Create a `.env` file:

```bash
YGGGO_MYSQL_HOST=localhost
YGGGO_MYSQL_PORT=3306
YGGGO_MYSQL_USERNAME=root
YGGGO_MYSQL_PASSWORD=password
YGGGO_MYSQL_DATABASE=test
```

### Database Management

```go
// Get database connection object
db, _ := pool.GetDB()

// View all databases
fmt.Println("All databases:", db.GetAllDatabase())

// Add new database
db.AddDatabase("test2")
fmt.Println("All databases:", db.GetAllDatabase())

// Delete database
db.DeleteDatabase("test2")
fmt.Println("All databases:", db.GetAllDatabase())
```

### Table Management

```go
// Define user table structure
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

// Create table manager
tableManager, err := ggm.NewTableManager(pool, User{})
if err != nil {
    log.Fatal(err)
}

// Create table
err = tableManager.AddTable(ctx)
if err != nil {
    log.Printf("Failed to create table: %v", err)
}

// View all tables
tables, err := tableManager.GetAllTable(ctx)
if err != nil {
    log.Printf("Failed to query tables: %v", err)
} else {
    fmt.Println("All tables:", tables)
}
```

### Table Data Management

```go
// Create table data manager
userManager, err := ggm.NewTableDataManager(pool, User{})
if err != nil {
    log.Fatal(err)
}

// Add user
user := User{
    Name:  "John Doe",
    Email: "john@example.com",
    Age:   25,
}

err = userManager.Add(ctx, &user)
if err != nil {
    log.Printf("Failed to add user: %v", err)
} else {
    fmt.Printf("User added successfully, ID: %d\n", user.ID)
}

// Batch add users
users := []User{
    {Name: "Jane Smith", Email: "jane@example.com", Age: 30},
    {Name: "Bob Johnson", Email: "bob@example.com", Age: 28},
}

err = userManager.AddMany(ctx, users)
if err != nil {
    log.Printf("Failed to batch add users: %v", err)
}

// Query user
var retrievedUser User
err = userManager.Get(ctx, user.ID, &retrievedUser)
if err != nil {
    log.Printf("Failed to query user: %v", err)
} else {
    fmt.Printf("Retrieved user: %+v\n", retrievedUser)
}

// Paginated query
var allUsers []User
err = userManager.GetPage(ctx, 1, 10, &allUsers, "status = ?", 1)
if err != nil {
    log.Printf("Failed to paginate query: %v", err)
} else {
    fmt.Printf("Found %d users\n", len(allUsers))
}
```

### Data Import/Export

```go
// Create import/export manager
exportImportManager := ggm.NewExportImportManager(pool)

// Export to SQL format
var sqlBuf bytes.Buffer
sqlOptions := ggm.ExportOptions{
    Format: ggm.FormatSQL,
    Output: &sqlBuf,
}

err = exportImportManager.ExportTable(ctx, "users", sqlOptions)
if err != nil {
    log.Printf("Failed to export SQL: %v", err)
} else {
    // Save to file
    os.WriteFile("users.sql", sqlBuf.Bytes(), 0644)
    fmt.Println("SQL file saved")
}

// Export to CSV format
var csvBuf bytes.Buffer
csvOptions := ggm.ExportOptions{
    Format: ggm.FormatCSV,
    Output: &csvBuf,
}

err = exportImportManager.ExportTable(ctx, "users", csvOptions)
if err != nil {
    log.Printf("Failed to export CSV: %v", err)
} else {
    fmt.Println("CSV export content:")
    fmt.Println(csvBuf.String())
}

// Import from CSV data
csvData := `id,name,email,age,status
100,Test User 1,test1@example.com,22,1
101,Test User 2,test2@example.com,24,1`

importOptions := ggm.ImportOptions{
    Format:        ggm.FormatCSV,
    Input:         strings.NewReader(csvData),
    TruncateFirst: false, // Don't truncate table, append data
}

err = exportImportManager.ImportTable(ctx, "users", importOptions)
if err != nil {
    log.Printf("Failed to import CSV: %v", err)
} else {
    fmt.Println("CSV data imported successfully")
}
```

## 🔧 Advanced Features

### Docker MySQL Management

```go
import "context"

ctx := context.Background()

// Check if Docker is installed
if !ggm.IsDockerInstalled(ctx) {
    log.Fatal("Docker not installed")
}

// Automatically install MySQL container
err := ggm.NewMySQL(ctx)
if err != nil {
    log.Printf("Failed to install MySQL: %v", err)
} else {
    fmt.Println("MySQL container installed successfully")
}

// Check if MySQL is running
if ggm.IsMySQL(ctx) {
    fmt.Println("MySQL container is running")
} else {
    fmt.Println("MySQL container is not running")
}

// Delete MySQL container
err = ggm.DeleteMySQL(ctx)
if err != nil {
    log.Printf("Failed to delete MySQL: %v", err)
} else {
    fmt.Println("MySQL container deleted successfully")
}
```

### Transaction Support

```go
// Automatic transaction management
err := pool.WithinTx(ctx, func(tx ggm.DatabaseTx) error {
    // Debit
    result, err := tx.Exec(ctx,
        "UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?",
        amount, fromID, amount)
    if err != nil {
        return err
    }

    affected, _ := result.RowsAffected()
    if affected == 0 {
        return errors.New("insufficient balance")
    }

    // Credit
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

## 📊 Performance

### Benchmark Results

The library includes comprehensive benchmark testing capabilities. Run benchmarks with:

```bash
go test -bench=. -timeout 30s
```

Performance characteristics vary based on hardware, network conditions, and MySQL configuration. The library is optimized for high-throughput scenarios with efficient connection pooling and prepared statement caching.

### Performance Optimizations

- **Connection Pool Optimization**: Smart connection count management
- **Prepared Statement Cache**: Reduces SQL parsing overhead
- **Bulk Operations**: Reduces network round trips
- **Async Logging**: Non-blocking log recording

## 🧪 Testing

The library provides complete testing support, built with TDD principles:

```go
func TestUserService(t *testing.T) {
    // Use test helper to create test environment
    helper, err := ggm.NewTestHelper(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    defer helper.Close()

    // Test business logic
    service := NewUserService(helper.Pool())
    err = service.CreateUser(ctx, "Alice", "alice@example.com")
    assert.NoError(t, err)
}
```

### Running Tests

```bash
# Run all tests (automatically manages MySQL container)
go test -timeout 30s

# Run specific tests
go test -timeout 30s -run TestTableDataManager

# Run benchmarks
go test -bench=. -timeout 30s

# View test coverage
go test -cover -timeout 30s
```

### Testing Features

- **TDD-Based Development**: All features developed following Test-Driven Development principles
- **Automatic Container Management**: Automatically detects and starts MySQL test containers
- **Test Independence**: Each test case runs independently without interference
- **Data Cleanup**: Automatically cleans up test data to ensure repeatable tests
- **30-Second Timeout**: All tests complete within 30 seconds for fast feedback
- **Comprehensive Coverage**: High test coverage ensuring code quality and reliability

## 📚 Documentation

- [Development Notes](docs/开发笔记.md) - Detailed development process and feature descriptions (Chinese)
- [API Documentation](docs/API文档.md) - Complete API reference (Chinese)
- [User Manual](docs/使用手册.md) - Detailed usage guide (Chinese)
- [Official Tutorial](docs/官方教程.md) - From beginner to advanced (Chinese)
- [Project Introduction](docs/项目介绍.md) - Project background and design philosophy (Chinese)
- [Commercial License Guide](docs/商业许可证申请指南.md) - Commercial usage guide (Chinese)

### Example Code

Check out the [examples/](examples/) directory for complete working examples:

- [Docker MySQL Management](examples/c01_mysql/main.go) - Docker container management
- [Environment Variable Connection](examples/c02_new_env/main.go) - Environment-based connection
- [Database Management](examples/c03_database/main.go) - Database CRUD operations
- [Table Management](examples/c04_table/main.go) - Table structure management
- [Table Data Management](examples/c05_table_data/main.go) - Complete CRUD operations
- [Data Import/Export](examples/c06_export_import/main.go) - Multi-format data import/export

## 🌟 Use Cases

### Suitable Scenarios

- **Web Application Backends**: High-concurrency web services
- **Microservice Architecture**: Data access layer in distributed systems
- **Data Processing Services**: Batch data processing and ETL tasks
- **API Gateways**: API services requiring database access
- **Enterprise Applications**: Enterprise-grade applications requiring reliable database access

### Industry Applications

- **E-commerce Platforms**: Order processing, inventory management
- **Financial Services**: Transaction processing, account management
- **Content Management**: User data, content storage
- **IoT**: Device data collection and analysis
- **Data Analytics**: Big data processing and analytics platforms

### Technology Stack Integration

- **Gin/Echo**: Web framework integration
- **gRPC**: Microservice communication
- **Docker**: Containerized deployment
- **Kubernetes**: Cloud-native deployment
- **Prometheus**: Monitoring and metrics collection

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

### Development Standards

- **TDD Development**: Based on Test-Driven Development
- **Code Coverage**: Maintain high code coverage
- **Complete Documentation**: Comprehensive documentation and examples
- **Performance Optimization**: Continuous performance optimization

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

### Current Version (v1.0.2)
- ✅ Core functionality implementation
- ✅ Environment variable configuration support (ygggo_env integration)
- ✅ Docker MySQL management (complete lifecycle)
- ✅ Database management features (auto-creation, CRUD operations)
- ✅ Table management features (ggm tag support)
- ✅ Table data management (complete CRUD with batch operations)
- ✅ Data import/export (SQL/CSV/JSON formats)
- ✅ Integrated logging system (ygggo_log integration)
- ✅ Simplified dependencies (removed OpenTelemetry)
- ✅ Complete documentation and examples
- ✅ Comprehensive test coverage (TDD-based)

### Next Version (v1.1)
- 🔄 API stabilization
- 🔄 Performance optimizations
- 🔄 Additional database support
- 🔄 Enhanced monitoring features

### Future Plans
- 📋 Read/write splitting support
- 📋 Database sharding support
- 📋 Enhanced observability features
- 📋 Cloud-native integrations
- 📋 GraphQL support
- 📋 More ORM features

## 🏆 Project Highlights

### TDD-Based Development
- **Test-Driven**: All features developed with test-driven approach
- **High-Quality Code**: Strict code quality standards and comprehensive test coverage
- **Fast Feedback**: All tests complete within 30 seconds for rapid development cycles

### Enterprise-Grade Features
- **Production-Ready**: Thoroughly tested for production use with automatic database creation
- **Zero-Configuration**: Deep environment variable integration for seamless deployment
- **Docker Integration**: Complete MySQL container lifecycle management
- **Developer-Friendly**: Clean API design with simplified context handling

## 🙏 Acknowledgments

Thanks to all developers and community members who have contributed to this project!

---

**Start using ygggo_mysql to build high-performance database applications!**

For more information, visit our [documentation](docs/) or check out the [examples](examples/).
