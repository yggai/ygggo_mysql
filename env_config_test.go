package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
)

func TestEnv_OverridesRawDSN(t *testing.T) {
	// Given env DSN should override any cfg.DSN
	const envDSN = "envuser:envpass@tcp(127.0.0.1:3307)/envdb?parseTime=true"
	t.Setenv("YGGGO_MYSQL_DSN", envDSN)

	// Register that DSN with sqlmock
	_, mock, err := sqlmock.NewWithDSN(envDSN, sqlmock.MonitorPingsOption(true))
	if err != nil { t.Fatalf("sqlmock.NewWithDSN: %v", err) }
	mock.ExpectPing()

	cfg := Config{Driver: "sqlmock", DSN: "ignored:ignored@tcp(localhost:3306)/ignored"}

	ctx := context.Background()
	p, err := NewPool(ctx, cfg)
	if err != nil { t.Fatalf("NewPool err: %v", err) }
	defer p.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEnv_FieldBasedValues_BuildsDSN(t *testing.T) {
	// Set env for field-based build
	t.Setenv("YGGGO_MYSQL_DRIVER", "mysql")
	t.Setenv("YGGGO_MYSQL_HOST", "127.0.0.1")
	t.Setenv("YGGGO_MYSQL_PORT", "3306")
	t.Setenv("YGGGO_MYSQL_USERNAME", "root")
	t.Setenv("YGGGO_MYSQL_PASSWORD", "pa%￥@ss:word/!")
	t.Setenv("YGGGO_MYSQL_DATABASE", "dbname/withslash")
	t.Setenv("YGGGO_MYSQL_PARAMS", "parseTime=true&loc=Local")

	cfg := Config{} // empty on purpose
	applyEnv(&cfg)

	dsn, err := dsnFromConfig(cfg)
	if err != nil { t.Fatalf("dsnFromConfig: %v", err) }

	mc, err := mysql.ParseDSN(dsn)
	if err != nil { t.Fatalf("ParseDSN err: %v, dsn=%q", err, dsn) }
	if mc.User != "root" { t.Fatalf("user=%q", mc.User) }
	if mc.Passwd != "pa%￥@ss:word/!" { t.Fatalf("passwd=%q", mc.Passwd) }
	if mc.Addr != "127.0.0.1:3306" { t.Fatalf("addr=%q", mc.Addr) }
	if mc.DBName != "dbname/withslash" { t.Fatalf("db=%q", mc.DBName) }
	if !mc.ParseTime { t.Fatalf("parseTime expected true") }
	// loc is a recognized parameter; driver sets mc.Loc and removes from Params
	if mc.Loc == nil || mc.Loc.String() != "Local" { t.Fatalf("loc expected Local, got %#v", mc.Loc) }
}

