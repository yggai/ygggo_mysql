# DSN Builder

The DSN Builder provides a fluent, type-safe interface for constructing MySQL Data Source Names (DSNs) with comprehensive support for TLS, compression, timeouts, character sets, and advanced MySQL parameters. It includes preset configurations for common environments and extensive validation capabilities.

## Features

- **Fluent Interface**: Chainable methods for intuitive DSN construction
- **Type Safety**: Compile-time safety and runtime validation
- **TLS/SSL Support**: Complete TLS configuration including custom certificates
- **Compression**: One-click MySQL compression enabling
- **Timeout Configuration**: Granular timeout control (connection, read, write)
- **Character Set Support**: UTF8MB4 and custom charset/collation configuration
- **Preset Configurations**: Pre-configured settings for development, production, testing, and high-performance scenarios
- **Advanced Parameters**: Support for all MySQL connection parameters
- **Validation**: Comprehensive configuration validation
- **Config Integration**: Seamless integration with existing Config structs

## Basic Usage

### Simple DSN Construction

```go
builder := ygggo.NewDSNBuilder()
dsn := builder.
    Host("localhost").
    Port(3306).
    Username("myuser").
    Password("mypassword").
    Database("mydatabase").
    Build()

// Result: "myuser:mypassword@tcp(localhost:3306)/mydatabase"
```

### With TLS and Compression

```go
dsn := ygggo.NewDSNBuilder().
    Host("db.example.com").
    Username("user").
    Password("pass").
    Database("mydb").
    RequireTLS().
    EnableCompression().
    Build()
```

### With Timeouts and Character Set

```go
dsn := ygggo.NewDSNBuilder().
    Host("localhost").
    Username("user").
    Password("pass").
    Database("mydb").
    SetTimeout(30*time.Second).
    SetReadTimeout(10*time.Second).
    SetWriteTimeout(10*time.Second).
    SetCharset("utf8mb4").
    Build()
```

## Preset Configurations

### Development Preset

```go
dsn := ygggo.NewDSNBuilder().
    DevelopmentPreset().
    Host("localhost").
    Username("devuser").
    Password("devpass").
    Database("development").
    Build()
```

**Development Preset Includes:**
- TLS disabled (for local development)
- UTF8MB4 character set
- ParseTime enabled
- Local timezone
- Reasonable timeouts
- Strict SQL mode

### Production Preset

```go
dsn := ygggo.NewDSNBuilder().
    ProductionPreset().
    Host("prod-db.example.com").
    Username("produser").
    Password("prodpass").
    Database("production").
    Build()
```

**Production Preset Includes:**
- TLS required
- Compression enabled
- UTF8MB4 character set
- UTC timezone
- Conservative timeouts
- Strict SQL mode
- Autocommit enabled
- READ-COMMITTED isolation

### Testing Preset

```go
dsn := ygggo.NewDSNBuilder().
    TestingPreset().
    Host("test-db").
    Username("testuser").
    Password("testpass").
    Database("test").
    Build()
```

**Testing Preset Includes:**
- TLS disabled
- Fast timeouts (quick failure detection)
- UTF8MB4 character set
- UTC timezone
- Minimal SQL mode

### High Performance Preset

```go
dsn := ygggo.NewDSNBuilder().
    HighPerformancePreset().
    Host("perf-db.example.com").
    Username("perfuser").
    Password("perfpass").
    Database("performance").
    Build()
```

**High Performance Preset Includes:**
- Compression enabled
- Optimized timeouts
- Performance-focused parameters
- Connection pooling friendly settings

### Secure Preset

```go
dsn := ygggo.NewDSNBuilder().
    SecurePreset().
    Host("secure-db.example.com").
    Username("secureuser").
    Password("securepass").
    Database("secure").
    Build()
```

**Secure Preset Includes:**
- TLS required
- SERIALIZABLE isolation level
- Strict SQL mode
- Security-focused parameters

## TLS/SSL Configuration

### Basic TLS Options

```go
// Disable TLS
builder.DisableTLS()

// Require TLS
builder.RequireTLS()

// TLS with skip verification (development only)
builder.TLSSkipVerify()

// Custom TLS configuration name
builder.TLSCustom("my-tls-config")
```

### Advanced TLS Configuration

```go
// TLS with client certificates
builder.TLSWithCertificates(
    "/path/to/client-cert.pem",
    "/path/to/client-key.pem", 
    "/path/to/ca-cert.pem",
)

// Custom TLS configuration
tlsConfig := &ygggo.TLSConfig{
    Mode:               "required",
    CertFile:           "/path/to/cert.pem",
    KeyFile:            "/path/to/key.pem",
    CAFile:             "/path/to/ca.pem",
    ServerName:         "db.example.com",
    InsecureSkipVerify: false,
}
builder.TLSWithConfig(tlsConfig)
```

## Compression and Performance

### Enable Compression

```go
builder.EnableCompression()  // Enable MySQL compression
builder.DisableCompression() // Disable MySQL compression
```

### Timeout Configuration

```go
builder.
    SetTimeout(30*time.Second).        // Connection timeout
    SetReadTimeout(10*time.Second).    // Read timeout
    SetWriteTimeout(10*time.Second)    // Write timeout
```

## Character Set and Collation

### Character Set Configuration

```go
builder.
    SetCharset("utf8mb4").                    // Character set
    SetCollation("utf8mb4_unicode_ci").       // Collation
    EnableParseTime().                        // Parse TIME/DATE values
    SetLocation("America/New_York")           // Timezone location
```

## Advanced Parameters

### MySQL-Specific Settings

```go
builder.
    SetSQLMode("STRICT_TRANS_TABLES,NO_ZERO_DATE").
    SetTimeZone("UTC").
    SetAutoCommit(true).
    SetTransactionIsolation("READ-COMMITTED").
    EnableMultiStatements().
    EnableInterpolateParams().
    SetMaxAllowedPacket(16777216).
    SetNetBufferLength(32768)
```

### Custom Parameters

```go
builder.
    SetParam("innodb_lock_wait_timeout", "50").
    SetParam("wait_timeout", "28800").
    SetParam("interactive_timeout", "28800")
```

## Validation and Error Handling

### Configuration Validation

```go
builder := ygggo.NewDSNBuilder().
    Host("localhost").
    Username("user").
    Password("pass").
    Database("mydb")

// Validate configuration
if err := builder.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}

// Build with validation
dsn, err := builder.BuildWithValidation()
if err != nil {
    log.Fatalf("Build failed: %v", err)
}
```

### Common Validation Errors

- Missing required fields (host, database)
- Invalid port numbers (must be 1-65535)
- Invalid timeout values (must be positive)
- Invalid TLS configuration

## Config Integration

### Convert to Config Struct

```go
config := ygggo.NewDSNBuilder().
    ProductionPreset().
    Host("db.example.com").
    Username("user").
    Password("pass").
    Database("mydb").
    ToConfig()

// Use with connection pool
pool, err := ygggo.NewPool(ctx, config)
```

### Create from Existing Config

```go
existingConfig := ygggo.Config{
    Host:     "localhost",
    Port:     3306,
    Username: "user",
    Password: "pass",
    Database: "mydb",
}

builder := ygggo.FromConfig(existingConfig)
dsn := builder.
    EnableCompression().
    SetCharset("utf8mb4").
    Build()
```

## Configuration Cloning

### Clone and Modify

```go
baseBuilder := ygggo.NewDSNBuilder().
    ProductionPreset().
    Host("db.example.com").
    Username("user").
    Password("pass")

// Clone for different databases
userDB := baseBuilder.Clone().Database("users").Build()
orderDB := baseBuilder.Clone().Database("orders").Build()
```

## Best Practices

### 1. Use Preset Configurations

Start with preset configurations and customize as needed:

```go
// Good: Start with preset
dsn := ygggo.NewDSNBuilder().
    ProductionPreset().
    Host("my-db.com").
    Username("user").
    Password("pass").
    Database("mydb").
    Build()
```

### 2. Always Validate in Production

```go
builder := ygggo.NewDSNBuilder().ProductionPreset()
// ... configure builder ...

dsn, err := builder.BuildWithValidation()
if err != nil {
    log.Fatalf("Invalid DSN configuration: %v", err)
}
```

### 3. Use TLS in Production

```go
// Production should always use TLS
builder.RequireTLS()

// Development can skip TLS
if isDevelopment {
    builder.DisableTLS()
}
```

### 4. Set Appropriate Timeouts

```go
// Production: Conservative timeouts
builder.
    SetTimeout(30*time.Second).
    SetReadTimeout(10*time.Second).
    SetWriteTimeout(10*time.Second)

// Testing: Fast timeouts
builder.
    SetTimeout(5*time.Second).
    SetReadTimeout(2*time.Second).
    SetWriteTimeout(2*time.Second)
```

### 5. Use UTF8MB4 Character Set

```go
// Always use utf8mb4 for full Unicode support
builder.SetCharset("utf8mb4")
```

## Common Patterns

### Environment-Based Configuration

```go
func createDSN(env string, host, user, pass, db string) string {
    builder := ygggo.NewDSNBuilder().
        Host(host).
        Username(user).
        Password(pass).
        Database(db)
    
    switch env {
    case "production":
        return builder.ProductionPreset().Build()
    case "testing":
        return builder.TestingPreset().Build()
    default:
        return builder.DevelopmentPreset().Build()
    }
}
```

### Multi-Database Configuration

```go
type DatabaseConfig struct {
    Users     string
    Orders    string
    Inventory string
}

func createMultiDBConfig(baseBuilder *ygggo.DSNBuilder) DatabaseConfig {
    return DatabaseConfig{
        Users:     baseBuilder.Clone().Database("users").Build(),
        Orders:    baseBuilder.Clone().Database("orders").Build(),
        Inventory: baseBuilder.Clone().Database("inventory").Build(),
    }
}
```

## API Reference

### Core Methods

- `NewDSNBuilder()` - Create new DSN builder
- `Host(string)` - Set database host
- `Port(int)` - Set database port
- `Username(string)` - Set username
- `Password(string)` - Set password
- `Database(string)` - Set database name
- `Build()` - Build DSN string
- `BuildWithValidation()` - Build with validation
- `Validate()` - Validate configuration
- `Clone()` - Clone builder

### TLS Methods

- `DisableTLS()` - Disable TLS
- `RequireTLS()` - Require TLS
- `TLSSkipVerify()` - TLS with skip verification
- `TLSCustom(string)` - Custom TLS config name
- `TLSWithCertificates(cert, key, ca)` - TLS with certificates
- `TLSWithConfig(*TLSConfig)` - Custom TLS configuration

### Performance Methods

- `EnableCompression()` / `DisableCompression()` - MySQL compression
- `SetTimeout(duration)` - Connection timeout
- `SetReadTimeout(duration)` - Read timeout
- `SetWriteTimeout(duration)` - Write timeout

### Character Set Methods

- `SetCharset(string)` - Character set
- `SetCollation(string)` - Collation
- `EnableParseTime()` / `DisableParseTime()` - Parse time values
- `SetLocation(string)` - Timezone location

### Preset Methods

- `DevelopmentPreset()` - Development configuration
- `ProductionPreset()` - Production configuration
- `TestingPreset()` - Testing configuration
- `HighPerformancePreset()` - High performance configuration
- `SecurePreset()` - Security-focused configuration

### Integration Methods

- `ToConfig()` - Convert to Config struct
- `FromConfig(Config)` - Create from Config struct
- `SetParam(key, value)` - Set custom parameter
