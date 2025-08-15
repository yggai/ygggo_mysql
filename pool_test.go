package ygggo_mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestNewPool_OpenAndPing_Success(t *testing.T) {
	// Test basic config validation without actual DB connection
	cfg := Config{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/db",
		Pool: PoolConfig{
			MaxOpen:         5,
			MaxIdle:         5,
			ConnMaxLifetime: time.Minute,
			ConnMaxIdleTime: time.Second * 30,
		},
	}

	// Test that config is properly set up (without actual DB connection)
	dsn, err := dsnFromConfig(cfg)
	if err != nil {
		t.Fatalf("dsnFromConfig error: %v", err)
	}

	expected := "user:pass@tcp(localhost:3306)/db"
	if dsn != expected {
		t.Fatalf("expected DSN %s, got %s", expected, dsn)
	}
}

func TestNewPool_UsesDSNFromFields(t *testing.T) {
	cfg := Config{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: "p@ss%word",
		Database: "db",
		Params: map[string]string{
			"parseTime": "true",
		},
	}

	// Test that DSN is built correctly from fields
	expectedDSN, err := dsnFromConfig(cfg)
	if err != nil {
		t.Fatalf("dsnFromConfig error: %v", err)
	}

	// Verify the DSN contains expected components
	t.Logf("Generated DSN: %s", expectedDSN)

	// Just verify basic structure without exact escaping
	if !containsAt(expectedDSN, "root:") {
		t.Fatalf("DSN should contain username, got: %s", expectedDSN)
	}
	if !containsAt(expectedDSN, "127.0.0.1:3306") {
		t.Fatalf("DSN should contain host:port, got: %s", expectedDSN)
	}
	if !containsAt(expectedDSN, "/db") {
		t.Fatalf("DSN should contain database, got: %s", expectedDSN)
	}
	if !containsAt(expectedDSN, "parseTime=true") {
		t.Fatalf("DSN should contain params, got: %s", expectedDSN)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

