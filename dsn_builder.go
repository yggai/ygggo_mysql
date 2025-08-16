package ygggo_mysql

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TLSConfig represents TLS/SSL configuration for MySQL connections
type TLSConfig struct {
	Mode               string // "disabled", "preferred", "required", "verify-ca", "verify-identity"
	CertFile           string // Client certificate file
	KeyFile            string // Client private key file
	CAFile             string // Certificate Authority file
	ServerName         string // Server name for certificate verification
	InsecureSkipVerify bool   // Skip certificate verification
}

// TimeoutConfig represents timeout configuration
type TimeoutConfig struct {
	Connection time.Duration // Connection timeout
	Read       time.Duration // Read timeout
	Write      time.Duration // Write timeout
}

// DSNBuilder provides a fluent interface for building MySQL DSN strings
type DSNBuilder struct {
	host     string
	port     int
	username string
	password string
	database string
	
	// TLS configuration
	tlsMode   string
	tlsConfig *TLSConfig
	
	// Performance settings
	compression bool
	
	// Timeout settings
	timeout      *time.Duration
	readTimeout  *time.Duration
	writeTimeout *time.Duration
	
	// Character encoding
	charset string
	
	// MySQL-specific settings
	parseTime bool
	location  string
	
	// Custom parameters
	params map[string]string
}

// NewDSNBuilder creates a new DSN builder with default settings
func NewDSNBuilder() *DSNBuilder {
	return &DSNBuilder{
		port:   3306, // Default MySQL port
		params: make(map[string]string),
	}
}

// Host sets the database host
func (b *DSNBuilder) Host(host string) *DSNBuilder {
	b.host = host
	return b
}

// Port sets the database port
func (b *DSNBuilder) Port(port int) *DSNBuilder {
	b.port = port
	return b
}

// Username sets the database username
func (b *DSNBuilder) Username(username string) *DSNBuilder {
	b.username = username
	return b
}

// Password sets the database password
func (b *DSNBuilder) Password(password string) *DSNBuilder {
	b.password = password
	return b
}

// Database sets the database name
func (b *DSNBuilder) Database(database string) *DSNBuilder {
	b.database = database
	return b
}

// DisableTLS disables TLS/SSL encryption
func (b *DSNBuilder) DisableTLS() *DSNBuilder {
	b.tlsMode = "false"
	return b
}

// RequireTLS enables TLS/SSL encryption
func (b *DSNBuilder) RequireTLS() *DSNBuilder {
	b.tlsMode = "true"
	return b
}

// TLSSkipVerify enables TLS but skips certificate verification
func (b *DSNBuilder) TLSSkipVerify() *DSNBuilder {
	b.tlsMode = "skip-verify"
	return b
}

// TLSCustom sets a custom TLS configuration name
func (b *DSNBuilder) TLSCustom(configName string) *DSNBuilder {
	b.tlsMode = configName
	return b
}

// TLSWithCertificates configures TLS with client certificates
func (b *DSNBuilder) TLSWithCertificates(certFile, keyFile, caFile string) *DSNBuilder {
	b.tlsConfig = &TLSConfig{
		Mode:     "required",
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}
	b.tlsMode = "custom"
	return b
}

// TLSWithConfig sets a complete TLS configuration
func (b *DSNBuilder) TLSWithConfig(config *TLSConfig) *DSNBuilder {
	b.tlsConfig = config
	b.tlsMode = "custom"
	return b
}

// TLSVerifyCA enables TLS with CA certificate verification
func (b *DSNBuilder) TLSVerifyCA() *DSNBuilder {
	b.tlsMode = "verify-ca"
	return b
}

// TLSVerifyIdentity enables TLS with full certificate and identity verification
func (b *DSNBuilder) TLSVerifyIdentity() *DSNBuilder {
	b.tlsMode = "verify-identity"
	return b
}

// EnableCompression enables MySQL compression
func (b *DSNBuilder) EnableCompression() *DSNBuilder {
	b.compression = true
	return b
}

// DisableCompression disables MySQL compression
func (b *DSNBuilder) DisableCompression() *DSNBuilder {
	b.compression = false
	return b
}

// SetTimeout sets the connection timeout
func (b *DSNBuilder) SetTimeout(timeout time.Duration) *DSNBuilder {
	b.timeout = &timeout
	return b
}

// SetReadTimeout sets the read timeout
func (b *DSNBuilder) SetReadTimeout(timeout time.Duration) *DSNBuilder {
	b.readTimeout = &timeout
	return b
}

// SetWriteTimeout sets the write timeout
func (b *DSNBuilder) SetWriteTimeout(timeout time.Duration) *DSNBuilder {
	b.writeTimeout = &timeout
	return b
}

// SetCharset sets the character set
func (b *DSNBuilder) SetCharset(charset string) *DSNBuilder {
	b.charset = charset
	return b
}

// EnableParseTime enables automatic parsing of TIME and DATE values
func (b *DSNBuilder) EnableParseTime() *DSNBuilder {
	b.parseTime = true
	return b
}

// DisableParseTime disables automatic parsing of TIME and DATE values
func (b *DSNBuilder) DisableParseTime() *DSNBuilder {
	b.parseTime = false
	return b
}

// SetLocation sets the timezone location
func (b *DSNBuilder) SetLocation(location string) *DSNBuilder {
	b.location = location
	return b
}

// SetParam sets a custom parameter
func (b *DSNBuilder) SetParam(key, value string) *DSNBuilder {
	b.params[key] = value
	return b
}

// SetCollation sets the collation
func (b *DSNBuilder) SetCollation(collation string) *DSNBuilder {
	return b.SetParam("collation", collation)
}

// SetSQLMode sets the SQL mode
func (b *DSNBuilder) SetSQLMode(mode string) *DSNBuilder {
	return b.SetParam("sql_mode", mode)
}

// SetTimeZone sets the time zone
func (b *DSNBuilder) SetTimeZone(timezone string) *DSNBuilder {
	return b.SetParam("time_zone", timezone)
}

// SetAutoCommit sets the autocommit mode
func (b *DSNBuilder) SetAutoCommit(enabled bool) *DSNBuilder {
	if enabled {
		return b.SetParam("autocommit", "true")
	}
	return b.SetParam("autocommit", "false")
}

// SetTransactionIsolation sets the transaction isolation level
func (b *DSNBuilder) SetTransactionIsolation(level string) *DSNBuilder {
	return b.SetParam("tx_isolation", level)
}

// SetMaxAllowedPacket sets the maximum allowed packet size
func (b *DSNBuilder) SetMaxAllowedPacket(size int) *DSNBuilder {
	return b.SetParam("maxAllowedPacket", strconv.Itoa(size))
}

// SetNetBufferLength sets the network buffer length
func (b *DSNBuilder) SetNetBufferLength(length int) *DSNBuilder {
	return b.SetParam("netBufferLength", strconv.Itoa(length))
}

// EnableMultiStatements enables multi-statement support
func (b *DSNBuilder) EnableMultiStatements() *DSNBuilder {
	return b.SetParam("multiStatements", "true")
}

// DisableMultiStatements disables multi-statement support
func (b *DSNBuilder) DisableMultiStatements() *DSNBuilder {
	return b.SetParam("multiStatements", "false")
}

// EnableInterpolateParams enables client-side parameter interpolation
func (b *DSNBuilder) EnableInterpolateParams() *DSNBuilder {
	return b.SetParam("interpolateParams", "true")
}

// DisableInterpolateParams disables client-side parameter interpolation
func (b *DSNBuilder) DisableInterpolateParams() *DSNBuilder {
	return b.SetParam("interpolateParams", "false")
}

// Build constructs the final DSN string
func (b *DSNBuilder) Build() string {
	var dsn strings.Builder
	
	// Build authentication part
	if b.username != "" {
		dsn.WriteString(url.QueryEscape(b.username))
		if b.password != "" {
			dsn.WriteString(":")
			dsn.WriteString(url.QueryEscape(b.password))
		}
		dsn.WriteString("@")
	}
	
	// Build network and address part
	dsn.WriteString("tcp(")
	if b.host != "" {
		dsn.WriteString(b.host)
		dsn.WriteString(":")
		dsn.WriteString(strconv.Itoa(b.port))
	} else {
		dsn.WriteString(":")
		dsn.WriteString(strconv.Itoa(b.port))
	}
	dsn.WriteString(")")
	
	// Build database part
	dsn.WriteString("/")
	if b.database != "" {
		dsn.WriteString(url.QueryEscape(b.database))
	}
	
	// Build parameters
	params := b.buildParams()
	if len(params) > 0 {
		dsn.WriteString("?")
		dsn.WriteString(params)
	}
	
	return dsn.String()
}

// buildParams constructs the parameter string
func (b *DSNBuilder) buildParams() string {
	params := make(map[string]string)
	
	// Copy custom parameters first
	for k, v := range b.params {
		params[k] = v
	}
	
	// Add TLS configuration
	if b.tlsMode != "" {
		params["tls"] = b.tlsMode
	}
	
	// Add compression
	if b.compression {
		params["compress"] = "true"
	}
	
	// Add timeouts
	if b.timeout != nil {
		params["timeout"] = formatDuration(*b.timeout)
	}
	if b.readTimeout != nil {
		params["readTimeout"] = formatDuration(*b.readTimeout)
	}
	if b.writeTimeout != nil {
		params["writeTimeout"] = formatDuration(*b.writeTimeout)
	}
	
	// Add charset
	if b.charset != "" {
		params["charset"] = b.charset
	}
	
	// Add parseTime
	if b.parseTime {
		params["parseTime"] = "true"
	}
	
	// Add location
	if b.location != "" {
		params["loc"] = b.location
	}
	
	// Build parameter string
	if len(params) == 0 {
		return ""
	}
	
	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", 
			url.QueryEscape(key), url.QueryEscape(value)))
	}
	
	return strings.Join(parts, "&")
}

// formatDuration formats a duration for MySQL DSN
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Nanoseconds()/1000000)
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}

// ToConfig converts the DSN builder to a Config struct
func (b *DSNBuilder) ToConfig() Config {
	return Config{
		Driver:   "mysql",
		DSN:      b.Build(),
		Host:     b.host,
		Port:     b.port,
		Username: b.username,
		Password: b.password,
		Database: b.database,
		Params:   b.params,
	}
}

// FromConfig creates a DSN builder from an existing Config
func FromConfig(config Config) *DSNBuilder {
	builder := NewDSNBuilder()
	
	if config.Host != "" {
		builder.Host(config.Host)
	}
	if config.Port > 0 {
		builder.Port(config.Port)
	}
	if config.Username != "" {
		builder.Username(config.Username)
	}
	if config.Password != "" {
		builder.Password(config.Password)
	}
	if config.Database != "" {
		builder.Database(config.Database)
	}
	
	// Copy parameters
	for k, v := range config.Params {
		builder.SetParam(k, v)
	}
	
	return builder
}

// Validation methods

// Validate checks if the DSN builder configuration is valid
func (b *DSNBuilder) Validate() error {
	if b.host == "" {
		return fmt.Errorf("host is required")
	}

	if b.port <= 0 || b.port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", b.port)
	}

	if b.database == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate TLS configuration
	if b.tlsConfig != nil {
		if err := b.validateTLSConfig(); err != nil {
			return fmt.Errorf("TLS configuration error: %w", err)
		}
	}

	// Validate timeouts
	if b.timeout != nil && *b.timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", *b.timeout)
	}
	if b.readTimeout != nil && *b.readTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive, got %v", *b.readTimeout)
	}
	if b.writeTimeout != nil && *b.writeTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive, got %v", *b.writeTimeout)
	}

	return nil
}

// validateTLSConfig validates TLS configuration
func (b *DSNBuilder) validateTLSConfig() error {
	if b.tlsConfig == nil {
		return nil
	}

	config := b.tlsConfig

	// Validate mode
	validModes := []string{"disabled", "preferred", "required", "verify-ca", "verify-identity"}
	validMode := false
	for _, mode := range validModes {
		if config.Mode == mode {
			validMode = true
			break
		}
	}
	if !validMode {
		return fmt.Errorf("invalid TLS mode: %s", config.Mode)
	}

	// If certificates are specified, validate they exist
	if config.CertFile != "" && config.KeyFile == "" {
		return fmt.Errorf("key file is required when certificate file is specified")
	}
	if config.KeyFile != "" && config.CertFile == "" {
		return fmt.Errorf("certificate file is required when key file is specified")
	}

	return nil
}

// BuildWithValidation builds the DSN after validating the configuration
func (b *DSNBuilder) BuildWithValidation() (string, error) {
	if err := b.Validate(); err != nil {
		return "", err
	}
	return b.Build(), nil
}

// Clone creates a copy of the DSN builder
func (b *DSNBuilder) Clone() *DSNBuilder {
	clone := &DSNBuilder{
		host:         b.host,
		port:         b.port,
		username:     b.username,
		password:     b.password,
		database:     b.database,
		tlsMode:      b.tlsMode,
		compression:  b.compression,
		charset:      b.charset,
		parseTime:    b.parseTime,
		location:     b.location,
		params:       make(map[string]string),
	}

	// Copy timeouts
	if b.timeout != nil {
		timeout := *b.timeout
		clone.timeout = &timeout
	}
	if b.readTimeout != nil {
		readTimeout := *b.readTimeout
		clone.readTimeout = &readTimeout
	}
	if b.writeTimeout != nil {
		writeTimeout := *b.writeTimeout
		clone.writeTimeout = &writeTimeout
	}

	// Copy TLS config
	if b.tlsConfig != nil {
		clone.tlsConfig = &TLSConfig{
			Mode:               b.tlsConfig.Mode,
			CertFile:           b.tlsConfig.CertFile,
			KeyFile:            b.tlsConfig.KeyFile,
			CAFile:             b.tlsConfig.CAFile,
			ServerName:         b.tlsConfig.ServerName,
			InsecureSkipVerify: b.tlsConfig.InsecureSkipVerify,
		}
	}

	// Copy parameters
	for k, v := range b.params {
		clone.params[k] = v
	}

	return clone
}

// Preset configurations for common scenarios

// DevelopmentPreset configures the builder for development environment
func (b *DSNBuilder) DevelopmentPreset() *DSNBuilder {
	return b.
		DisableTLS().                    // No TLS for local development
		SetCharset("utf8mb4").           // Modern UTF-8 support
		EnableParseTime().               // Parse time values
		SetLocation("Local").            // Use local timezone
		SetTimeout(10 * time.Second).    // Reasonable timeout
		SetParam("sql_mode", "STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO")
}

// ProductionPreset configures the builder for production environment
func (b *DSNBuilder) ProductionPreset() *DSNBuilder {
	return b.
		RequireTLS().                    // Require TLS in production
		EnableCompression().             // Enable compression for performance
		SetCharset("utf8mb4").           // Modern UTF-8 support
		EnableParseTime().               // Parse time values
		SetLocation("UTC").              // Use UTC timezone
		SetTimeout(30 * time.Second).    // Conservative timeout
		SetReadTimeout(10 * time.Second).
		SetWriteTimeout(10 * time.Second).
		SetParam("sql_mode", "STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO").
		SetParam("autocommit", "true").
		SetParam("tx_isolation", "READ-COMMITTED")
}

// TestingPreset configures the builder for testing environment
func (b *DSNBuilder) TestingPreset() *DSNBuilder {
	return b.
		DisableTLS().                    // No TLS for testing
		SetCharset("utf8mb4").           // Modern UTF-8 support
		EnableParseTime().               // Parse time values
		SetLocation("UTC").              // Use UTC for consistent testing
		SetTimeout(5 * time.Second).     // Fast timeout for quick test failures
		SetReadTimeout(3 * time.Second).
		SetWriteTimeout(3 * time.Second).
		SetParam("sql_mode", "STRICT_TRANS_TABLES")
}

// HighPerformancePreset configures the builder for high-performance scenarios
func (b *DSNBuilder) HighPerformancePreset() *DSNBuilder {
	return b.
		EnableCompression().             // Enable compression
		SetCharset("utf8mb4").           // Modern UTF-8 support
		EnableParseTime().               // Parse time values
		SetLocation("UTC").              // Use UTC timezone
		SetTimeout(60 * time.Second).    // Longer timeout for complex queries
		SetReadTimeout(30 * time.Second).
		SetWriteTimeout(30 * time.Second).
		SetParam("autocommit", "true").
		SetParam("tx_isolation", "READ-COMMITTED").
		SetParam("innodb_lock_wait_timeout", "50")
}

// SecurePreset configures the builder with security-focused settings
func (b *DSNBuilder) SecurePreset() *DSNBuilder {
	return b.
		RequireTLS().                    // Require TLS
		SetCharset("utf8mb4").           // Modern UTF-8 support
		EnableParseTime().               // Parse time values
		SetLocation("UTC").              // Use UTC timezone
		SetTimeout(15 * time.Second).    // Moderate timeout
		SetParam("sql_mode", "STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO").
		SetParam("autocommit", "true").
		SetParam("tx_isolation", "SERIALIZABLE") // Highest isolation level
}
