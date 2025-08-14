package ygggo_mysql

import (
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

func TestDSNString_UseRawDSN(t *testing.T) {
	cfg := Config{Driver: "mysql", DSN: "user:pass@tcp(localhost:3306)/db?parseTime=true"}
	dsn, err := dsnFromConfig(cfg)
	if err != nil {
		t.Fatalf("dsnFromConfig error: %v", err)
	}
	if dsn != cfg.DSN {
		t.Fatalf("expected raw DSN unchanged, got %q", dsn)
	}
}

func TestDSNString_BuildFromFields_WithSpecialPassword(t *testing.T) {
	password := "pa%￥@ss:word/!"
	cfg := Config{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		Username: "root",
		Password: password,
		Database: "dbname/withslash",
		Params: map[string]string{
			"parseTime": "true",
		},
	}
	dsn, err := dsnFromConfig(cfg)
	if err != nil {
		t.Fatalf("dsnFromConfig error: %v", err)
	}
	// Ensure mysql.ParseDSN can parse it and recover the same values
	mc, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN error: %v. dsn=%q", err, dsn)
	}
	if mc.User != cfg.Username {
		t.Fatalf("user mismatch: got %q want %q", mc.User, cfg.Username)
	}
	if mc.Passwd != cfg.Password {
		t.Fatalf("password mismatch: got %q want %q", mc.Passwd, cfg.Password)
	}
	if mc.Net != "tcp" || mc.Addr != "127.0.0.1:3306" {
		t.Fatalf("addr mismatch: net=%q addr=%q", mc.Net, mc.Addr)
	}
	if mc.DBName != cfg.Database {
		t.Fatalf("dbname mismatch: got %q want %q", mc.DBName, cfg.Database)
	}
	// go-sql-driver/mysql consumes parseTime into mc.ParseTime, not mc.Params
	if !mc.ParseTime {
		t.Fatalf("expected parseTime to be true")
	}
}

