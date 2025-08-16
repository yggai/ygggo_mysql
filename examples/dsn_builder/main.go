package main

import (
	"fmt"
	"time"

	ygggo "github.com/yggai/ygggo_mysql"
)

func main() {
	fmt.Println("=== DSN Builder Examples ===\n")

	// Example 1: Basic DSN Construction
	fmt.Println("1. Basic DSN Construction:")
	basicDSNExample()

	// Example 2: Development Environment
	fmt.Println("\n2. Development Environment Preset:")
	developmentPresetExample()

	// Example 3: Production Environment
	fmt.Println("\n3. Production Environment Preset:")
	productionPresetExample()

	// Example 4: Testing Environment
	fmt.Println("\n4. Testing Environment Preset:")
	testingPresetExample()

	// Example 5: High Performance Configuration
	fmt.Println("\n5. High Performance Configuration:")
	highPerformanceExample()

	// Example 6: Secure Configuration
	fmt.Println("\n6. Secure Configuration:")
	secureConfigurationExample()

	// Example 7: Custom TLS Configuration
	fmt.Println("\n7. Custom TLS Configuration:")
	customTLSExample()

	// Example 8: Advanced Parameters
	fmt.Println("\n8. Advanced Parameters:")
	advancedParametersExample()

	// Example 9: Configuration Validation
	fmt.Println("\n9. Configuration Validation:")
	validationExample()

	// Example 10: Config Integration
	fmt.Println("\n10. Config Integration:")
	configIntegrationExample()

	fmt.Println("\n=== DSN Builder Examples Complete ===")
}

func basicDSNExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		Host("localhost").
		Port(3306).
		Username("myuser").
		Password("mypassword").
		Database("mydatabase").
		Build()

	fmt.Printf("Basic DSN: %s\n", dsn)
}

func developmentPresetExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		DevelopmentPreset().
		Host("localhost").
		Username("devuser").
		Password("devpass").
		Database("development_db").
		Build()

	fmt.Printf("Development DSN: %s\n", dsn)
	fmt.Println("Features: No TLS, UTF8MB4, ParseTime enabled, Local timezone")
}

func productionPresetExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		ProductionPreset().
		Host("prod-db.example.com").
		Username("produser").
		Password("secure_prod_password").
		Database("production_db").
		Build()

	fmt.Printf("Production DSN: %s\n", dsn)
	fmt.Println("Features: TLS required, Compression enabled, UTC timezone, Conservative timeouts")
}

func testingPresetExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		TestingPreset().
		Host("test-db").
		Username("testuser").
		Password("testpass").
		Database("test_db").
		Build()

	fmt.Printf("Testing DSN: %s\n", dsn)
	fmt.Println("Features: Fast timeouts, UTF8MB4, UTC timezone")
}

func highPerformanceExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		HighPerformancePreset().
		Host("perf-db.example.com").
		Username("perfuser").
		Password("perfpass").
		Database("performance_db").
		Build()

	fmt.Printf("High Performance DSN: %s\n", dsn)
	fmt.Println("Features: Compression enabled, Optimized timeouts, Performance parameters")
}

func secureConfigurationExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		SecurePreset().
		Host("secure-db.example.com").
		Username("secureuser").
		Password("very_secure_password").
		Database("secure_db").
		Build()

	fmt.Printf("Secure DSN: %s\n", dsn)
	fmt.Println("Features: TLS required, SERIALIZABLE isolation, Strict SQL mode")
}

func customTLSExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		Host("tls-db.example.com").
		Username("tlsuser").
		Password("tlspass").
		Database("tls_db").
		TLSCustom("custom-tls-config").
		Build()

	fmt.Printf("Custom TLS DSN: %s\n", dsn)
	fmt.Println("Features: Custom TLS configuration (requires registration with driver)")

	// Example with TLS skip verify (for development/testing)
	builder2 := ygggo.NewDSNBuilder()
	dsn2 := builder2.
		Host("dev-tls-db.example.com").
		Username("devuser").
		Password("devpass").
		Database("dev_db").
		TLSSkipVerify().
		Build()

	fmt.Printf("TLS Skip Verify DSN: %s\n", dsn2)
	fmt.Println("Features: TLS enabled but certificate verification skipped")
}

func advancedParametersExample() {
	builder := ygggo.NewDSNBuilder()
	dsn := builder.
		Host("advanced-db.example.com").
		Username("advuser").
		Password("advpass").
		Database("advanced_db").
		SetCharset("utf8mb4").
		SetCollation("utf8mb4_unicode_ci").
		SetSQLMode("STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE").
		SetTimeZone("America/New_York").
		SetAutoCommit(true).
		SetTransactionIsolation("READ-COMMITTED").
		EnableMultiStatements().
		EnableInterpolateParams().
		SetTimeout(45 * time.Second).
		SetReadTimeout(15 * time.Second).
		SetWriteTimeout(15 * time.Second).
		EnableCompression().
		EnableParseTime().
		Build()

	fmt.Printf("Advanced DSN: %s\n", dsn)
	fmt.Println("Features: Custom charset/collation, SQL mode, timezone, multi-statements")
}

func validationExample() {
	// Valid configuration
	builder := ygggo.NewDSNBuilder()
	builder.
		Host("localhost").
		Username("user").
		Password("pass").
		Database("db").
		SetTimeout(30 * time.Second)

	if err := builder.Validate(); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Configuration is valid")
		dsn, err := builder.BuildWithValidation()
		if err != nil {
			fmt.Printf("Build failed: %v\n", err)
		} else {
			fmt.Printf("Validated DSN: %s\n", dsn)
		}
	}

	// Invalid configuration example
	invalidBuilder := ygggo.NewDSNBuilder()
	invalidBuilder.Port(70000) // Invalid port

	if err := invalidBuilder.Validate(); err != nil {
		fmt.Printf("❌ Invalid configuration detected: %v\n", err)
	}
}

func configIntegrationExample() {
	// Create DSN builder and convert to Config
	builder := ygggo.NewDSNBuilder()
	config := builder.
		Host("config-db.example.com").
		Port(3307).
		Username("configuser").
		Password("configpass").
		Database("config_db").
		ProductionPreset().
		ToConfig()

	fmt.Printf("Generated Config:\n")
	fmt.Printf("  Driver: %s\n", config.Driver)
	fmt.Printf("  Host: %s\n", config.Host)
	fmt.Printf("  Port: %d\n", config.Port)
	fmt.Printf("  Username: %s\n", config.Username)
	fmt.Printf("  Database: %s\n", config.Database)
	fmt.Printf("  DSN: %s\n", config.DSN)

	// Create builder from existing config
	newBuilder := ygggo.FromConfig(config)
	newDSN := newBuilder.
		SetParam("additional", "parameter").
		Build()

	fmt.Printf("Modified DSN: %s\n", newDSN)
}

// Example of using DSN builder with connection pool
func connectionPoolExample() {
	builder := ygggo.NewDSNBuilder()
	config := builder.
		ProductionPreset().
		Host("prod-db.example.com").
		Username("produser").
		Password("prodpass").
		Database("production").
		ToConfig()

	// Set pool configuration
	config.Pool = ygggo.PoolConfig{
		MaxOpen:         25,
		MaxIdle:         10,
		ConnMaxLifetime: 5 * time.Minute,
	}

	fmt.Printf("Pool Config DSN: %s\n", config.DSN)
	fmt.Printf("Pool Settings: MaxOpen=%d, MaxIdle=%d, ConnMaxLifetime=%v\n",
		config.Pool.MaxOpen, config.Pool.MaxIdle, config.Pool.ConnMaxLifetime)

	// In a real application, you would create the pool like this:
	// pool, err := ygggo.NewPool(context.Background(), config)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// defer pool.Close()
}

// Example of cloning and modifying configurations
func configurationCloningExample() {
	// Base configuration
	baseBuilder := ygggo.NewDSNBuilder()
	baseBuilder.
		Host("base-db.example.com").
		Username("baseuser").
		Password("basepass").
		ProductionPreset()

	// Clone for different databases
	userDBBuilder := baseBuilder.Clone()
	userDSN := userDBBuilder.Database("users").Build()

	orderDBBuilder := baseBuilder.Clone()
	orderDSN := orderDBBuilder.Database("orders").Build()

	inventoryDBBuilder := baseBuilder.Clone()
	inventoryDSN := inventoryDBBuilder.Database("inventory").Build()

	fmt.Printf("User DB DSN: %s\n", userDSN)
	fmt.Printf("Order DB DSN: %s\n", orderDSN)
	fmt.Printf("Inventory DB DSN: %s\n", inventoryDSN)
}
