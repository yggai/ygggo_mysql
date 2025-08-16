package ygggo_mysql

import (
	"context"
	"database/sql"
)

// DatabasePool defines the interface that all database pool implementations must satisfy.
//
// This interface ensures consistent behavior between mock and real database pools,
// enabling easy testing and dependency injection. It provides connection management,
// transaction handling, and health monitoring capabilities.
//
// Example usage:
//
//	var pool DatabasePool = myPool
//	err := pool.WithConn(ctx, func(conn DatabaseConn) error {
//		return conn.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
//	})
type DatabasePool interface {
	// WithConn executes a function with a database connection.
	//
	// The connection is automatically acquired from the pool and returned
	// when the function completes, regardless of success or failure.
	// This is the recommended way to execute database operations.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - fn: Function to execute with the connection
	//
	// Returns an error if connection acquisition fails or if fn returns an error.
	WithConn(ctx context.Context, fn func(DatabaseConn) error) error

	// Acquire obtains a connection from the pool.
	//
	// The caller is responsible for calling Close() on the returned connection
	// to return it to the pool. Use WithConn() for automatic connection management.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//
	// Returns a DatabaseConn or an error if acquisition fails.
	Acquire(ctx context.Context) (DatabaseConn, error)

	// WithinTx executes a function within a database transaction.
	//
	// The transaction is automatically committed if fn returns nil,
	// or rolled back if fn returns an error. Supports configurable
	// retry policies for handling transient failures.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - fn: Function to execute within the transaction
	//   - opts: Optional transaction options (implementation-specific)
	//
	// Returns an error if transaction setup fails or if fn returns an error.
	WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error

	// Ping verifies connectivity to the database.
	//
	// This method sends a simple query to the database to verify
	// that the connection is still alive and responsive.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//
	// Returns an error if the database is unreachable or unresponsive.
	Ping(ctx context.Context) error

	// SelfCheck performs a comprehensive health check.
	//
	// This method performs more thorough health checks than Ping(),
	// potentially including connection pool status, query performance,
	// and other health indicators.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//
	// Returns an error if any health check fails.
	SelfCheck(ctx context.Context) error

	// Close closes the pool and all its connections.
	//
	// After calling Close(), the pool should not be used for new operations.
	// Any ongoing operations may be interrupted.
	//
	// Returns an error if cleanup fails.
	Close() error
}

// DatabaseConn defines the interface that all database connection implementations must satisfy.
//
// This interface provides a comprehensive set of database operations including
// basic queries, prepared statement caching, named parameters, bulk operations,
// and streaming capabilities. It abstracts the underlying database connection
// to enable testing and different implementation strategies.
//
// Example usage:
//
//	// Basic query
//	rows, err := conn.Query(ctx, "SELECT * FROM users WHERE age > ?", 18)
//
//	// Bulk insert
//	columns := []string{"name", "age"}
//	data := [][]any{{"Alice", 25}, {"Bob", 30}}
//	result, err := conn.BulkInsert(ctx, "users", columns, data)
type DatabaseConn interface {
	// Query executes a query that returns rows.
	//
	// The query should use placeholder parameters (?) to prevent SQL injection.
	// The caller is responsible for closing the returned *sql.Rows.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - args: Arguments to bind to placeholders
	//
	// Returns *sql.Rows for iteration or an error if the query fails.
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRow executes a query that is expected to return at most one row.
	//
	// The query should use placeholder parameters (?) to prevent SQL injection.
	// QueryRow always returns a non-nil value. Errors are deferred until
	// Row's Scan method is called.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - args: Arguments to bind to placeholders
	//
	// Returns *sql.Row for scanning the result.
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row

	// Exec executes a query without returning any rows.
	//
	// This is typically used for INSERT, UPDATE, DELETE, and DDL statements.
	// The query should use placeholder parameters (?) to prevent SQL injection.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - args: Arguments to bind to placeholders
	//
	// Returns sql.Result containing information about the execution or an error.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	// QueryStream executes a query and streams results through a callback function.
	//
	// This method is useful for processing large result sets without loading
	// all rows into memory at once. The callback function is called for each row.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - cb: Callback function that receives each row as []any
	//   - args: Arguments to bind to placeholders
	//
	// Returns an error if the query fails or if the callback returns an error.
	QueryStream(ctx context.Context, query string, cb func([]any) error, args ...any) error

	// EnableStmtCache enables prepared statement caching with the specified capacity.
	//
	// Prepared statement caching can significantly improve performance for
	// frequently executed queries by avoiding repeated parsing and planning.
	//
	// Parameters:
	//   - capacity: Maximum number of prepared statements to cache
	EnableStmtCache(capacity int)

	// ExecCached executes a query using cached prepared statements when available.
	//
	// If statement caching is enabled and a prepared statement exists for the query,
	// it will be reused. Otherwise, the query is executed normally.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - args: Arguments to bind to placeholders
	//
	// Returns sql.Result containing information about the execution or an error.
	ExecCached(ctx context.Context, query string, args ...any) (sql.Result, error)

	// QueryCached executes a query using cached prepared statements when available.
	//
	// If statement caching is enabled and a prepared statement exists for the query,
	// it will be reused. Otherwise, the query is executed normally.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - args: Arguments to bind to placeholders
	//
	// Returns *sql.Rows for iteration or an error if the query fails.
	QueryCached(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// NamedExec executes a query with named parameters.
	//
	// Named parameters use the format :name in the query string and are bound
	// from struct fields or map keys in the arg parameter.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with named parameters (:name)
	//   - arg: Struct or map containing parameter values
	//
	// Returns sql.Result containing information about the execution or an error.
	NamedExec(ctx context.Context, query string, arg any) (sql.Result, error)

	// NamedQuery executes a query with named parameters that returns rows.
	//
	// Named parameters use the format :name in the query string and are bound
	// from struct fields or map keys in the arg parameter.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with named parameters (:name)
	//   - arg: Struct or map containing parameter values
	//
	// Returns *sql.Rows for iteration or an error if the query fails.
	NamedQuery(ctx context.Context, query string, arg any) (*sql.Rows, error)

	// BulkInsert performs a bulk insert operation using a single multi-value INSERT statement.
	//
	// This method is optimized for inserting multiple rows efficiently by
	// constructing a single INSERT statement with multiple value sets.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - table: Target table name
	//   - columns: Column names for the insert
	//   - rows: Data rows, where each row must have the same length as columns
	//
	// Returns sql.Result containing information about the insertion or an error.
	BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (sql.Result, error)

	// InsertOnDuplicate performs a bulk insert with ON DUPLICATE KEY UPDATE.
	//
	// This method combines bulk insert with update behavior for handling
	// duplicate key conflicts. When a duplicate key is encountered,
	// the specified columns are updated instead of causing an error.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - table: Target table name
	//   - columns: Column names for the insert
	//   - rows: Data rows, where each row must have the same length as columns
	//   - updateCols: Columns to update on duplicate key conflicts
	//
	// Returns sql.Result containing information about the operation or an error.
	InsertOnDuplicate(ctx context.Context, table string, columns []string, rows [][]any, updateCols []string) (sql.Result, error)

	// Close closes the connection and returns it to the pool.
	//
	// After calling Close(), the connection should not be used for further operations.
	// This method should always be called to prevent connection leaks.
	//
	// Returns an error if closing the connection fails.
	Close() error
}

// DatabaseTx defines the interface that all database transaction implementations must satisfy.
//
// This interface provides transaction-scoped database operations. Transactions
// ensure ACID properties (Atomicity, Consistency, Isolation, Durability) for
// a group of database operations. All operations within a transaction either
// succeed together or fail together.
//
// Example usage:
//
//	err := pool.WithinTx(ctx, func(tx DatabaseTx) error {
//		_, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - ? WHERE id = ?", 100, fromID)
//		if err != nil {
//			return err // This will cause a rollback
//		}
//		_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + ? WHERE id = ?", 100, toID)
//		return err // If nil, transaction commits; if error, transaction rolls back
//	})
//
// Note: Transactions are typically kept simple and focused. Advanced features
// like statement caching, named parameters, or bulk operations are usually
// not needed within transactions, but can be added to implementations if required.
type DatabaseTx interface {
	// Exec executes a query without returning any rows within the transaction.
	//
	// This is typically used for INSERT, UPDATE, DELETE, and DDL statements
	// within a transaction context. The query should use placeholder parameters (?)
	// to prevent SQL injection.
	//
	// All operations within the same transaction see a consistent view of the
	// database, and changes are not visible to other transactions until commit.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - query: SQL query with placeholder parameters
	//   - args: Arguments to bind to placeholders
	//
	// Returns sql.Result containing information about the execution or an error.
	// If an error is returned, the transaction should be rolled back.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Ensure our concrete types implement the interfaces at compile time
var (
	_ DatabasePool = (*Pool)(nil)
	_ DatabaseConn = (*Conn)(nil)
	_ DatabaseTx   = (*Tx)(nil)
)
