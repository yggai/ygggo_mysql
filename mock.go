package ygggo_mysql

import (
	"context"
	"database/sql/driver"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockExpectations provides a simplified interface for setting mock expectations
// without exposing the underlying sqlmock types to users.
type MockExpectations interface {
	ExpectPing()
	ExpectQuery(query string) QueryExpectation
	ExpectExec(query string) ExecExpectation
	ExpectBegin()
	ExpectCommit()
	ExpectRollback()
	ExpectPrepare(query string) PrepareExpectation
	ExpectationsWereMet() error
}

type QueryExpectation interface {
	WillReturnRows(rows MockRows) QueryExpectation
	WithArgs(args ...interface{}) QueryExpectation
}

type ExecExpectation interface {
	WillReturnResult(result MockResult) ExecExpectation
	WillReturnError(err error) ExecExpectation
	WithArgs(args ...interface{}) ExecExpectation
}

type PrepareExpectation interface {
	ExpectQuery() QueryExpectation
	ExpectExec() ExecExpectation
}

type MockRows interface{}
type MockResult interface{}

// mockWrapper wraps sqlmock.Sqlmock to implement MockExpectations
type mockWrapper struct {
	mock sqlmock.Sqlmock
}

func (m *mockWrapper) ExpectPing() {
	m.mock.ExpectPing()
}

func (m *mockWrapper) ExpectQuery(query string) QueryExpectation {
	return &queryExpWrapper{m.mock.ExpectQuery(query)}
}

func (m *mockWrapper) ExpectExec(query string) ExecExpectation {
	return &execExpWrapper{m.mock.ExpectExec(query)}
}

func (m *mockWrapper) ExpectBegin() {
	m.mock.ExpectBegin()
}

func (m *mockWrapper) ExpectCommit() {
	m.mock.ExpectCommit()
}

func (m *mockWrapper) ExpectRollback() {
	m.mock.ExpectRollback()
}

func (m *mockWrapper) ExpectPrepare(query string) PrepareExpectation {
	return &prepareExpWrapper{m.mock.ExpectPrepare(query)}
}

func (m *mockWrapper) ExpectationsWereMet() error {
	return m.mock.ExpectationsWereMet()
}

type queryExpWrapper struct {
	exp *sqlmock.ExpectedQuery
}

func (q *queryExpWrapper) WillReturnRows(rows MockRows) QueryExpectation {
	if r, ok := rows.(*sqlmock.Rows); ok {
		q.exp.WillReturnRows(r)
	}
	return q
}

func (q *queryExpWrapper) WithArgs(args ...interface{}) QueryExpectation {
	driverArgs := make([]driver.Value, len(args))
	for i, arg := range args {
		driverArgs[i] = arg
	}
	q.exp.WithArgs(driverArgs...)
	return q
}

type execExpWrapper struct {
	exp *sqlmock.ExpectedExec
}

func (e *execExpWrapper) WillReturnResult(result MockResult) ExecExpectation {
	if r, ok := result.(driver.Result); ok {
		e.exp.WillReturnResult(r)
	}
	return e
}

func (e *execExpWrapper) WillReturnError(err error) ExecExpectation {
	e.exp.WillReturnError(err)
	return e
}

func (e *execExpWrapper) WithArgs(args ...interface{}) ExecExpectation {
	driverArgs := make([]driver.Value, len(args))
	for i, arg := range args {
		driverArgs[i] = arg
	}
	e.exp.WithArgs(driverArgs...)
	return e
}

type prepareExpWrapper struct {
	exp *sqlmock.ExpectedPrepare
}

func (p *prepareExpWrapper) ExpectQuery() QueryExpectation {
	return &queryExpWrapper{p.exp.ExpectQuery()}
}

func (p *prepareExpWrapper) ExpectExec() ExecExpectation {
	return &execExpWrapper{p.exp.ExpectExec()}
}

// NewRows creates a new MockRows for use with mock expectations
func NewRows(columns []string) MockRows {
	return sqlmock.NewRows(columns)
}

// AddRow adds a row to MockRows
func AddRow(rows MockRows, values ...interface{}) MockRows {
	if r, ok := rows.(*sqlmock.Rows); ok {
		driverValues := make([]driver.Value, len(values))
		for i, v := range values {
			driverValues[i] = v
		}
		return r.AddRow(driverValues...)
	}
	return rows
}

// NewResult creates a new MockResult for use with mock expectations
func NewResult(lastInsertId, rowsAffected int64) MockResult {
	return sqlmock.NewResult(lastInsertId, rowsAffected)
}

// NewMockPool creates a Pool backed by sqlmock for testing.
// Returns the Pool, the MockExpectations for setting expectations, and any error.
func NewMockPool(ctx context.Context, cfg Config) (*Pool, MockExpectations, error) {
	// Apply env overrides first (convention over configuration)
	applyEnv(&cfg)

	// Create sqlmock DB
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		return nil, nil, err
	}

	p := &Pool{db: db}
	// Apply retry policy from config
	p.retry = cfg.Retry

	// Apply pool settings (though they may not be meaningful for mock)
	if cfg.Pool.MaxOpen > 0 {
		db.SetMaxOpenConns(cfg.Pool.MaxOpen)
	}
	if cfg.Pool.MaxIdle > 0 {
		db.SetMaxIdleConns(cfg.Pool.MaxIdle)
	}
	if cfg.Pool.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.Pool.ConnMaxLifetime)
	}
	if cfg.Pool.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.Pool.ConnMaxIdleTime)
	}

	return p, &mockWrapper{mock}, nil
}

// NewPoolWithMock creates either a real Pool or mock Pool based on isMock flag.
func NewPoolWithMock(ctx context.Context, cfg Config, isMock bool) (*Pool, MockExpectations, error) {
	if isMock {
		return NewMockPool(ctx, cfg)
	}
	pool, err := NewPool(ctx, cfg)
	return pool, nil, err
}
