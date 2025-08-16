# Query Builder

The Query Builder provides a fluent, type-safe interface for building SQL queries in ygggo_mysql. It supports SELECT, INSERT, UPDATE, and DELETE operations with a chainable API that makes complex queries easy to read and maintain.

## Features

- **Fluent Interface**: Chainable methods for building queries
- **Type Safety**: Compile-time safety for query construction
- **Parameter Binding**: Automatic SQL injection protection
- **Full SQL Support**: SELECT, INSERT, UPDATE, DELETE with JOINs, WHERE, ORDER BY, GROUP BY, HAVING
- **Integration**: Works seamlessly with existing DatabaseConn interface

## Basic Usage

### Creating a Query Builder

```go
qb := ygggo.NewQueryBuilder(conn)
```

### SELECT Queries

#### Basic SELECT
```go
rows, err := qb.Select("id", "name", "email").
    From("users").
    Query(ctx)
```

#### SELECT with WHERE conditions
```go
rows, err := qb.Select("name", "age").
    From("users").
    Where("age > ?", 25).
    Where("status = ?", "active").
    Query(ctx)
```

#### SELECT with ORDER BY and LIMIT
```go
rows, err := qb.Select("name", "created_at").
    From("users").
    OrderBy("created_at DESC").
    Limit(10).
    Query(ctx)
```

#### SELECT with JOINs
```go
rows, err := qb.Select("u.name", "d.name AS department").
    From("users u").
    Join("INNER JOIN departments d ON u.department_id = d.id").
    Where("d.budget > ?", 100000).
    Query(ctx)
```

#### SELECT with GROUP BY and HAVING
```go
rows, err := qb.Select("department_id", "COUNT(*) as user_count").
    From("users").
    GroupBy("department_id").
    Having("COUNT(*) > ?", 2).
    Query(ctx)
```

### INSERT Queries

#### Basic INSERT
```go
result, err := qb.Insert("users").
    Values(map[string]any{
        "name":  "John Doe",
        "email": "john@example.com",
        "age":   30,
    }).
    Exec(ctx)
```

### UPDATE Queries

#### Basic UPDATE
```go
result, err := qb.Update("users").
    Set("age", 31).
    Set("email", "john.doe@example.com").
    Where("id = ?", 1).
    Exec(ctx)
```

### DELETE Queries

#### Basic DELETE
```go
result, err := qb.Delete("users").
    Where("age < ?", 18).
    Exec(ctx)
```

## API Reference

### Query Builder Methods

#### SELECT Operations
- `Select(columns ...string)` - Specify columns to select
- `From(table string)` - Specify the table
- `Join(joinClause string)` - Add JOIN clause
- `Where(condition string, args ...any)` - Add WHERE condition
- `GroupBy(columns ...string)` - Add GROUP BY clause
- `Having(condition string, args ...any)` - Add HAVING condition
- `OrderBy(orderBy string)` - Add ORDER BY clause
- `Limit(limit int)` - Set LIMIT
- `Offset(offset int)` - Set OFFSET
- `Query(ctx context.Context)` - Execute and return rows

#### INSERT Operations
- `Insert(table string)` - Start INSERT query
- `Values(values map[string]any)` - Set values to insert
- `Exec(ctx context.Context)` - Execute and return result

#### UPDATE Operations
- `Update(table string)` - Start UPDATE query
- `Set(column string, value any)` - Set column value
- `Where(condition string, args ...any)` - Add WHERE condition
- `Exec(ctx context.Context)` - Execute and return result

#### DELETE Operations
- `Delete(table string)` - Start DELETE query
- `Where(condition string, args ...any)` - Add WHERE condition
- `Exec(ctx context.Context)` - Execute and return result

## Examples

### Complex Query Example
```go
// Find users in high-budget departments with their manager info
rows, err := qb.Select(
    "u.name",
    "u.email", 
    "d.name AS department",
    "m.name AS manager",
).
From("users u").
Join("INNER JOIN departments d ON u.department_id = d.id").
Join("LEFT JOIN users m ON u.manager_id = m.id").
Where("d.budget > ?", 500000).
Where("u.status = ?", "active").
OrderBy("d.name ASC", "u.name ASC").
Limit(50).
Query(ctx)
```

### Batch Operations
```go
// Insert multiple users
users := []map[string]any{
    {"name": "Alice", "email": "alice@example.com", "age": 25},
    {"name": "Bob", "email": "bob@example.com", "age": 30},
}

for _, user := range users {
    _, err := qb.Insert("users").Values(user).Exec(ctx)
    if err != nil {
        return err
    }
}
```

## Best Practices

1. **Always use parameter binding**: Never concatenate user input directly into queries
2. **Close rows**: Always defer `rows.Close()` when using `Query()`
3. **Handle errors**: Check errors from both `Query()`/`Exec()` and `Scan()`
4. **Use transactions**: For multiple related operations, use transactions
5. **Limit results**: Use `Limit()` for potentially large result sets

## Error Handling

```go
rows, err := qb.Select("*").From("users").Query(ctx)
if err != nil {
    return fmt.Errorf("query failed: %w", err)
}
defer rows.Close()

for rows.Next() {
    var user User
    if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
        return fmt.Errorf("scan failed: %w", err)
    }
    // Process user...
}

if err := rows.Err(); err != nil {
    return fmt.Errorf("rows iteration failed: %w", err)
}
```

## Integration with Existing Features

The Query Builder integrates seamlessly with existing ygggo_mysql features:

- **Connection Pooling**: Works with any `DatabaseConn`
- **Metrics & Telemetry**: Automatically instrumented when enabled
- **Prepared Statements**: Can be combined with cached queries
- **Transactions**: Use within transaction contexts

```go
// Use with connection pool
conn, err := pool.Acquire(ctx)
if err != nil {
    return err
}
defer conn.Close()

qb := ygggo.NewQueryBuilder(conn)
// ... use query builder

// Use within transaction
err = pool.WithinTx(ctx, func(tx ygggo.DatabaseTx) error {
    // Note: Query builder currently works with DatabaseConn
    // For transactions, use regular SQL or extend the builder
    return tx.Exec(ctx, "INSERT INTO ...", args...)
})
```
