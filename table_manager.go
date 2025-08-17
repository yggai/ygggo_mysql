package ygggo_mysql

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Table management methods are separated from database management for clarity.

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

// GetCreateTableSQL returns the CREATE TABLE SQL that would be used for the given model.
func (m *DBManager) GetCreateTableSQL(model any) string {
	name, cols, ok := buildCreateTableSQL(model)
	if !ok || len(cols) == 0 {
		return ""
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", name, strings.Join(cols, ","))
}

// ShowCreateTable executes SHOW CREATE TABLE for the table inferred from the model
// and returns the DDL string if available.
func (m *DBManager) ShowCreateTable(model any) string {
	if m == nil || m.db == nil || model == nil {
		return ""
	}
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	name := toSnake(t.Name())
	ctx := context.Background()
	row := m.db.QueryRowContext(ctx, "SHOW CREATE TABLE `"+name+"`")
	var tbl, ddl string
	if err := row.Scan(&tbl, &ddl); err != nil {
		return ""
	}
	return ddl
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
	var constraints []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("ggm")
		colName := toSnake(f.Name)
		colType := mapGoTypeToSQL(f.Type)
		pk := false
		auto := false
		notnull := false
		unique := false
		idx := false
		uidx := false
		var defVal string
		if tag != "" {
			tokens := strings.Split(tag, ",")
			for _, raw := range tokens {
				kv := strings.TrimSpace(raw)
				if kv == "" {
					continue
				}
				low := strings.ToLower(kv)
				switch {
				case low == "pk" || low == "primary" || low == "primarykey":
					pk = true
				case low == "auto" || low == "auto_increment":
					auto = true
				case low == "notnull" || low == "not null":
					notnull = true
				case low == "unique":
					unique = true
				case low == "index":
					idx = true
				case low == "uniqueindex" || low == "unique_index" || low == "uniq":
					uidx = true
				case strings.HasPrefix(low, "name="):
					colName = strings.TrimSpace(kv[len("name="):])
				case strings.HasPrefix(low, "type="):
					colType = strings.TrimSpace(kv[len("type="):])
				case strings.HasPrefix(low, "default="):
					defVal = strings.TrimSpace(kv[len("default="):])
				default:
					// If no key=value and not a known flag, treat as column name
					if !strings.Contains(kv, "=") {
						colName = kv
					}
				}
			}
		}
		colDef := fmt.Sprintf("`%s` %s", colName, colType)
		if notnull {
			colDef += " NOT NULL"
		}
		if defVal != "" {
			colDef += " DEFAULT " + formatDefaultValue(defVal)
		}
		if auto {
			colDef += " AUTO_INCREMENT"
		}
		if pk {
			colDef += " PRIMARY KEY"
		}
		if unique && !pk {
			colDef += " UNIQUE"
		}
		columns = append(columns, colDef)
		// Table-level indexes
		if idx {
			constraints = append(constraints, fmt.Sprintf("INDEX (`%s`)", colName))
		}
		if uidx {
			constraints = append(constraints, fmt.Sprintf("UNIQUE KEY `uniq_%s` (`%s`)", colName, colName))
		}
	}
	if len(constraints) > 0 {
		columns = append(columns, constraints...)
	}
	return table, columns, true
}

// formatDefaultValue formats DEFAULT literal; wrap non-numeric/non-function values with quotes.
func formatDefaultValue(v string) string {
	lv := strings.ToLower(v)
	// common MySQL functions allowed unquoted
	switch lv {
	case "current_timestamp", "current_timestamp()", "now()", "null":
		return v
	}
	// numeric?
	if _, err := fmt.Sscan(v, new(float64)); err == nil {
		return v
	}
	// otherwise quote
	return "'" + strings.Trim(v, "'\"") + "'"
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
