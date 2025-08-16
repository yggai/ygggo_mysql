# ygggo_mysql Documentation

## 📚 查看文档的方式

### 1. 命令行查看文档

```bash
# 查看包的完整文档
go doc -all

# 查看特定类型的文档
go doc Pool
go doc Config
go doc DatabaseConn

# 查看特定方法的文档
go doc Pool.WithConn
go doc Pool.WithinTx
go doc Pool.NewPool
```

### 2. 启动本地文档服务器

```bash
# 启动 godoc 服务器（需要先安装 godoc）
go install golang.org/x/tools/cmd/godoc@latest
godoc -http=:6060

# 然后在浏览器中访问：
# http://localhost:6060/pkg/github.com/yggai/ygggo_mysql/
```

### 3. 在线查看文档

如果项目发布到 GitHub，可以在以下地址查看：
- https://pkg.go.dev/github.com/yggai/ygggo_mysql

## 📖 文档结构

### 包级别文档
- **ygggo_mysql.go**: 主包文档，包含库的概述、快速开始指南和最佳实践
- **doc.go**: 详细的包文档，包含完整的使用示例和高级功能说明

### 核心接口文档
- **interfaces.go**: 定义了所有核心接口（DatabasePool、DatabaseConn、DatabaseTx）
- 每个接口方法都有详细的参数说明、返回值说明和使用示例

### 主要类型文档
- **Pool**: 连接池的完整文档，包括配置、使用方法和最佳实践
- **Config**: 配置结构的详细说明，包括所有字段和环境变量覆盖
- **Conn**: 数据库连接的文档，包括所有查询方法和缓存功能
- **Tx**: 事务的文档，包括ACID特性和重试策略

## 🎯 文档特色

### 1. 完整的代码示例
每个主要功能都包含可运行的代码示例：
- 基本连接和查询
- 事务处理
- 批量操作
- 连接池配置
- 错误处理

### 2. 最佳实践指导
- 性能优化建议
- 安全性考虑
- 错误处理模式
- 资源管理

### 3. 企业级功能说明
- 连接泄漏检测
- 慢查询分析
- 可观测性集成
- 健康检查

### 4. 配置指南
- 环境变量配置
- 程序化配置
- 生产环境推荐配置

## 🚀 快速开始

查看包文档中的快速开始部分：

```bash
go doc ygggo_mysql
```

或者查看 examples/ 目录中的实际示例：

```bash
# 基本连接示例
go run examples/c01_connect/main.go

# 数据库操作示例
go run examples/c02_database/main.go

# 表操作示例
go run examples/c03_table/main.go

# 表数据操作示例
go run examples/c03_table_data/main.go

# 事务示例
go run examples/c04_tx/main.go
```

## 📝 文档贡献

如果发现文档有需要改进的地方，欢迎：
1. 提交 Issue 报告文档问题
2. 提交 Pull Request 改进文档
3. 建议添加更多示例

## 🔍 搜索文档

使用 `go doc` 命令可以快速搜索和查看特定功能的文档：

```bash
# 搜索包含 "transaction" 的文档
go doc -all | grep -i transaction

# 查看所有以 "With" 开头的方法
go doc -all | grep "func.*With"

# 查看所有配置相关的类型
go doc -all | grep -i config
```

这样的文档结构确保了开发者可以通过多种方式快速找到所需的信息，无论是通过命令行工具、在线文档还是IDE集成。
