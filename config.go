package ygggo_mysql

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PoolConfig holds pool-related settings (placeholders).
type PoolConfig struct {
	MaxOpen          int
	MaxIdle          int
	ConnMaxLifetime  time.Duration
	ConnMaxIdleTime  time.Duration
}

// Config holds library configuration (placeholders).
type Config struct {
	// Driver allows overriding the sql driver (e.g., "mysql" in prod, "sqlmock" in tests).
	Driver             string
	DSN                string
	// Field-based DSN building (used when DSN is empty)
	Host               string
	Port               int
	Username           string
	Password           string
	Database           string
	Params             map[string]string
	Pool               PoolConfig
	Retry              RetryPolicy
	Telemetry          TelemetryConfig
	SlowQueryThreshold time.Duration
}

// applyEnv overrides config with env vars (prefix YGGGO_MYSQL_*) when present.
func applyEnv(c *Config) {
	lookup := func(key string) (string, bool) { v, ok := os.LookupEnv(key); return v, ok }
	if v, ok := lookup("YGGGO_MYSQL_DRIVER"); ok { c.Driver = v }
	if v, ok := lookup("YGGGO_MYSQL_DSN"); ok { c.DSN = v }
	if v, ok := lookup("YGGGO_MYSQL_HOST"); ok { c.Host = v }
	if v, ok := lookup("YGGGO_MYSQL_PORT"); ok {
		if p, err := strconv.Atoi(v); err == nil { c.Port = p }
	}
	if v, ok := lookup("YGGGO_MYSQL_USERNAME"); ok { c.Username = v }
	if v, ok := lookup("YGGGO_MYSQL_PASSWORD"); ok { c.Password = v }
	if v, ok := lookup("YGGGO_MYSQL_DATABASE"); ok { c.Database = v }
	if v, ok := lookup("YGGGO_MYSQL_PARAMS"); ok {
		// parse "k=v&k2=v2" into map
		m := map[string]string{}
		for _, pair := range strings.Split(v, "&") {
			if pair == "" { continue }
			kv := strings.SplitN(pair, "=", 2)
			k := kv[0]
			val := ""
			if len(kv) == 2 { val = kv[1] }
			m[k] = val
		}
		c.Params = m
	}
}

// dsnFromConfig returns a DSN string.
// Priority: if Config.DSN is non-empty, return it unchanged.
// Otherwise build from host/port/username/password/database/params.
func dsnFromConfig(c Config) (string, error) {
	if strings.TrimSpace(c.DSN) != "" {
		return c.DSN, nil
	}
	addr := c.Host
	if c.Port > 0 {
		addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}
	dbEscaped := url.PathEscape(c.Database)
	// Build query params in stable order for test determinism
	var q string
	if len(c.Params) > 0 {
		keys := make([]string, 0, len(c.Params))
		for k := range c.Params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			// mysql driver recognizes plain keys like parseTime=true
			parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(c.Params[k])))
		}
		q = strings.Join(parts, "&")
	}
	// auth part: do not URL-encode password; mysql driver expects raw
	auth := ""
	if c.Username != "" {
		if c.Password != "" {
			auth = fmt.Sprintf("%s:%s@", c.Username, c.Password)
		} else {
			auth = c.Username + "@"
		}
	}
	dsn := fmt.Sprintf("%stcp(%s)/%s", auth, addr, dbEscaped)
	if q != "" {
		dsn += "?" + q
	}
	return dsn, nil
}
