package ygggo_mysql

import (
	"context"
	"testing"
	"time"
)

// Validate example c01 pattern with WithTimeout
func TestExampleC01_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "pass")

	pool, err := NewPoolEnv(ctx)
	if err != nil { t.Fatalf("NewPoolEnv err: %v", err) }
	t.Cleanup(func(){ _ = pool.Close() })
	if err := pool.Ping(ctx); err != nil { t.Fatalf("Ping err: %v", err) }
}

// Validate example c02 pattern with WithTimeout and DB manager
func TestExampleC02_WithTimeout_DBManager(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	enhancedFakeDriverInstance.databases = map[string]bool{"mysql": true}

	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "pass")

	pool, err := NewPoolEnv(ctx)
	if err != nil { t.Fatalf("NewPoolEnv err: %v", err) }
	t.Cleanup(func(){ _ = pool.Close() })

	mgr, err := pool.GetDB()
	if err != nil { t.Fatalf("GetDB err: %v", err) }
	// list, add, delete
	_ = mgr.GetAllDatabase()
	mgr.AddDatabase("tmp_x")
	mgr.DeleteDatabase("tmp_x")
}

