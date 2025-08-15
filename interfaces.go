package ygggo_mysql

import (
	"context"
	"database/sql"
)

// DatabasePool defines the interface that all database pool implementations must satisfy.
// This ensures consistent behavior between mock and real database pools.
type DatabasePool interface {
	// Connection management
	WithConn(ctx context.Context, fn func(DatabaseConn) error) error
	Acquire(ctx context.Context) (DatabaseConn, error)
	
	// Transaction management
	WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error
	
	// Health and lifecycle
	Ping(ctx context.Context) error
	SelfCheck(ctx context.Context) error
	Close() error
}

// DatabaseConn defines the interface that all database connection implementations must satisfy.
// This includes both regular connections and cached/prepared statement functionality.
type DatabaseConn interface {
	// Basic query operations
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryStream(ctx context.Context, query string, cb func([]any) error, args ...any) error
	
	// Cached/prepared statement operations
	EnableStmtCache(capacity int)
	ExecCached(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryCached(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	
	// Named parameter operations
	NamedExec(ctx context.Context, query string, arg any) (sql.Result, error)
	NamedQuery(ctx context.Context, query string, arg any) (*sql.Rows, error)
	
	// Bulk operations
	BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (sql.Result, error)
	InsertOnDuplicate(ctx context.Context, table string, columns []string, rows [][]any, updateCols []string) (sql.Result, error)
	
	// Lifecycle
	Close() error
}

// DatabaseTx defines the interface that all database transaction implementations must satisfy.
type DatabaseTx interface {
	// Basic transaction operations
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	// Note: Transactions typically don't need all the advanced features like caching,
	// named parameters, or bulk operations, but they can be added if needed.
}

// Ensure our concrete types implement the interfaces at compile time
var (
	_ DatabasePool = (*Pool)(nil)
	_ DatabaseConn = (*Conn)(nil)
	_ DatabaseTx   = (*Tx)(nil)
)
