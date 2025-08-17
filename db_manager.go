package ygggo_mysql

import (
	"context"
	"database/sql"
)

// DBManager provides simple database management operations.
// Note: methods follow the example signature (no error returns) and use background context.
// Errors are silently ignored to match the example semantics.
type DBManager struct {
	db *sql.DB
}

// GetDB returns a DBManager bound to the underlying *sql.DB.
func (p *Pool) GetDB() (*DBManager, error) {
	if p == nil || p.db == nil {
		return nil, sql.ErrConnDone
	}
	return &DBManager{db: p.db}, nil
}

// GetAllDatabase returns all database names.
func (m *DBManager) GetAllDatabase() []string {
	if m == nil || m.db == nil {
		return nil
	}
	ctx := context.Background()
	rows, err := m.db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return res
		}
		res = append(res, name)
	}
	return res
}

// AddDatabase creates a database if it doesn't exist.
func (m *DBManager) AddDatabase(name string) {
	if m == nil || m.db == nil || name == "" {
		return
	}
	ctx := context.Background()
	_, _ = m.db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+name+"`")
}

// DeleteDatabase drops a database if it exists.
func (m *DBManager) DeleteDatabase(name string) {
	if m == nil || m.db == nil || name == "" {
		return
	}
	ctx := context.Background()
	_, _ = m.db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+name+"`")
}
