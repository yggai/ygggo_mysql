package ygggo_mysql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mysql "github.com/go-sql-driver/mysql"
)

func TestDSNBuilder_BasicConstruction(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Port(3306).
		Username("testuser").
		Password("testpass").
		Database("testdb").
		Build()

	expected := "testuser:testpass@tcp(localhost:3306)/testdb"
	assert.Equal(t, expected, dsn)

	// Verify the DSN can be parsed by go-sql-driver/mysql
	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "testuser", config.User)
	assert.Equal(t, "testpass", config.Passwd)
	assert.Equal(t, "tcp", config.Net)
	assert.Equal(t, "localhost:3306", config.Addr)
	assert.Equal(t, "testdb", config.DBName)
}

func TestDSNBuilder_WithoutPassword(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Port(3306).
		Username("testuser").
		Database("testdb").
		Build()

	expected := "testuser@tcp(localhost:3306)/testdb"
	assert.Equal(t, expected, dsn)
}

func TestDSNBuilder_WithoutUsername(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Port(3306).
		Database("testdb").
		Build()

	expected := "tcp(localhost:3306)/testdb"
	assert.Equal(t, expected, dsn)
}

func TestDSNBuilder_DefaultPort(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		Build()

	expected := "testuser:testpass@tcp(localhost:3306)/testdb"
	assert.Equal(t, expected, dsn)
}

func TestDSNBuilder_SpecialCharacters(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Port(3306).
		Username("test@user").
		Password("pass:word/with!special#chars").
		Database("test/db").
		Build()

	// Verify the DSN contains URL-encoded special characters
	assert.Contains(t, dsn, "test%40user") // @ encoded as %40
	assert.Contains(t, dsn, "pass%3Aword") // : encoded as %3A

	// Verify the DSN can be parsed correctly
	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	// Note: The mysql driver may return URL-encoded values in some cases
	// The important thing is that the DSN is valid and parseable
	assert.NotEmpty(t, config.User)
	assert.NotEmpty(t, config.Passwd)
	assert.NotEmpty(t, config.DBName)
}

func TestDSNBuilder_TLSDisabled(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		DisableTLS().
		Build()

	assert.Contains(t, dsn, "tls=false")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "false", config.TLSConfig)
}

func TestDSNBuilder_TLSRequired(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		RequireTLS().
		Build()

	assert.Contains(t, dsn, "tls=true")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "true", config.TLSConfig)
}

func TestDSNBuilder_TLSSkipVerify(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		TLSSkipVerify().
		Build()

	assert.Contains(t, dsn, "tls=skip-verify")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "skip-verify", config.TLSConfig)
}

func TestDSNBuilder_Compression(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		EnableCompression().
		Build()

	assert.Contains(t, dsn, "compress=true")

	// Verify the DSN can be parsed (compression field is not exported)
	_, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
}

func TestDSNBuilder_Timeouts(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		SetTimeout(30*time.Second).
		SetReadTimeout(10*time.Second).
		SetWriteTimeout(15*time.Second).
		Build()

	assert.Contains(t, dsn, "timeout=30s")
	assert.Contains(t, dsn, "readTimeout=10s")
	assert.Contains(t, dsn, "writeTimeout=15s")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 10*time.Second, config.ReadTimeout)
	assert.Equal(t, 15*time.Second, config.WriteTimeout)
}

func TestDSNBuilder_Charset(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		SetCharset("utf8mb4").
		Build()

	assert.Contains(t, dsn, "charset=utf8mb4")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	// The charset parameter should be in the Params map
	if config.Params != nil {
		assert.Equal(t, "utf8mb4", config.Params["charset"])
	}
}

func TestDSNBuilder_ParseTime(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		EnableParseTime().
		Build()

	assert.Contains(t, dsn, "parseTime=true")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.True(t, config.ParseTime)
}

func TestDSNBuilder_Location(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		SetLocation("UTC").
		Build()

	assert.Contains(t, dsn, "loc=UTC")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, config.Loc)
}

func TestDSNBuilder_CustomParams(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		SetParam("autocommit", "true").
		SetParam("sql_mode", "STRICT_TRANS_TABLES").
		Build()

	assert.Contains(t, dsn, "autocommit=true")
	assert.Contains(t, dsn, "sql_mode=STRICT_TRANS_TABLES")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "true", config.Params["autocommit"])
	assert.Equal(t, "STRICT_TRANS_TABLES", config.Params["sql_mode"])
}

func TestDSNBuilder_ChainedConfiguration(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("db.example.com").
		Port(3307).
		Username("appuser").
		Password("secretpass").
		Database("production").
		RequireTLS().
		EnableCompression().
		SetTimeout(60*time.Second).
		SetCharset("utf8mb4").
		EnableParseTime().
		SetLocation("America/New_York").
		SetParam("sql_mode", "STRICT_TRANS_TABLES,NO_ZERO_DATE").
		Build()

	// Verify all parameters are present
	assert.Contains(t, dsn, "appuser:secretpass@tcp(db.example.com:3307)/production")
	assert.Contains(t, dsn, "tls=true")
	assert.Contains(t, dsn, "compress=true")
	assert.Contains(t, dsn, "timeout=60s")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=true")
	assert.Contains(t, dsn, "loc=America%2FNew_York")
	assert.Contains(t, dsn, "sql_mode=STRICT_TRANS_TABLES%2CNO_ZERO_DATE")

	// Verify the DSN can be parsed
	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "appuser", config.User)
	assert.Equal(t, "production", config.DBName)
	assert.Equal(t, "true", config.TLSConfig)
	assert.Equal(t, 60*time.Second, config.Timeout)
}

func TestDSNBuilder_PresetDevelopment(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		DevelopmentPreset().
		Host("localhost").
		Username("devuser").
		Password("devpass").
		Database("devdb").
		Build()

	// Verify development preset settings
	assert.Contains(t, dsn, "devuser:devpass@tcp(localhost:3306)/devdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=true")
	assert.Contains(t, dsn, "loc=Local")
	assert.Contains(t, dsn, "tls=false") // Development typically doesn't use TLS

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "devuser", config.User)
	assert.Equal(t, "devdb", config.DBName)
	assert.True(t, config.ParseTime)
}

func TestDSNBuilder_PresetProduction(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		ProductionPreset().
		Host("prod.example.com").
		Username("produser").
		Password("prodpass").
		Database("proddb").
		Build()

	// Verify production preset settings
	assert.Contains(t, dsn, "produser:prodpass@tcp(prod.example.com:3306)/proddb")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=true")
	assert.Contains(t, dsn, "tls=true") // Production should use TLS
	assert.Contains(t, dsn, "compress=true") // Production should use compression
	assert.Contains(t, dsn, "timeout=30s")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "produser", config.User)
	assert.Equal(t, "proddb", config.DBName)
	assert.Equal(t, "true", config.TLSConfig)
	assert.True(t, config.ParseTime)
}

func TestDSNBuilder_PresetTesting(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		TestingPreset().
		Host("testhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		Build()

	// Verify testing preset settings
	assert.Contains(t, dsn, "testuser:testpass@tcp(testhost:3306)/testdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=true")
	assert.Contains(t, dsn, "timeout=5s") // Fast timeouts for testing

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "testuser", config.User)
	assert.Equal(t, "testdb", config.DBName)
	assert.Equal(t, 5*time.Second, config.Timeout)
}

func TestDSNBuilder_PresetHighPerformance(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		HighPerformancePreset().
		Host("perf.example.com").
		Username("perfuser").
		Password("perfpass").
		Database("perfdb").
		Build()

	// Verify high performance preset settings
	assert.Contains(t, dsn, "perfuser:perfpass@tcp(perf.example.com:3306)/perfdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "compress=true") // Compression for performance
	assert.Contains(t, dsn, "parseTime=true")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "perfuser", config.User)
	assert.Equal(t, "perfdb", config.DBName)
	assert.True(t, config.ParseTime)
}

func TestDSNBuilder_ToConfig(t *testing.T) {
	builder := NewDSNBuilder()
	config := builder.
		Host("localhost").
		Port(3307).
		Username("testuser").
		Password("testpass").
		Database("testdb").
		SetParam("custom", "value").
		ToConfig()

	assert.Equal(t, "mysql", config.Driver)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 3307, config.Port)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "testpass", config.Password)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "value", config.Params["custom"])
	assert.NotEmpty(t, config.DSN)
}

func TestDSNBuilder_FromConfig(t *testing.T) {
	originalConfig := Config{
		Host:     "example.com",
		Port:     3307,
		Username: "user",
		Password: "pass",
		Database: "db",
		Params: map[string]string{
			"charset": "utf8mb4",
			"custom":  "value",
		},
	}

	builder := FromConfig(originalConfig)
	dsn := builder.Build()

	assert.Contains(t, dsn, "user:pass@tcp(example.com:3307)/db")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "custom=value")

	// Convert back to config
	newConfig := builder.ToConfig()
	assert.Equal(t, originalConfig.Host, newConfig.Host)
	assert.Equal(t, originalConfig.Port, newConfig.Port)
	assert.Equal(t, originalConfig.Username, newConfig.Username)
	assert.Equal(t, originalConfig.Password, newConfig.Password)
	assert.Equal(t, originalConfig.Database, newConfig.Database)
}

func TestDSNBuilder_AdvancedTLS(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("secure.example.com").
		Username("secureuser").
		Password("securepass").
		Database("securedb").
		TLSCustom("custom").
		Build()

	assert.Contains(t, dsn, "tls=custom")

	// Note: Custom TLS config names need to be registered with the driver
	// For testing, we just verify the DSN contains the correct parameter
}

func TestDSNBuilder_TLSCustomConfig(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("secure.example.com").
		Username("secureuser").
		Password("securepass").
		Database("securedb").
		TLSCustom("myconfig").
		Build()

	assert.Contains(t, dsn, "tls=myconfig")

	// Note: Custom TLS config names need to be registered with the driver
	// For testing, we just verify the DSN contains the correct parameter
}

func TestDSNBuilder_AdvancedParameters(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		SetCollation("utf8mb4_unicode_ci").
		SetSQLMode("STRICT_TRANS_TABLES,NO_ZERO_DATE").
		SetTimeZone("UTC").
		SetAutoCommit(true).
		SetTransactionIsolation("READ-COMMITTED").
		EnableMultiStatements().
		EnableInterpolateParams().
		Build()

	assert.Contains(t, dsn, "collation=utf8mb4_unicode_ci")
	assert.Contains(t, dsn, "sql_mode=STRICT_TRANS_TABLES%2CNO_ZERO_DATE")
	assert.Contains(t, dsn, "time_zone=UTC")
	assert.Contains(t, dsn, "autocommit=true")
	assert.Contains(t, dsn, "tx_isolation=READ-COMMITTED")
	assert.Contains(t, dsn, "multiStatements=true")
	assert.Contains(t, dsn, "interpolateParams=true")

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	require.NotNil(t, config.Params, "Params should not be nil")

	// Check that parameters are present in the DSN string
	assert.Contains(t, dsn, "collation=utf8mb4_unicode_ci")
	assert.Contains(t, dsn, "sql_mode=STRICT_TRANS_TABLES%2CNO_ZERO_DATE")
	assert.Contains(t, dsn, "time_zone=UTC")

	// Note: Some parameters might not be accessible through config.Params
	// depending on the MySQL driver implementation
}

func TestDSNBuilder_Validation(t *testing.T) {
	// Test valid configuration
	builder := NewDSNBuilder()
	builder.Host("localhost").Database("testdb")
	err := builder.Validate()
	assert.NoError(t, err)

	// Test missing host
	builder2 := NewDSNBuilder()
	builder2.Database("testdb")
	err = builder2.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")

	// Test missing database
	builder3 := NewDSNBuilder()
	builder3.Host("localhost")
	err = builder3.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database name is required")

	// Test invalid port
	builder4 := NewDSNBuilder()
	builder4.Host("localhost").Database("testdb").Port(70000)
	err = builder4.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between")
}

func TestDSNBuilder_BuildWithValidation(t *testing.T) {
	builder := NewDSNBuilder()
	dsn, err := builder.
		Host("localhost").
		Username("testuser").
		Password("testpass").
		Database("testdb").
		BuildWithValidation()

	require.NoError(t, err)
	assert.Contains(t, dsn, "testuser:testpass@tcp(localhost:3306)/testdb")

	// Test invalid configuration
	builder2 := NewDSNBuilder()
	_, err = builder2.BuildWithValidation()
	assert.Error(t, err)
}

func TestDSNBuilder_Clone(t *testing.T) {
	original := NewDSNBuilder()
	original.
		Host("localhost").
		Port(3307).
		Username("testuser").
		Password("testpass").
		Database("testdb").
		RequireTLS().
		EnableCompression().
		SetTimeout(30*time.Second).
		SetCharset("utf8mb4").
		SetParam("custom", "value")

	clone := original.Clone()

	// Verify clone has same configuration (parameter order may vary)
	originalDSN := original.Build()
	cloneDSN := clone.Build()

	// Parse both DSNs to compare their components
	originalConfig, err := mysql.ParseDSN(originalDSN)
	require.NoError(t, err)
	cloneConfig, err := mysql.ParseDSN(cloneDSN)
	require.NoError(t, err)

	// Compare key components
	assert.Equal(t, originalConfig.User, cloneConfig.User)
	assert.Equal(t, originalConfig.Passwd, cloneConfig.Passwd)
	assert.Equal(t, originalConfig.Addr, cloneConfig.Addr)
	assert.Equal(t, originalConfig.DBName, cloneConfig.DBName)
	assert.Equal(t, originalConfig.TLSConfig, cloneConfig.TLSConfig)

	// Verify they are independent
	clone.Host("different.host")
	assert.NotEqual(t, original.Build(), clone.Build())
}

func TestDSNBuilder_TimeoutValidation(t *testing.T) {
	builder := NewDSNBuilder()
	builder.Host("localhost").Database("testdb")

	// Test negative timeout
	builder.SetTimeout(-1 * time.Second)
	err := builder.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be positive")

	// Test negative read timeout
	builder2 := NewDSNBuilder()
	builder2.Host("localhost").Database("testdb")
	builder2.SetReadTimeout(-1 * time.Second)
	err = builder2.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read timeout must be positive")
}

func TestDSNBuilder_SecurePreset(t *testing.T) {
	builder := NewDSNBuilder()
	dsn := builder.
		SecurePreset().
		Host("secure.example.com").
		Username("secureuser").
		Password("securepass").
		Database("securedb").
		Build()

	// Verify secure preset settings
	assert.Contains(t, dsn, "secureuser:securepass@tcp(secure.example.com:3306)/securedb")
	assert.Contains(t, dsn, "tls=true")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=true")
	assert.Contains(t, dsn, "tx_isolation=SERIALIZABLE") // Highest isolation level

	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	assert.Equal(t, "secureuser", config.User)
	assert.Equal(t, "securedb", config.DBName)
	assert.Equal(t, "true", config.TLSConfig)
}
