package ygggo_mysql

import (
	"context"
	"testing"
)

func TestDatabaseManager_GetAll_Add_Delete(t *testing.T) {
	ctx := context.Background()

	// reset enhanced fake driver state
	enhancedFakeDriverInstance.databases = map[string]bool{
		"mysql":              true,
		"information_schema": true,
	}

	// env for pool
	t.Setenv("YGGGO_MYSQL_DRIVER", "enhanced_fake")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "pass")
	// do not require specific database

	pool, err := NewPoolEnv(ctx)
	if err != nil { t.Fatalf("NewPoolEnv err: %v", err) }
	t.Cleanup(func() { _ = pool.Close() })

	db, err := pool.GetDB()
	if err != nil { t.Fatalf("GetDB err: %v", err) }

	all := db.GetAllDatabase()
	if len(all) < 2 { t.Fatalf("expected at least 2 databases, got %v", all) }

	// add a new database
	db.AddDatabase("unit_db")
	if !enhancedFakeDriverInstance.databases["unit_db"] {
		t.Fatalf("unit_db should have been created in driver state")
	}
	// verify appears in list
	found := false
	for _, name := range db.GetAllDatabase() { if name == "unit_db" { found = true; break } }
	if !found { t.Fatalf("unit_db not found in GetAllDatabase list") }

	// delete it
	db.DeleteDatabase("unit_db")
	if enhancedFakeDriverInstance.databases["unit_db"] {
		t.Fatalf("unit_db should have been removed by DeleteDatabase")
	}
	found = false
	for _, name := range db.GetAllDatabase() { if name == "unit_db" { found = true; break } }
	if found { t.Fatalf("unit_db should not be present after delete") }
}

