package ygggo_mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewPool_OpenAndPing_Success(t *testing.T) {
	// Arrange: register DSN with sqlmock so sql.Open(driver, dsn) succeeds
	dsn := "sqlmock_dsn"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.NewWithDSN error: %v", err)
	}
	defer db.Close()

	// Expect Ping from NewPool
	mock.ExpectPing()

	cfg := Config{
		Driver: "sqlmock",
		DSN:    dsn,
		Pool: PoolConfig{
			MaxOpen:         5,
			MaxIdle:         5,
			ConnMaxLifetime: time.Minute,
			ConnMaxIdleTime: time.Second * 30,
		},
	}

	// Act
	ctx := context.Background()
	p, err := NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPool error: %v", err)
	}
	defer p.Close()

	// Replace internal db with our mocked db so subsequent checks hit the mock.
	p.db = db

	// Expect two more pings: one for Ping, one for SelfCheck
	mock.ExpectPing()
	mock.ExpectPing()

	if err := p.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if err := p.SelfCheck(ctx); err != nil {
		t.Fatalf("SelfCheck failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNewPool_UsesDSNFromFields(t *testing.T) {
	cfg := Config{
		Driver:   "sqlmock",
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: "p@ss%word",
		Database: "db",
		Params: map[string]string{
			"parseTime": "true",
		},
	}
	expectedDSN, err := dsnFromConfig(cfg)
	if err != nil {
		t.Fatalf("dsnFromConfig error: %v", err)
	}

	// Register DSN with sqlmock so NewPool(sql.Open) attaches to this mock.
	_, mock, err := sqlmock.NewWithDSN(expectedDSN, sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.NewWithDSN error: %v", err)
	}
	mock.ExpectPing()

	ctx := context.Background()
	p, err := NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPool error: %v", err)
	}
	defer p.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}


func TestNewPool_OpenFails(t *testing.T) {
	// Arrange: invalid driver should fail
	cfg := Config{Driver: "nonexist", DSN: ""}
	ctx := context.Background()
	if _, err := NewPool(ctx, cfg); err == nil {
		t.Fatal("expected error for invalid driver, got nil")
	}
}

// Minimal stubs to make pool.db assignable in tests
func (p *Pool) setDB(db *sql.DB) { p.db = db }

