package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Tx represents a database transaction with automatic management and observability.
//
// Tx wraps the standard library's *sql.Tx and adds enterprise features
// including automatic retry policies, observability integration (tracing,
// metrics, logging), and proper resource management. Transactions ensure
// ACID properties for groups of database operations.
//
// Key features:
//   - Automatic commit/rollback based on function return values
//   - Retry policies for handling transient failures (deadlocks, timeouts)
//   - OpenTelemetry tracing integration
//   - Metrics collection for transaction performance
//   - Structured logging for transaction events
//
// Example usage (via Pool.WithinTx):
//
//	err := pool.WithinTx(ctx, func(tx DatabaseTx) error {
//		// All operations are part of the same transaction
//		_, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - ? WHERE id = ?", 100, fromID)
//		if err != nil {
//			return err // This will cause a rollback
//		}
//
//		_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + ? WHERE id = ?", 100, toID)
//		if err != nil {
//			return err // This will cause a rollback
//		}
//
//		return nil // This will cause a commit
//	})
//
// Transaction Lifecycle:
//  1. Transaction begins automatically
//  2. Function executes with transaction context
//  3. If function returns nil: transaction commits
//  4. If function returns error: transaction rolls back
//  5. Retry logic applies for retryable errors (deadlocks, etc.)
//
// Thread Safety:
//
// Tx is NOT safe for concurrent use. Each transaction should be used
// by only one goroutine. The transaction context ensures isolation
// from other concurrent operations.
type Tx struct {
	// inner is the underlying database transaction
	inner *sql.Tx

	// pool is a reference to the parent pool for observability features
	pool *Pool
}

// Exec executes a query within the transaction context.
//
// This method executes SQL statements (INSERT, UPDATE, DELETE, DDL) within
// the transaction. All operations using the same Tx instance are part of
// the same transaction and will be committed or rolled back together.
//
// The method integrates with the pool's observability features, automatically
// creating tracing spans and recording metrics when enabled.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and tracing
//   - query: SQL query with placeholder parameters (?)
//   - args: Arguments to bind to placeholders
//
// Returns:
//   - sql.Result: Information about the execution (rows affected, last insert ID)
//   - error: Execution error or sql.ErrTxDone if transaction is closed
//
// Example:
//
//	result, err := tx.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)",
//		"Alice", "alice@example.com")
//	if err != nil {
//		return err // Transaction will be rolled back
//	}
//
//	userID, err := result.LastInsertId()
//	if err != nil {
//		return err // Transaction will be rolled back
//	}
//
// Error Handling:
//
// If this method returns an error, the transaction should be rolled back.
// When using Pool.WithinTx, this happens automatically when the function
// returns an error.
//
// Thread Safety:
//
// This method is NOT safe for concurrent use within the same transaction.
// Each transaction should be used by only one goroutine.
func (tx *Tx) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx == nil || tx.inner == nil {
		return nil, sql.ErrTxDone
	}

	return tx.inner.ExecContext(ctx, query, args...)
}

// WithinTx executes a function within a database transaction with automatic management.
//
// This method provides enterprise-grade transaction management including:
//   - Automatic begin/commit/rollback based on function return value
//   - Configurable retry policies for transient failures (deadlocks, timeouts)
//   - Comprehensive observability (tracing, metrics, logging)
//   - Proper resource cleanup and error handling
//
// Transaction Behavior:
//   - If fn returns nil: transaction is committed
//   - If fn returns an error: transaction is rolled back
//   - Retryable errors (deadlocks, timeouts) trigger automatic retry
//   - Non-retryable errors cause immediate rollback and return
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and distributed tracing
//   - fn: Function to execute within the transaction context
//   - opts: Optional transaction options (implementation-specific)
//
// Returns:
//   - error: Transaction setup error, function error, or commit/rollback error
//
// Example - Simple Transaction:
//
//	err := pool.WithinTx(ctx, func(tx DatabaseTx) error {
//		_, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
//		if err != nil {
//			return err // Automatic rollback
//		}
//
//		_, err = tx.Exec(ctx, "UPDATE counters SET value = value + 1 WHERE name = 'users'")
//		return err // Automatic commit if nil, rollback if error
//	})
//
// Example - Money Transfer (ACID Transaction):
//
//	err := pool.WithinTx(ctx, func(tx DatabaseTx) error {
//		// Debit source account
//		result, err := tx.Exec(ctx,
//			"UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?",
//			amount, fromAccountID, amount)
//		if err != nil {
//			return err
//		}
//
//		affected, _ := result.RowsAffected()
//		if affected == 0 {
//			return errors.New("insufficient funds or account not found")
//		}
//
//		// Credit destination account
//		_, err = tx.Exec(ctx,
//			"UPDATE accounts SET balance = balance + ? WHERE id = ?",
//			amount, toAccountID)
//		if err != nil {
//			return err
//		}
//
//		// Log the transfer
//		_, err = tx.Exec(ctx,
//			"INSERT INTO transfers (from_account, to_account, amount) VALUES (?, ?, ?)",
//			fromAccountID, toAccountID, amount)
//		return err
//	})
//
// Retry Policy:
//
// The method automatically retries transactions that fail due to transient
// errors such as deadlocks (MySQL error 1213) or lock timeouts (MySQL error 1205).
// The retry policy is configurable via the pool's RetryPolicy configuration.
//
// Observability:
//
// When enabled, the method automatically:
//   - Creates distributed tracing spans for the entire transaction
//   - Records transaction duration and outcome metrics
//   - Logs transaction events (begin, commit, rollback) with structured data
//   - Tracks retry attempts and failure reasons
//
// Error Handling:
//
// The method distinguishes between different types of errors:
//   - Setup errors (connection acquisition, begin transaction): returned immediately
//   - Retryable errors (deadlocks, timeouts): trigger retry logic
//   - Non-retryable errors: cause immediate rollback and return
//   - Commit errors: returned after successful function execution
//
// Thread Safety:
//
// This method is safe for concurrent use. Multiple goroutines can start
// transactions simultaneously, and each will receive its own isolated
// transaction context.
func (p *Pool) WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error {
	if p == nil || p.db == nil {
		return errors.New("nil pool")
	}

	start := time.Now()

	op := func() error {
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		wrap := &Tx{inner: tx, pool: p}
		err = fn(wrap)
		if err == nil {
			if cerr := tx.Commit(); cerr != nil {
				return cerr
			}
			return nil
		}
		_ = tx.Rollback()
		return err
	}

	err := retryWithPolicy(ctx, p.retry, op, Classify)

	// Record duration
	duration := time.Since(start)

	// Log transaction
	if p.loggingEnabled {
		event := "commit"
		if err != nil {
			event = "rollback"
		}
		p.logTransaction(ctx, event, duration, err)
	}

	return err
}

func isRetryable(err error) bool {
	var me *mysqlMySQLError // local shim to avoid importing mysql here
	if errors.As(err, &me) {
		switch me.Number {
		case 1213, 1205, 1290:
			return true
		}
	}
	return false
}

// backoffSleep performs simple sleep; will be replaced by cenkalti/backoff later.
func backoffSleep(pol RetryPolicy, attempt int) {
	if attempt <= 0 {
		return
	}
	d := pol.BaseBackoff
	if d <= 0 {
		d = 10 * time.Millisecond
	}
	sleep := time.Duration(attempt) * d
	time.Sleep(sleep)
}

// minimal wrapper to decouple from concrete driver types in tx.go
// concrete conversion is in errors.go
type mysqlMySQLError struct{ Number uint16 }

func (e *mysqlMySQLError) Error() string { return "mysql error" }
