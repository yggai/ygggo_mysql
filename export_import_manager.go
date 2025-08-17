package ygggo_mysql

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// FileFormat 文件格式枚举
type FileFormat string

const (
	FormatSQL  FileFormat = "sql"
	FormatCSV  FileFormat = "csv"
	FormatJSON FileFormat = "json"
)

// ExportOptions 导出选项
type ExportOptions struct {
	Format      FileFormat // 文件格式
	Output      io.Writer  // 输出流
	TableNames  []string   // 指定表名（可选）
	WhereClause string     // WHERE条件（可选）
	Args        []any      // WHERE条件参数（可选）
}

// ImportOptions 导入选项
type ImportOptions struct {
	Format        FileFormat // 文件格式
	Input         io.Reader  // 输入流
	TableNames    []string   // 指定表名（可选）
	TruncateFirst bool       // 导入前是否清空表
	IgnoreErrors  bool       // 是否忽略错误继续导入
}

// ExportImportManager 导入导出管理器接口
type ExportImportManager interface {
	// 导出功能
	ExportTable(ctx context.Context, tableName string, options ExportOptions) error
	ExportTables(ctx context.Context, tableNames []string, options ExportOptions) error
	Export(ctx context.Context, options ExportOptions) error

	// 导入功能
	ImportTable(ctx context.Context, tableName string, options ImportOptions) error
	ImportTables(ctx context.Context, tableNames []string, options ImportOptions) error
	Import(ctx context.Context, options ImportOptions) error
}

// TableSchema 表结构信息
type TableSchema struct {
	TableName string
	Columns   []ColumnInfo
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name         string
	Type         string
	IsPrimaryKey bool
	IsNullable   bool
	DefaultValue string
}

// exportImportManager 导入导出管理器实现
type exportImportManager struct {
	pool DatabasePool
}

// NewExportImportManager 创建导入导出管理器
func NewExportImportManager(pool DatabasePool) ExportImportManager {
	return &exportImportManager{
		pool: pool,
	}
}

// DataFormatter 数据格式化器接口
type DataFormatter interface {
	// 导出数据
	ExportTable(ctx context.Context, schema TableSchema, rows [][]any, writer io.Writer) error
	ExportTables(ctx context.Context, schemas []TableSchema, tablesData map[string][][]any, writer io.Writer) error

	// 导入数据
	ImportTable(ctx context.Context, reader io.Reader) (TableSchema, [][]any, error)
	ImportTables(ctx context.Context, reader io.Reader) ([]TableSchema, map[string][][]any, error)
}

// sqlFormatter SQL格式化器
type sqlFormatter struct{}

// csvFormatter CSV格式化器
type csvFormatter struct{}

// jsonFormatter JSON格式化器
type jsonFormatter struct{}

// NewDataFormatter 创建数据格式化器
func NewDataFormatter(format FileFormat) (DataFormatter, error) {
	switch format {
	case FormatSQL:
		return &sqlFormatter{}, nil
	case FormatCSV:
		return &csvFormatter{}, nil
	case FormatJSON:
		return &jsonFormatter{}, nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

// SQL格式化器实现
func (f *sqlFormatter) ExportTable(ctx context.Context, schema TableSchema, rows [][]any, writer io.Writer) error {
	// 生成CREATE TABLE语句
	createSQL := f.generateCreateTableSQL(schema)
	if _, err := writer.Write([]byte(createSQL + "\n\n")); err != nil {
		return err
	}

	// 生成INSERT语句
	if len(rows) > 0 {
		insertSQL := f.generateInsertSQL(schema, rows)
		if _, err := writer.Write([]byte(insertSQL + "\n")); err != nil {
			return err
		}
	}

	return nil
}

func (f *sqlFormatter) ExportTables(ctx context.Context, schemas []TableSchema, tablesData map[string][][]any, writer io.Writer) error {
	for _, schema := range schemas {
		if rows, exists := tablesData[schema.TableName]; exists {
			if err := f.ExportTable(ctx, schema, rows, writer); err != nil {
				return err
			}
			if _, err := writer.Write([]byte("\n")); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *sqlFormatter) ImportTable(ctx context.Context, reader io.Reader) (TableSchema, [][]any, error) {
	// SQL导入实现（简化版本）
	return TableSchema{}, nil, errors.New("SQL import not implemented yet")
}

func (f *sqlFormatter) ImportTables(ctx context.Context, reader io.Reader) ([]TableSchema, map[string][][]any, error) {
	// SQL导入实现（简化版本）
	return nil, nil, errors.New("SQL import not implemented yet")
}

func (f *sqlFormatter) generateCreateTableSQL(schema TableSchema) string {
	var columns []string
	for _, col := range schema.Columns {
		colDef := fmt.Sprintf("`%s` %s", col.Name, col.Type)
		if col.IsPrimaryKey {
			colDef += " PRIMARY KEY"
		}
		if !col.IsNullable {
			colDef += " NOT NULL"
		}
		if col.DefaultValue != "" {
			colDef += fmt.Sprintf(" DEFAULT %s", col.DefaultValue)
		}
		columns = append(columns, colDef)
	}

	return fmt.Sprintf("CREATE TABLE `%s` (\n  %s\n);",
		schema.TableName, strings.Join(columns, ",\n  "))
}

func (f *sqlFormatter) generateInsertSQL(schema TableSchema, rows [][]any) string {
	if len(rows) == 0 {
		return ""
	}

	// 构建列名
	var columnNames []string
	for _, col := range schema.Columns {
		columnNames = append(columnNames, fmt.Sprintf("`%s`", col.Name))
	}

	// 构建VALUES子句
	var valueStrings []string
	for _, row := range rows {
		var values []string
		for _, val := range row {
			if val == nil {
				values = append(values, "NULL")
			} else {
				switch v := val.(type) {
				case string:
					values = append(values, fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''")))
				case int, int64, float64:
					values = append(values, fmt.Sprintf("%v", v))
				default:
					values = append(values, fmt.Sprintf("'%v'", v))
				}
			}
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(values, ", ")))
	}

	return fmt.Sprintf("INSERT INTO `%s` (%s) VALUES\n%s;",
		schema.TableName,
		strings.Join(columnNames, ", "),
		strings.Join(valueStrings, ",\n"))
}

// CSV格式化器实现
func (f *csvFormatter) ExportTable(ctx context.Context, schema TableSchema, rows [][]any, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// 写入头部
	var headers []string
	for _, col := range schema.Columns {
		headers = append(headers, col.Name)
	}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	// 写入数据行
	for _, row := range rows {
		var record []string
		for _, val := range row {
			if val == nil {
				record = append(record, "")
			} else {
				record = append(record, fmt.Sprintf("%v", val))
			}
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func (f *csvFormatter) ExportTables(ctx context.Context, schemas []TableSchema, tablesData map[string][][]any, writer io.Writer) error {
	// CSV格式不支持多表导出，只导出第一个表
	if len(schemas) > 0 {
		schema := schemas[0]
		if rows, exists := tablesData[schema.TableName]; exists {
			return f.ExportTable(ctx, schema, rows, writer)
		}
	}
	return nil
}

func (f *csvFormatter) ImportTable(ctx context.Context, reader io.Reader) (TableSchema, [][]any, error) {
	csvReader := csv.NewReader(reader)

	// 读取头部
	headers, err := csvReader.Read()
	if err != nil {
		return TableSchema{}, nil, err
	}

	// 构建表结构（简化版本，所有字段都是VARCHAR类型）
	var columns []ColumnInfo
	for _, header := range headers {
		columns = append(columns, ColumnInfo{
			Name:       header,
			Type:       "VARCHAR(255)",
			IsNullable: true,
		})
	}

	schema := TableSchema{
		TableName: "imported_table", // 默认表名
		Columns:   columns,
	}

	// 读取数据行
	var rows [][]any
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return schema, rows, err
		}

		var row []any
		for _, field := range record {
			if field == "" {
				row = append(row, nil)
			} else {
				row = append(row, field)
			}
		}
		rows = append(rows, row)
	}

	return schema, rows, nil
}

func (f *csvFormatter) ImportTables(ctx context.Context, reader io.Reader) ([]TableSchema, map[string][][]any, error) {
	// CSV格式只支持单表导入
	schema, rows, err := f.ImportTable(ctx, reader)
	if err != nil {
		return nil, nil, err
	}

	schemas := []TableSchema{schema}
	tablesData := map[string][][]any{
		schema.TableName: rows,
	}

	return schemas, tablesData, nil
}

// JSON格式化器实现
func (f *jsonFormatter) ExportTable(ctx context.Context, schema TableSchema, rows [][]any, writer io.Writer) error {
	// 构建JSON数据结构
	var records []map[string]any
	for _, row := range rows {
		record := make(map[string]any)
		for i, col := range schema.Columns {
			if i < len(row) {
				record[col.Name] = row[i]
			}
		}
		records = append(records, record)
	}

	// 创建完整的JSON结构
	data := map[string]any{
		"table":   schema.TableName,
		"schema":  schema,
		"records": records,
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (f *jsonFormatter) ExportTables(ctx context.Context, schemas []TableSchema, tablesData map[string][][]any, writer io.Writer) error {
	// 构建多表JSON数据结构
	tables := make(map[string]any)
	for _, schema := range schemas {
		if rows, exists := tablesData[schema.TableName]; exists {
			var records []map[string]any
			for _, row := range rows {
				record := make(map[string]any)
				for i, col := range schema.Columns {
					if i < len(row) {
						record[col.Name] = row[i]
					}
				}
				records = append(records, record)
			}

			tables[schema.TableName] = map[string]any{
				"schema":  schema,
				"records": records,
			}
		}
	}

	data := map[string]any{
		"database": "exported_database",
		"tables":   tables,
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (f *jsonFormatter) ImportTable(ctx context.Context, reader io.Reader) (TableSchema, [][]any, error) {
	var data map[string]any
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return TableSchema{}, nil, err
	}

	// 解析表结构
	var schema TableSchema
	if schemaData, ok := data["schema"].(map[string]any); ok {
		if tableName, ok := schemaData["TableName"].(string); ok {
			schema.TableName = tableName
		}
		if columns, ok := schemaData["Columns"].([]any); ok {
			for _, col := range columns {
				if colMap, ok := col.(map[string]any); ok {
					column := ColumnInfo{}
					if name, ok := colMap["Name"].(string); ok {
						column.Name = name
					}
					if colType, ok := colMap["Type"].(string); ok {
						column.Type = colType
					}
					if isPK, ok := colMap["IsPrimaryKey"].(bool); ok {
						column.IsPrimaryKey = isPK
					}
					if isNull, ok := colMap["IsNullable"].(bool); ok {
						column.IsNullable = isNull
					}
					schema.Columns = append(schema.Columns, column)
				}
			}
		}
	}

	// 解析数据行
	var rows [][]any
	if records, ok := data["records"].([]any); ok {
		for _, record := range records {
			if recordMap, ok := record.(map[string]any); ok {
				var row []any
				for _, col := range schema.Columns {
					row = append(row, recordMap[col.Name])
				}
				rows = append(rows, row)
			}
		}
	}

	return schema, rows, nil
}

func (f *jsonFormatter) ImportTables(ctx context.Context, reader io.Reader) ([]TableSchema, map[string][][]any, error) {
	var data map[string]any
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return nil, nil, err
	}

	var schemas []TableSchema
	tablesData := make(map[string][][]any)

	if tables, ok := data["tables"].(map[string]any); ok {
		for tableName, tableData := range tables {
			if tableMap, ok := tableData.(map[string]any); ok {
				// 解析表结构
				var schema TableSchema
				schema.TableName = tableName

				if schemaData, ok := tableMap["schema"].(map[string]any); ok {
					if columns, ok := schemaData["Columns"].([]any); ok {
						for _, col := range columns {
							if colMap, ok := col.(map[string]any); ok {
								column := ColumnInfo{}
								if name, ok := colMap["Name"].(string); ok {
									column.Name = name
								}
								if colType, ok := colMap["Type"].(string); ok {
									column.Type = colType
								}
								schema.Columns = append(schema.Columns, column)
							}
						}
					}
				}

				// 解析数据行
				var rows [][]any
				if records, ok := tableMap["records"].([]any); ok {
					for _, record := range records {
						if recordMap, ok := record.(map[string]any); ok {
							var row []any
							for _, col := range schema.Columns {
								row = append(row, recordMap[col.Name])
							}
							rows = append(rows, row)
						}
					}
				}

				schemas = append(schemas, schema)
				tablesData[tableName] = rows
			}
		}
	}

	return schemas, tablesData, nil
}

// ExportTable 导出指定表的数据
func (m *exportImportManager) ExportTable(ctx context.Context, tableName string, options ExportOptions) error {
	if tableName == "" {
		return ErrEmptyTableName
	}

	// 获取表结构
	schema, err := m.getTableSchema(ctx, tableName)
	if err != nil {
		return err
	}

	// 获取表数据
	rows, err := m.getTableData(ctx, tableName, options.WhereClause, options.Args...)
	if err != nil {
		return err
	}

	// 创建格式化器
	formatter, err := NewDataFormatter(options.Format)
	if err != nil {
		return err
	}

	// 导出数据
	return formatter.ExportTable(ctx, schema, rows, options.Output)
}

// ExportTables 导出多个表的数据
func (m *exportImportManager) ExportTables(ctx context.Context, tableNames []string, options ExportOptions) error {
	if len(tableNames) == 0 {
		return errors.New("table names cannot be empty")
	}

	var schemas []TableSchema
	tablesData := make(map[string][][]any)

	// 获取所有表的结构和数据
	for _, tableName := range tableNames {
		schema, err := m.getTableSchema(ctx, tableName)
		if err != nil {
			return err
		}
		schemas = append(schemas, schema)

		rows, err := m.getTableData(ctx, tableName, options.WhereClause, options.Args...)
		if err != nil {
			return err
		}
		tablesData[tableName] = rows
	}

	// 创建格式化器
	formatter, err := NewDataFormatter(options.Format)
	if err != nil {
		return err
	}

	// 导出数据
	return formatter.ExportTables(ctx, schemas, tablesData, options.Output)
}

// Export 导出整个数据库的数据
func (m *exportImportManager) Export(ctx context.Context, options ExportOptions) error {
	// 获取所有表名
	tableNames, err := m.getAllTableNames(ctx)
	if err != nil {
		return err
	}

	// 如果指定了表名，则只导出指定的表
	if len(options.TableNames) > 0 {
		tableNames = options.TableNames
	}

	// 调用ExportTables
	return m.ExportTables(ctx, tableNames, options)
}

// getTableSchema 获取表结构
func (m *exportImportManager) getTableSchema(ctx context.Context, tableName string) (TableSchema, error) {
	var schema TableSchema
	schema.TableName = tableName

	err := m.pool.WithConn(ctx, func(c DatabaseConn) error {
		// 查询表结构
		rows, err := c.Query(ctx, `
			SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_KEY
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`, tableName)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var columnName, dataType, isNullable, columnKey string
			var columnDefault *string // 使用指针处理NULL值
			err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault, &columnKey)
			if err != nil {
				return err
			}

			defaultValue := ""
			if columnDefault != nil {
				defaultValue = *columnDefault
			}

			column := ColumnInfo{
				Name:         columnName,
				Type:         dataType,
				IsPrimaryKey: columnKey == "PRI",
				IsNullable:   isNullable == "YES",
				DefaultValue: defaultValue,
			}
			schema.Columns = append(schema.Columns, column)
		}

		return rows.Err()
	})

	if err != nil {
		return schema, err
	}

	if len(schema.Columns) == 0 {
		return schema, ErrTableNotFound
	}

	return schema, nil
}

// getTableData 获取表数据
func (m *exportImportManager) getTableData(ctx context.Context, tableName string, whereClause string, args ...any) ([][]any, error) {
	var rows [][]any

	err := m.pool.WithConn(ctx, func(c DatabaseConn) error {
		// 构建查询SQL
		sql := fmt.Sprintf("SELECT * FROM `%s`", tableName)
		if whereClause != "" {
			sql += " WHERE " + whereClause
		}

		// 执行查询
		queryRows, err := c.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer queryRows.Close()

		// 获取列信息
		columns, err := queryRows.Columns()
		if err != nil {
			return err
		}

		// 读取数据行
		for queryRows.Next() {
			// 创建扫描目标
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			// 扫描行数据
			if err := queryRows.Scan(valuePtrs...); err != nil {
				return err
			}

			// 转换数据类型
			row := make([]any, len(values))
			for i, val := range values {
				if val == nil {
					row[i] = nil
				} else {
					switch v := val.(type) {
					case []byte:
						row[i] = string(v)
					default:
						row[i] = v
					}
				}
			}

			rows = append(rows, row)
		}

		return queryRows.Err()
	})

	return rows, err
}

// getAllTableNames 获取所有表名
func (m *exportImportManager) getAllTableNames(ctx context.Context) ([]string, error) {
	var tableNames []string

	err := m.pool.WithConn(ctx, func(c DatabaseConn) error {
		rows, err := c.Query(ctx, `
			SELECT TABLE_NAME
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
			ORDER BY TABLE_NAME`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return err
			}
			tableNames = append(tableNames, tableName)
		}

		return rows.Err()
	})

	return tableNames, err
}

// ImportTable 导入指定表的数据
func (m *exportImportManager) ImportTable(ctx context.Context, tableName string, options ImportOptions) error {
	if tableName == "" {
		return ErrEmptyTableName
	}

	// 创建格式化器
	formatter, err := NewDataFormatter(options.Format)
	if err != nil {
		return err
	}

	// 解析导入数据
	schema, rows, err := formatter.ImportTable(ctx, options.Input)
	if err != nil {
		return err
	}

	// 如果指定了表名，使用指定的表名
	if tableName != "" {
		schema.TableName = tableName
	}

	// 执行导入
	return m.importTableData(ctx, schema, rows, options)
}

// ImportTables 导入多个表的数据
func (m *exportImportManager) ImportTables(ctx context.Context, tableNames []string, options ImportOptions) error {
	// 创建格式化器
	formatter, err := NewDataFormatter(options.Format)
	if err != nil {
		return err
	}

	// 解析导入数据
	schemas, tablesData, err := formatter.ImportTables(ctx, options.Input)
	if err != nil {
		return err
	}

	// 如果指定了表名，只导入指定的表
	if len(tableNames) > 0 {
		filteredSchemas := make([]TableSchema, 0)
		filteredData := make(map[string][][]any)

		for _, tableName := range tableNames {
			for _, schema := range schemas {
				if schema.TableName == tableName {
					filteredSchemas = append(filteredSchemas, schema)
					if data, exists := tablesData[tableName]; exists {
						filteredData[tableName] = data
					}
					break
				}
			}
		}

		schemas = filteredSchemas
		tablesData = filteredData
	}

	// 执行导入
	for _, schema := range schemas {
		if rows, exists := tablesData[schema.TableName]; exists {
			if err := m.importTableData(ctx, schema, rows, options); err != nil {
				if !options.IgnoreErrors {
					return err
				}
				// 如果忽略错误，记录日志但继续
				fmt.Printf("Warning: Failed to import table %s: %v\n", schema.TableName, err)
			}
		}
	}

	return nil
}

// Import 导入整个数据库的数据
func (m *exportImportManager) Import(ctx context.Context, options ImportOptions) error {
	// 创建格式化器
	formatter, err := NewDataFormatter(options.Format)
	if err != nil {
		return err
	}

	// 解析导入数据
	schemas, tablesData, err := formatter.ImportTables(ctx, options.Input)
	if err != nil {
		return err
	}

	// 如果指定了表名，只导入指定的表
	if len(options.TableNames) > 0 {
		return m.ImportTables(ctx, options.TableNames, options)
	}

	// 导入所有表
	for _, schema := range schemas {
		if rows, exists := tablesData[schema.TableName]; exists {
			if err := m.importTableData(ctx, schema, rows, options); err != nil {
				if !options.IgnoreErrors {
					return err
				}
				// 如果忽略错误，记录日志但继续
				fmt.Printf("Warning: Failed to import table %s: %v\n", schema.TableName, err)
			}
		}
	}

	return nil
}

// importTableData 导入表数据的辅助方法
func (m *exportImportManager) importTableData(ctx context.Context, schema TableSchema, rows [][]any, options ImportOptions) error {
	return m.pool.WithConn(ctx, func(c DatabaseConn) error {
		// 如果需要，先清空表
		if options.TruncateFirst {
			_, err := c.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE `%s`", schema.TableName))
			if err != nil {
				// 如果TRUNCATE失败，尝试DELETE
				_, err = c.Exec(ctx, fmt.Sprintf("DELETE FROM `%s`", schema.TableName))
				if err != nil {
					return fmt.Errorf("failed to clear table %s: %v", schema.TableName, err)
				}
			}
		}

		// 如果没有数据，直接返回
		if len(rows) == 0 {
			return nil
		}

		// 构建INSERT语句
		var columnNames []string
		for _, col := range schema.Columns {
			columnNames = append(columnNames, fmt.Sprintf("`%s`", col.Name))
		}

		// 批量插入数据
		batchSize := 1000 // 每批插入1000条记录
		for i := 0; i < len(rows); i += batchSize {
			end := i + batchSize
			if end > len(rows) {
				end = len(rows)
			}

			batch := rows[i:end]
			if err := m.insertBatch(ctx, c, schema.TableName, columnNames, batch); err != nil {
				return fmt.Errorf("failed to insert batch starting at row %d: %v", i, err)
			}
		}

		return nil
	})
}

// insertBatch 批量插入数据
func (m *exportImportManager) insertBatch(ctx context.Context, c DatabaseConn, tableName string, columnNames []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}

	// 构建VALUES子句
	var valueStrings []string
	var args []any

	for _, row := range rows {
		placeholders := make([]string, len(row))
		for i, val := range row {
			placeholders[i] = "?"
			args = append(args, val)
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
	}

	// 构建完整的INSERT语句
	sql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(valueStrings, ", "))

	// 执行插入
	_, err := c.Exec(ctx, sql, args...)
	return err
}

// 错误定义
var (
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrTableNotFound     = errors.New("table not found")
	ErrInvalidData       = errors.New("invalid data format")
	ErrEmptyTableName    = errors.New("table name cannot be empty")
)
