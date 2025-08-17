package ygggo_mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

// fakeDriver is a minimal sql driver that always succeeds
// so we can test NewPoolEnv without a real DB.
type fakeDriver struct{}

type fakeConn struct{}

func (d fakeDriver) Open(name string) (driver.Conn, error) { return fakeConn{}, nil }

// Implement driver.Conn
func (fakeConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (fakeConn) Close() error                              { return nil }
func (fakeConn) Begin() (driver.Tx, error)                 { return nil, nil }

// Implement driver.Pinger so db.Ping() succeeds
func (fakeConn) Ping(ctx context.Context) error { return nil }

// Implement driver.SessionResetter to satisfy possible checks (no-op)
func (fakeConn) ResetSession(ctx context.Context) error { return nil }

// Implement driver.QueryerContext for QueryRowContext support
func (fakeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return &fakeRows{}, nil
}

// Implement driver.ExecerContext for ExecContext support
func (fakeConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return &fakeResult{}, nil
}

// Simple fake rows implementation
type fakeRows struct{ closed bool }

func (r *fakeRows) Columns() []string              { return []string{} }
func (r *fakeRows) Close() error                   { r.closed = true; return nil }
func (r *fakeRows) Next(dest []driver.Value) error { return fmt.Errorf("EOF") }

// Simple fake result implementation
type fakeResult struct{}

func (r *fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeResult) RowsAffected() (int64, error) { return 0, nil }

// Register the fake driver once
func init() { sql.Register("fake", fakeDriver{}) }

func TestNewPoolEnv_UsesEnvDSNAndSetsLastDSN(t *testing.T) {
	ctx := context.Background()
	// Ensure clean slate
	lastUsedDSN.Store("")

	const envDSN = "envuser:envpass@tcp(127.0.0.1:3307)/envdb?parseTime=true"
	t.Setenv("YGGGO_MYSQL_DRIVER", "fake")
	t.Setenv("YGGGO_MYSQL_DSN", envDSN)

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		t.Fatalf("NewPoolEnv error: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if got := GetDSN(); got != envDSN {
		t.Fatalf("GetDSN mismatch: want %q, got %q", envDSN, got)
	}
}

func TestNewPoolEnv_BuildsDSNFromEnvFields(t *testing.T) {
	ctx := context.Background()
	// Ensure clean slate
	lastUsedDSN.Store("")

	t.Setenv("YGGGO_MYSQL_DRIVER", "fake")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "p@ss:word/!")
	t.Setenv("YGGGO_MYSQL_DATABASE", "") // No database specified to avoid auto-creation
	t.Setenv("YGGGO_MYSQL_PARAMS", "parseTime=true&loc=Local")

	pool, err := NewPoolEnv(ctx)
	if err != nil {
		t.Fatalf("NewPoolEnv error: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	dsn := GetDSN()
	if dsn == "" {
		t.Fatalf("expected non-empty DSN")
	}
	// Validate DSN structure via mysql.ParseDSN for correctness
	mc, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN err: %v, dsn=%q", err, dsn)
	}
	if mc.User != "root" {
		t.Fatalf("user=%q", mc.User)
	}
	if mc.Passwd != "p@ss:word/!" {
		t.Fatalf("passwd=%q", mc.Passwd)
	}
	if mc.Addr != "127.0.0.1:3306" {
		t.Fatalf("addr=%q", mc.Addr)
	}
	if mc.DBName != "" {
		t.Fatalf("db should be empty, got=%q", mc.DBName)
	}
	if !mc.ParseTime {
		t.Fatalf("parseTime expected true")
	}
	if mc.Loc == nil || mc.Loc.String() != "Local" {
		t.Fatalf("loc expected Local, got %#v", mc.Loc)
	}
}
