package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// MockExpectations provides a simplified interface for setting mock expectations
// without exposing the underlying implementation details to users.
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

// SQLite-based mock implementation
type sqliteMockExpectation struct {
	expectType   string        // "query", "exec", "ping", "begin", "commit", "rollback", "prepare"
	sql          string        // Expected SQL (with regex support)
	args         []interface{} // Expected arguments
	returnRows   MockRows      // For queries
	returnResult MockResult    // For exec
	returnError  error         // Error to return
	matched      bool          // Whether this expectation was matched
}

type sqliteMockExpectations struct {
	helper       *SQLiteTestHelper
	expectations []sqliteMockExpectation
	currentIndex int
	mutex        sync.RWMutex
	inTx         bool // Track transaction state
}

type sqliteQueryExpectation struct {
	parent *sqliteMockExpectations
	index  int
}

type sqliteExecExpectation struct {
	parent *sqliteMockExpectations
	index  int
}

type sqlitePrepareExpectation struct {
	parent *sqliteMockExpectations
	index  int
}

// SQLite-based MockRows implementation
type sqliteRows struct {
	columns []string
	rows    [][]interface{}
}

// SQLite-based MockResult implementation
type sqliteResult struct {
	lastInsertId int64
	rowsAffected int64
}

// mockPool wraps a real Pool and intercepts calls to match expectations
type mockPool struct {
	*Pool
	expectations *sqliteMockExpectations
}

// mockConn wraps a DatabaseConn to intercept calls
type mockConn struct {
	DatabaseConn
	expectations *sqliteMockExpectations
}

// mockTx wraps a DatabaseTx to intercept calls
type mockTx struct {
	DatabaseTx
	expectations *sqliteMockExpectations
}

// SQLite-based MockExpectations implementation
func (m *sqliteMockExpectations) ExpectPing() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "ping",
	})
}

func (m *sqliteMockExpectations) ExpectQuery(query string) QueryExpectation {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	index := len(m.expectations)
	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "query",
		sql:        query,
	})

	return &sqliteQueryExpectation{parent: m, index: index}
}

func (m *sqliteMockExpectations) ExpectExec(query string) ExecExpectation {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	index := len(m.expectations)
	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "exec",
		sql:        query,
	})

	return &sqliteExecExpectation{parent: m, index: index}
}

func (m *sqliteMockExpectations) ExpectBegin() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "begin",
	})
}

func (m *sqliteMockExpectations) ExpectCommit() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "commit",
	})
}

func (m *sqliteMockExpectations) ExpectRollback() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "rollback",
	})
}

func (m *sqliteMockExpectations) ExpectPrepare(query string) PrepareExpectation {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	index := len(m.expectations)
	m.expectations = append(m.expectations, sqliteMockExpectation{
		expectType: "prepare",
		sql:        query,
	})

	return &sqlitePrepareExpectation{parent: m, index: index}
}

func (m *sqliteMockExpectations) ExpectationsWereMet() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for i, exp := range m.expectations {
		if !exp.matched {
			return fmt.Errorf("expectation %d was not matched: %s %s", i, exp.expectType, exp.sql)
		}
	}
	return nil
}

// SQLite QueryExpectation implementation
func (q *sqliteQueryExpectation) WillReturnRows(rows MockRows) QueryExpectation {
	q.parent.mutex.Lock()
	defer q.parent.mutex.Unlock()

	if q.index < len(q.parent.expectations) {
		q.parent.expectations[q.index].returnRows = rows
	}
	return q
}

func (q *sqliteQueryExpectation) WithArgs(args ...interface{}) QueryExpectation {
	q.parent.mutex.Lock()
	defer q.parent.mutex.Unlock()

	if q.index < len(q.parent.expectations) {
		q.parent.expectations[q.index].args = args
	}
	return q
}

// SQLite ExecExpectation implementation
func (e *sqliteExecExpectation) WillReturnResult(result MockResult) ExecExpectation {
	e.parent.mutex.Lock()
	defer e.parent.mutex.Unlock()

	if e.index < len(e.parent.expectations) {
		e.parent.expectations[e.index].returnResult = result
	}
	return e
}

func (e *sqliteExecExpectation) WillReturnError(err error) ExecExpectation {
	e.parent.mutex.Lock()
	defer e.parent.mutex.Unlock()

	if e.index < len(e.parent.expectations) {
		e.parent.expectations[e.index].returnError = err
	}
	return e
}

func (e *sqliteExecExpectation) WithArgs(args ...interface{}) ExecExpectation {
	e.parent.mutex.Lock()
	defer e.parent.mutex.Unlock()

	if e.index < len(e.parent.expectations) {
		e.parent.expectations[e.index].args = args
	}
	return e
}

// SQLite PrepareExpectation implementation
func (p *sqlitePrepareExpectation) ExpectQuery() QueryExpectation {
	p.parent.mutex.Lock()
	defer p.parent.mutex.Unlock()

	// Get the SQL from the prepare expectation
	var sql string
	if p.index < len(p.parent.expectations) {
		sql = p.parent.expectations[p.index].sql
	}

	index := len(p.parent.expectations)
	p.parent.expectations = append(p.parent.expectations, sqliteMockExpectation{
		expectType: "prepare_query",
		sql:        sql,
	})

	return &sqliteQueryExpectation{parent: p.parent, index: index}
}

func (p *sqlitePrepareExpectation) ExpectExec() ExecExpectation {
	p.parent.mutex.Lock()
	defer p.parent.mutex.Unlock()

	// Get the SQL from the prepare expectation
	var sql string
	if p.index < len(p.parent.expectations) {
		sql = p.parent.expectations[p.index].sql
	}

	index := len(p.parent.expectations)
	p.parent.expectations = append(p.parent.expectations, sqliteMockExpectation{
		expectType: "prepare_exec",
		sql:        sql,
	})

	return &sqliteExecExpectation{parent: p.parent, index: index}
}

// SQLite MockRows implementation
func (r *sqliteRows) AddRow(values ...interface{}) MockRows {
	r.rows = append(r.rows, values)
	return r
}

// SQLite MockResult implementation
func (r *sqliteResult) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r *sqliteResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// NewRows creates a new MockRows for use with mock expectations
func NewRows(columns []string) MockRows {
	return &sqliteRows{
		columns: columns,
		rows:    make([][]interface{}, 0),
	}
}

// AddRow adds a row to MockRows
func AddRow(rows MockRows, values ...interface{}) MockRows {
	if r, ok := rows.(*sqliteRows); ok {
		return r.AddRow(values...)
	}
	return rows
}

// NewResult creates a new MockResult for use with mock expectations
func NewResult(lastInsertId, rowsAffected int64) MockResult {
	return &sqliteResult{
		lastInsertId: lastInsertId,
		rowsAffected: rowsAffected,
	}
}

// NewMockPool creates a Pool backed by SQLite for testing.
// Returns the DatabasePool, the MockExpectations for setting expectations, and any error.
func NewMockPool(ctx context.Context, cfg Config) (DatabasePool, MockExpectations, error) {
	// Apply env overrides first (convention over configuration)
	applyEnv(&cfg)

	// Create SQLite test helper
	helper, err := NewSQLiteTestHelper(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SQLite test helper: %w", err)
	}

	// Get the pool from helper
	pool := helper.Pool()

	// Apply retry policy from config
	pool.retry = cfg.Retry

	// Create SQLite mock expectations
	mockExp := &sqliteMockExpectations{
		helper:       helper,
		expectations: make([]sqliteMockExpectation, 0),
		currentIndex: 0,
	}

	// Create a custom pool that intercepts calls
	mockPool := &mockPool{
		Pool:         pool,
		expectations: mockExp,
	}

	return mockPool, mockExp, nil
}

// Helper methods for SQL and argument matching
func (m *sqliteMockExpectations) matchSQL(expected, actual string) bool {
	// Normalize both strings for comparison
	expectedNorm := m.normalizeSQL(expected)
	actualNorm := m.normalizeSQL(actual)

	// Try regex matching first (for patterns with escaped characters)
	if strings.Contains(expected, "\\") {
		regex, err := regexp.Compile("(?i)^" + expected + "$")
		if err == nil && regex.MatchString(actual) {
			return true
		}
	}

	// Fallback to normalized string comparison
	return strings.EqualFold(expectedNorm, actualNorm)
}

func (m *sqliteMockExpectations) normalizeSQL(sql string) string {
	// Remove extra whitespace
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(sql), " ")
	// Unescape common regex escapes for comparison
	sql = strings.ReplaceAll(sql, `\(`, "(")
	sql = strings.ReplaceAll(sql, `\)`, ")")
	sql = strings.ReplaceAll(sql, `\?`, "?")
	return sql
}

func (m *sqliteMockExpectations) matchArgs(expected, actual []interface{}) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i, exp := range expected {
		if !reflect.DeepEqual(exp, actual[i]) {
			return false
		}
	}
	return true
}

func (m *sqliteMockExpectations) findMatchingExpectation(expectType, sql string, args []interface{}) (*sqliteMockExpectation, int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i := m.currentIndex; i < len(m.expectations); i++ {
		exp := &m.expectations[i]
		if exp.matched {
			continue
		}
		if exp.expectType == expectType {
			if expectType == "ping" || expectType == "begin" || expectType == "commit" || expectType == "rollback" {
				exp.matched = true
				m.currentIndex = i + 1
				return exp, i
			}
			if m.matchSQL(exp.sql, sql) && m.matchArgs(exp.args, args) {
				exp.matched = true
				m.currentIndex = i + 1
				return exp, i
			}
		}
	}
	return nil, -1
}

// mockPool methods that intercept calls
func (mp *mockPool) Ping(ctx context.Context) error {
	// Find matching ping expectation
	exp, _ := mp.expectations.findMatchingExpectation("ping", "", nil)
	if exp == nil {
		return fmt.Errorf("unexpected ping call")
	}

	if exp.returnError != nil {
		return exp.returnError
	}

	// Call the real ping
	return mp.Pool.Ping(ctx)
}

func (mp *mockPool) WithConn(ctx context.Context, fn func(DatabaseConn) error) error {
	return mp.Pool.WithConn(ctx, func(conn DatabaseConn) error {
		// Wrap the connection to intercept calls
		mockConn := &mockConn{
			DatabaseConn: conn,
			expectations: mp.expectations,
		}
		return fn(mockConn)
	})
}

func (mp *mockPool) WithinTx(ctx context.Context, fn func(DatabaseTx) error, opts ...any) error {
	// Check for begin expectation
	beginExp, _ := mp.expectations.findMatchingExpectation("begin", "", nil)
	if beginExp == nil {
		return fmt.Errorf("unexpected begin transaction call")
	}

	if beginExp.returnError != nil {
		return beginExp.returnError
	}

	// Execute the transaction
	err := mp.Pool.WithinTx(ctx, func(tx DatabaseTx) error {
		mockTx := &mockTx{
			DatabaseTx:   tx,
			expectations: mp.expectations,
		}

		return fn(mockTx)
	}, opts...)

	// After transaction is complete, check commit/rollback expectations
	if err != nil {
		// Transaction failed, should have rollback expectation
		rollbackExp, _ := mp.expectations.findMatchingExpectation("rollback", "", nil)
		if rollbackExp != nil && rollbackExp.returnError != nil {
			return rollbackExp.returnError
		}
	} else {
		// Transaction succeeded, should have commit expectation
		commitExp, _ := mp.expectations.findMatchingExpectation("commit", "", nil)
		if commitExp != nil && commitExp.returnError != nil {
			return commitExp.returnError
		}
	}

	return err
}

func (mp *mockPool) Acquire(ctx context.Context) (DatabaseConn, error) {
	conn, err := mp.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &mockConn{
		DatabaseConn: conn,
		expectations: mp.expectations,
	}, nil
}

func (mp *mockPool) SelfCheck(ctx context.Context) error {
	return mp.Pool.SelfCheck(ctx)
}

func (mp *mockPool) Close() error {
	return mp.Pool.Close()
}

// mockConn methods that intercept calls
func (mc *mockConn) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	// Find matching query expectation
	exp, _ := mc.expectations.findMatchingExpectation("query", query, args)
	if exp == nil {
		return nil, fmt.Errorf("unexpected query call: %s with args %v", query, args)
	}

	if exp.returnError != nil {
		return nil, exp.returnError
	}

	// Convert MockRows to actual rows
	if rows, ok := exp.returnRows.(*sqliteRows); ok {
		return mc.createRowsFromMockData(rows)
	}

	return nil, fmt.Errorf("invalid return rows data")
}

func (mc *mockConn) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// Find matching exec expectation
	exp, _ := mc.expectations.findMatchingExpectation("exec", query, args)
	if exp == nil {
		return nil, fmt.Errorf("unexpected exec call: %s with args %v", query, args)
	}

	if exp.returnError != nil {
		return nil, exp.returnError
	}

	// Return the mock result
	if result, ok := exp.returnResult.(*sqliteResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid return result data")
}

// createRowsFromMockData creates actual SQL rows from mock data
func (mc *mockConn) createRowsFromMockData(mockRows *sqliteRows) (*sql.Rows, error) {
	// Create a temporary table with the mock data
	tableName := fmt.Sprintf("mock_table_%p", mockRows)

	// Build column definitions (assume all TEXT for simplicity)
	var columnDefs []string
	for _, col := range mockRows.columns {
		columnDefs = append(columnDefs, fmt.Sprintf("%s TEXT", col))
	}

	// Create table
	createSQL := fmt.Sprintf("CREATE TEMP TABLE %s (%s)", tableName, strings.Join(columnDefs, ", "))
	_, err := mc.DatabaseConn.Exec(context.Background(), createSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp table: %w", err)
	}

	// Insert mock data
	if len(mockRows.rows) > 0 {
		placeholders := strings.Repeat("?,", len(mockRows.columns))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

		insertSQL := fmt.Sprintf("INSERT INTO %s VALUES (%s)", tableName, placeholders)

		for _, row := range mockRows.rows {
			_, err := mc.DatabaseConn.Exec(context.Background(), insertSQL, row...)
			if err != nil {
				return nil, fmt.Errorf("failed to insert mock data: %w", err)
			}
		}
	}

	// Query the data
	selectSQL := fmt.Sprintf("SELECT %s FROM %s", strings.Join(mockRows.columns, ", "), tableName)
	return mc.DatabaseConn.Query(context.Background(), selectSQL)
}

func (mc *mockConn) BulkInsert(ctx context.Context, table string, columns []string, rows [][]any) (sql.Result, error) {
	// Convert BulkInsert to equivalent SQL for matching
	placeholders := "(" + strings.Repeat("?,", len(columns))
	placeholders = placeholders[:len(placeholders)-1] + ")"

	var allPlaceholders []string
	var allArgs []any
	for _, row := range rows {
		allPlaceholders = append(allPlaceholders, placeholders)
		allArgs = append(allArgs, row...)
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		table,
		strings.Join(columns, ","),
		strings.Join(allPlaceholders, ","))

	// Find matching exec expectation
	exp, _ := mc.expectations.findMatchingExpectation("exec", sql, allArgs)
	if exp == nil {
		return nil, fmt.Errorf("unexpected bulk insert call: %s with args %v", sql, allArgs)
	}

	if exp.returnError != nil {
		return nil, exp.returnError
	}

	// Return the mock result
	if result, ok := exp.returnResult.(*sqliteResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid return result data")
}

func (mc *mockConn) NamedExec(ctx context.Context, query string, arg any) (sql.Result, error) {
	// Convert named parameters to positional parameters for matching
	convertedQuery, args, err := mc.convertNamedQuery(query, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert named query: %w", err)
	}

	// Find matching exec expectation
	exp, _ := mc.expectations.findMatchingExpectation("exec", convertedQuery, args)
	if exp == nil {
		return nil, fmt.Errorf("unexpected named exec call: %s with args %v", convertedQuery, args)
	}

	if exp.returnError != nil {
		return nil, exp.returnError
	}

	// Return the mock result
	if result, ok := exp.returnResult.(*sqliteResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid return result data")
}

// convertNamedQuery converts named parameters to positional parameters
func (mc *mockConn) convertNamedQuery(query string, arg any) (string, []any, error) {
	// Simple implementation: extract named parameters and convert to positional
	// This is a simplified version - a full implementation would use reflection

	if argMap, ok := arg.(map[string]any); ok {
		var args []any
		convertedQuery := query

		// Find all named parameters in the query
		re := regexp.MustCompile(`:(\w+)`)
		matches := re.FindAllStringSubmatch(query, -1)

		for _, match := range matches {
			paramName := match[1]
			if value, exists := argMap[paramName]; exists {
				args = append(args, value)
				// Replace the named parameter with ?
				convertedQuery = strings.Replace(convertedQuery, ":"+paramName, "?", 1)
			}
		}

		return convertedQuery, args, nil
	}

	return "", nil, fmt.Errorf("named parameters must be provided as map[string]any")
}

// Additional DatabaseConn methods that need to be intercepted
func (mc *mockConn) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	// For mock purposes, we'll delegate to the underlying connection
	// In a real implementation, this might need more sophisticated handling
	return mc.DatabaseConn.QueryRow(ctx, query, args...)
}

func (mc *mockConn) QueryStream(ctx context.Context, query string, cb func([]any) error, args ...any) error {
	// For mock purposes, delegate to underlying connection
	return mc.DatabaseConn.QueryStream(ctx, query, cb, args...)
}

func (mc *mockConn) EnableStmtCache(capacity int) {
	mc.DatabaseConn.EnableStmtCache(capacity)
}

func (mc *mockConn) ExecCached(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// ExecCached uses prepared statements
	// First, check if there's a prepare expectation that needs to be matched
	prepareExp, _ := mc.expectations.findMatchingExpectation("prepare", query, nil)
	if prepareExp != nil {
		// Prepare expectation found and matched, now look for prepare_exec
		exp, _ := mc.expectations.findMatchingExpectation("prepare_exec", query, args)
		if exp != nil {
			if exp.returnError != nil {
				return nil, exp.returnError
			}

			// Return the mock result
			if result, ok := exp.returnResult.(*sqliteResult); ok {
				return result, nil
			}

			return nil, fmt.Errorf("invalid return result data")
		}
	}

	// Fallback to regular exec expectation
	return mc.Exec(ctx, query, args...)
}

func (mc *mockConn) QueryCached(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	// QueryCached should behave like Query for mock purposes
	return mc.Query(ctx, query, args...)
}

func (mc *mockConn) NamedQuery(ctx context.Context, query string, arg any) (*sql.Rows, error) {
	// Convert named parameters to positional parameters for matching
	convertedQuery, args, err := mc.convertNamedQuery(query, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert named query: %w", err)
	}

	// Find matching query expectation
	exp, _ := mc.expectations.findMatchingExpectation("query", convertedQuery, args)
	if exp == nil {
		return nil, fmt.Errorf("unexpected named query call: %s with args %v", convertedQuery, args)
	}

	if exp.returnError != nil {
		return nil, exp.returnError
	}

	// Convert MockRows to actual rows
	if rows, ok := exp.returnRows.(*sqliteRows); ok {
		return mc.createRowsFromMockData(rows)
	}

	return nil, fmt.Errorf("invalid return rows data")
}

func (mc *mockConn) InsertOnDuplicate(ctx context.Context, table string, columns []string, rows [][]any, updateCols []string) (sql.Result, error) {
	// For mock purposes, treat like BulkInsert
	return mc.BulkInsert(ctx, table, columns, rows)
}

func (mc *mockConn) Close() error {
	return mc.DatabaseConn.Close()
}

// mockTx methods that intercept calls
func (mt *mockTx) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// Find matching exec expectation
	exp, _ := mt.expectations.findMatchingExpectation("exec", query, args)
	if exp == nil {
		return nil, fmt.Errorf("unexpected exec call in transaction: %s with args %v", query, args)
	}

	if exp.returnError != nil {
		return nil, exp.returnError
	}

	// Return the mock result
	if result, ok := exp.returnResult.(*sqliteResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid return result data")
}

// Note: DatabaseTx interface only has Exec method, not Query
// If Query is needed in transactions, it should be added to the interface

// NewPoolWithMock creates either a real Pool or mock Pool based on isMock flag.
func NewPoolWithMock(ctx context.Context, cfg Config, isMock bool) (DatabasePool, MockExpectations, error) {
	if isMock {
		return NewMockPool(ctx, cfg)
	}
	pool, err := NewPool(ctx, cfg)
	return pool, nil, err
}
