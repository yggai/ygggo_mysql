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

// PoolConfig holds connection pool-related settings.
//
// These settings control how the database connection pool behaves,
// including connection limits, lifetimes, and idle timeouts.
//
// Example:
//
//	poolConfig := PoolConfig{
//		MaxOpen:         25,                // Maximum open connections
//		MaxIdle:         10,                // Maximum idle connections
//		ConnMaxLifetime: 5 * time.Minute,   // Maximum connection lifetime
//		ConnMaxIdleTime: 2 * time.Minute,   // Maximum idle time
//	}
type PoolConfig struct {
	// MaxOpen sets the maximum number of open connections to the database.
	//
	// If MaxOpen is 0, then there is no limit on the number of open connections.
	// The default is 0 (unlimited).
	MaxOpen int

	// MaxIdle sets the maximum number of connections in the idle connection pool.
	//
	// If MaxIdle is 0, no idle connections are retained.
	// If MaxIdle is greater than MaxOpen, MaxIdle is reduced to match MaxOpen.
	// The default is 2.
	MaxIdle int

	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	//
	// Expired connections may be closed lazily before reuse.
	// If ConnMaxLifetime is 0, connections are not closed due to age.
	// The default is 0 (no maximum lifetime).
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime sets the maximum amount of time a connection may be idle.
	//
	// Expired connections may be closed lazily before reuse.
	// If ConnMaxIdleTime is 0, connections are not closed due to idle time.
	// The default is 0 (no maximum idle time).
	ConnMaxIdleTime time.Duration
}

// Config holds the complete library configuration.
//
// This structure contains all settings needed to configure the database
// connection, pool behavior, retry policies, telemetry, and other features.
// Configuration can be provided programmatically or through environment variables.
//
// Example usage:
//
//	config := Config{
//		Host:     "localhost",
//		Port:     3306,
//		Username: "user",
//		Password: "password",
//		Database: "mydb",
//		Driver:   "mysql",
//		Pool: PoolConfig{
//			MaxOpen: 25,
//			MaxIdle: 10,
//		},
//		SlowQueryThreshold: 100 * time.Millisecond,
//	}
//
// Environment Variable Override:
//
// All configuration fields can be overridden using environment variables
// with the prefix YGGGO_MYSQL_. For example:
//   - YGGGO_MYSQL_HOST=localhost
//   - YGGGO_MYSQL_PORT=3306
//   - YGGGO_MYSQL_USERNAME=user
//   - YGGGO_MYSQL_PASSWORD=secret
type Config struct {
	// Driver specifies the SQL driver to use.
	//
	// Common values:
	//   - "mysql" for production MySQL connections
	//   - "sqlite" for testing with SQLite
	//
	// If empty, defaults to "mysql".
	Driver string

	// DSN is the complete Data Source Name connection string.
	//
	// If provided, this takes precedence over individual connection fields
	// (Host, Port, Username, etc.). If empty, the DSN will be constructed
	// from the individual fields.
	//
	// Example: "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True"
	DSN string

	// Host is the database server hostname or IP address.
	//
	// Used for DSN construction when DSN field is empty.
	// Can be overridden with YGGGO_MYSQL_HOST environment variable.
	Host string

	// Port is the database server port number.
	//
	// Used for DSN construction when DSN field is empty.
	// Default is 3306 for MySQL.
	// Can be overridden with YGGGO_MYSQL_PORT environment variable.
	Port int

	// Username is the database user name.
	//
	// Used for DSN construction when DSN field is empty.
	// Can be overridden with YGGGO_MYSQL_USERNAME environment variable.
	Username string

	// Password is the database user password.
	//
	// Used for DSN construction when DSN field is empty.
	// Can be overridden with YGGGO_MYSQL_PASSWORD environment variable.
	Password string

	// Database is the name of the database to connect to.
	//
	// Used for DSN construction when DSN field is empty.
	// Can be overridden with YGGGO_MYSQL_DATABASE environment variable.
	Database string

	// Params contains additional connection parameters.
	//
	// These are appended to the DSN as query parameters.
	// Common parameters include:
	//   - "charset": Character set (e.g., "utf8mb4")
	//   - "parseTime": Parse time values to time.Time ("true"/"false")
	//   - "loc": Time zone location (e.g., "UTC", "Local")
	//
	// Example:
	//	Params: map[string]string{
	//		"charset":   "utf8mb4",
	//		"parseTime": "true",
	//		"loc":       "UTC",
	//	}
	Params map[string]string

	// Pool contains connection pool configuration.
	//
	// See PoolConfig for detailed field descriptions.
	Pool PoolConfig

	// Retry contains retry policy configuration.
	//
	// See RetryPolicy for detailed field descriptions.
	Retry RetryPolicy

	// Telemetry contains observability configuration.
	//
	// See TelemetryConfig for detailed field descriptions.
	Telemetry TelemetryConfig

	// SlowQueryThreshold defines the duration above which queries are considered slow.
	//
	// Slow queries are logged and can be recorded for analysis.
	// If 0, slow query detection is disabled.
	// Default is 0 (disabled).
	//
	// Example: 100 * time.Millisecond
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
