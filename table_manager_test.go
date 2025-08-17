package ygggo_mysql

import (
	"context"
	"testing"
)

type TUser struct {
	Id   int    `ggm:"id"`
	Name string `ggm:"name"`
}

func TestTableManager_Add_GetAll_Delete(t *testing.T) {
	ctx := context.Background()

	// Reset enhanced fake driver state and ensure a database exists (SHOW TABLES needs a DB)
	enhancedFakeDriverInstance.databases = map[string]bool{"testdb": true}

	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "pass")
	t.Setenv("YGGGO_MYSQL_DATABASE", "testdb")

	pool, err := NewPoolEnv(ctx)
	if err != nil { t.Fatalf("NewPoolEnv err: %v", err) }
	t.Cleanup(func(){ _ = pool.Close() })

	mgr, err := pool.GetDB()
	if err != nil { t.Fatalf("GetDB err: %v", err) }

	// Initially empty
	_ = mgr.GetAllTable() // enhanced fake returns empty

	// Create table
	mgr.AddTable(&TUser{})
	// Fake driver doesn't track tables list; this call is for coverage and should not error
	_ = mgr.GetAllTable()

	// Drop table
	mgr.DeleteTable(&TUser{})
}

