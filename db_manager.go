package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
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

// GetAllTable returns all table names in the current database.
func (m *DBManager) GetAllTable() []string {
	if m == nil || m.db == nil {
		return nil
	}
	ctx := context.Background()
	rows, err := m.db.QueryContext(ctx, "SHOW TABLES")
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

// AddTable creates a table according to struct definition tagged by ggm.
// Supported tags: `ggm:"column"` on exported fields; basic Go types are mapped to generic SQL types.
func (m *DBManager) AddTable(model any) {
	if m == nil || m.db == nil || model == nil {
		return
	}
	name, cols, ok := buildCreateTableSQL(model)
	if !ok {
		return
	}
	ctx := context.Background()
	_, _ = m.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", name, strings.Join(cols, ",")))
}

// DeleteTable drops a table inferred from the struct name.
func (m *DBManager) DeleteTable(model any) {
	if m == nil || m.db == nil || model == nil {
		return
	}
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	name := toSnake(t.Name())
	ctx := context.Background()
	_, _ = m.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", name))
}

// buildCreateTableSQL builds table name and column definitions from struct tags and types.
func buildCreateTableSQL(model any) (table string, columns []string, ok bool) {
	t := reflect.TypeOf(model)
	if t == nil {
		return "", nil, false
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", nil, false
	}
	table = toSnake(t.Name())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		col := f.Tag.Get("ggm")
		if col == "" {
			col = toSnake(f.Name)
		}
		sqlType := mapGoTypeToSQL(f.Type)
		columns = append(columns, fmt.Sprintf("`%s` %s", col, sqlType))
	}
	return table, columns, true
}

// mapGoTypeToSQL maps basic Go types to MySQL column types.
func mapGoTypeToSQL(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INT"
	case reflect.Int64, reflect.Uint64:
		return "BIGINT"
	case reflect.String:
		return "VARCHAR(255)"
	case reflect.Bool:
		return "TINYINT(1)"
	case reflect.Float32, reflect.Float64:
		return "DOUBLE"
	case reflect.Struct:
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return "DATETIME"
		}
	}
	return "TEXT"
}

// toSnake converts CamelCase to snake_case
func toSnake(s string) string {
	var out []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		out = append(out, r)
	}
	return string(out)
}

// DeleteDatabase drops a database if it exists.
func (m *DBManager) DeleteDatabase(name string) {
	if m == nil || m.db == nil || name == "" {
		return
	}
	ctx := context.Background()
	_, _ = m.db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+name+"`")
}
