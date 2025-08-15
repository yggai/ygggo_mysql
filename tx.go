package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Tx wraps *sql.Tx and shares some methods with Conn.
type Tx struct{
	inner *sql.Tx
	pool  *Pool
}

// Exec executes within the transaction.
func (tx *Tx) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx == nil || tx.inner == nil { return nil, sql.ErrTxDone }
	if tx.pool != nil && tx.pool.telemetryEnabled {
		ctx, span := tx.pool.startSpan(ctx, "exec", query)
		result, err := tx.inner.ExecContext(ctx, query, args...)
		tx.pool.finishSpan(span, err)
		return result, err
	}
	return tx.inner.ExecContext(ctx, query, args...)
}

// WithinTx executes fn within a transaction using retryWithPolicy for retryable errors.
func (p *Pool) WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error {
	if p == nil || p.db == nil { return errors.New("nil pool") }

	var span trace.Span
	if p.telemetryEnabled {
		ctx, span = p.startSpan(ctx, "transaction", "")
	}

	op := func() error {
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil { return err }
		wrap := &Tx{inner: tx, pool: p}
		err = fn(wrap)
		if err == nil {
			if cerr := tx.Commit(); cerr != nil { return cerr }
			return nil
		}
		_ = tx.Rollback()
		return err
	}

	err := retryWithPolicy(ctx, p.retry, op, Classify)

	if p.telemetryEnabled {
		p.finishSpan(span, err)
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
	if attempt <= 0 { return }
	d := pol.BaseBackoff
	if d <= 0 { d = 10 * time.Millisecond }
	sleep := time.Duration(attempt) * d
	time.Sleep(sleep)
}

// minimal wrapper to decouple from concrete driver types in tx.go
// concrete conversion is in errors.go
type mysqlMySQLError struct{ Number uint16 }

func (e *mysqlMySQLError) Error() string { return "mysql error" }
