package ygggo_mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// TableDataManager 表格数据管理器接口
type TableDataManager interface {
	// 基础SQL操作方法
	PrepareSql(ctx context.Context, sql string) (*sql.Stmt, error)
	Execute(ctx context.Context, sql string, args ...any) (interface{}, error)
	Query(ctx context.Context, sql string, args ...any) (interface{}, error)

	// 增加操作
	Add(ctx context.Context, entity any) error
	AddMany(ctx context.Context, entities any) error

	// 删除操作
	Delete(ctx context.Context, id any) error
	DeleteIn(ctx context.Context, ids any) error
	DeleteBy(ctx context.Context, condition string, args ...any) error

	// 更新操作
	Update(ctx context.Context, entity any) error
	UpdateIn(ctx context.Context, ids any, updates map[string]any) error
	UpdateBy(ctx context.Context, condition string, updates map[string]any, args ...any) error

	// 查询操作
	Get(ctx context.Context, id any, result any) error
	GetBy(ctx context.Context, condition string, result any, args ...any) error
	GetIn(ctx context.Context, ids any, result any) error
	GetPage(ctx context.Context, page, pageSize int, result any, condition string, args ...any) error
	GetAll(ctx context.Context, result any, condition string, args ...any) error
}

// TableInfo 表信息
type TableInfo struct {
	TableName  string
	PrimaryKey string
	Fields     []FieldInfo
	FieldMap   map[string]FieldInfo
}

// FieldInfo 字段信息
type FieldInfo struct {
	FieldName    string // 结构体字段名
	ColumnName   string // 数据库列名
	ColumnType   string // 数据库列类型
	IsPrimaryKey bool   // 是否主键
	IsAutoIncr   bool   // 是否自增
	IsNullable   bool   // 是否可空
	DefaultValue string // 默认值
	Tag          string // 完整标签
}

// tableDataManager 表格数据管理器实现
type tableDataManager struct {
	pool       DatabasePool
	tableInfo  *TableInfo
	entityType reflect.Type
}

// NewTableDataManager 创建表格数据管理器
func NewTableDataManager(pool DatabasePool, entity any) (TableDataManager, error) {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	if entityType.Kind() != reflect.Struct {
		return nil, ErrInvalidEntityType
	}

	tableInfo, err := parseTableInfo(entityType, entity)
	if err != nil {
		return nil, err
	}

	return &tableDataManager{
		pool:       pool,
		tableInfo:  tableInfo,
		entityType: entityType,
	}, nil
}

// parseTableInfo 解析表信息
func parseTableInfo(entityType reflect.Type, entity any) (*TableInfo, error) {
	tableInfo := &TableInfo{
		Fields:   make([]FieldInfo, 0),
		FieldMap: make(map[string]FieldInfo),
	}

	// 获取表名
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if method := entityValue.MethodByName("TableName"); method.IsValid() {
		result := method.Call(nil)
		if len(result) > 0 {
			if tableName, ok := result[0].Interface().(string); ok && tableName != "" {
				tableInfo.TableName = tableName
			}
		}
	}

	if tableInfo.TableName == "" {
		// 如果没有TableName方法，使用结构体名称的小写形式
		tableInfo.TableName = strings.ToLower(entityType.Name())
	}

	// 解析字段
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)

		// 跳过非导出字段
		if !field.IsExported() {
			continue
		}

		fieldInfo := parseFieldInfo(field)
		if fieldInfo.ColumnName != "" {
			tableInfo.Fields = append(tableInfo.Fields, fieldInfo)
			tableInfo.FieldMap[fieldInfo.FieldName] = fieldInfo

			// 设置主键
			if fieldInfo.IsPrimaryKey {
				tableInfo.PrimaryKey = fieldInfo.ColumnName
			}
		}
	}

	if tableInfo.PrimaryKey == "" {
		return nil, ErrPrimaryKeyEmpty
	}

	return tableInfo, nil
}

// parseFieldInfo 解析字段信息
func parseFieldInfo(field reflect.StructField) FieldInfo {
	fieldInfo := FieldInfo{
		FieldName: field.Name,
		Tag:       string(field.Tag),
	}

	// 解析ggm标签
	tag := field.Tag.Get("ggm")
	if tag == "" {
		return fieldInfo
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		fieldInfo.ColumnName = strings.TrimSpace(parts[0])
	}

	// 解析标签选项
	for i := 1; i < len(parts); i++ {
		option := strings.TrimSpace(parts[i])
		switch {
		case option == "primary_key":
			fieldInfo.IsPrimaryKey = true
		case option == "auto_increment":
			fieldInfo.IsAutoIncr = true
		case option == "not_null":
			fieldInfo.IsNullable = false
		case option == "unique":
			// 唯一约束标记
		case strings.HasPrefix(option, "default:"):
			fieldInfo.DefaultValue = strings.TrimPrefix(option, "default:")
		}
	}

	return fieldInfo
}

// PrepareSql 准备SQL语句 (注意：当前DatabaseConn接口不支持Prepare，此方法暂不实现)
func (m *tableDataManager) PrepareSql(ctx context.Context, sql string) (*sql.Stmt, error) {
	return nil, errors.New("PrepareSql not supported by current DatabaseConn interface")
}

// Execute 执行SQL语句
func (m *tableDataManager) Execute(ctx context.Context, sql string, args ...any) (interface{}, error) {
	var result interface{}
	var err error

	err = m.pool.WithConn(ctx, func(c DatabaseConn) error {
		result, err = c.Exec(ctx, sql, args...)
		return err
	})

	return result, err
}

// Query 查询SQL语句
func (m *tableDataManager) Query(ctx context.Context, sql string, args ...any) (interface{}, error) {
	var rows interface{}
	var err error

	err = m.pool.WithConn(ctx, func(c DatabaseConn) error {
		rows, err = c.Query(ctx, sql, args...)
		return err
	})

	return rows, err
}

// Add 新增单个实体
func (m *tableDataManager) Add(ctx context.Context, entity any) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if entityValue.Kind() != reflect.Struct {
		return ErrInvalidEntityType
	}

	// 构建INSERT SQL
	columns := make([]string, 0)
	placeholders := make([]string, 0)
	values := make([]any, 0)

	for _, field := range m.tableInfo.Fields {
		// 跳过自增字段
		if field.IsAutoIncr {
			continue
		}

		fieldValue := entityValue.FieldByName(field.FieldName)
		if !fieldValue.IsValid() {
			continue
		}

		columns = append(columns, field.ColumnName)
		placeholders = append(placeholders, "?")
		values = append(values, fieldValue.Interface())
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		m.tableInfo.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// 直接使用连接执行，以便获取LastInsertId
	err := m.pool.WithConn(ctx, func(c DatabaseConn) error {
		result, err := c.Exec(ctx, sql, values...)
		if err != nil {
			return err
		}

		// 如果有自增主键，设置ID值
		if m.tableInfo.PrimaryKey != "" {
			for _, field := range m.tableInfo.Fields {
				if field.IsPrimaryKey && field.IsAutoIncr {
					if lastID, err := result.LastInsertId(); err == nil {
						fieldValue := entityValue.FieldByName(field.FieldName)
						if fieldValue.IsValid() && fieldValue.CanSet() {
							fieldValue.SetInt(lastID)
						}
					}
					break
				}
			}
		}
		return nil
	})

	return err
}

// AddMany 批量新增实体
func (m *tableDataManager) AddMany(ctx context.Context, entities any) error {
	entitiesValue := reflect.ValueOf(entities)
	if entitiesValue.Kind() == reflect.Ptr {
		entitiesValue = entitiesValue.Elem()
	}

	if entitiesValue.Kind() != reflect.Slice {
		return errors.New("entities must be a slice")
	}

	if entitiesValue.Len() == 0 {
		return nil
	}

	// 获取非自增字段
	columns := make([]string, 0)
	for _, field := range m.tableInfo.Fields {
		if !field.IsAutoIncr {
			columns = append(columns, field.ColumnName)
		}
	}

	// 构建批量插入的数据
	rows := make([][]any, 0, entitiesValue.Len())
	for i := 0; i < entitiesValue.Len(); i++ {
		entityValue := entitiesValue.Index(i)
		if entityValue.Kind() == reflect.Ptr {
			entityValue = entityValue.Elem()
		}

		row := make([]any, 0, len(columns))
		for _, field := range m.tableInfo.Fields {
			if field.IsAutoIncr {
				continue
			}

			fieldValue := entityValue.FieldByName(field.FieldName)
			if fieldValue.IsValid() {
				row = append(row, fieldValue.Interface())
			} else {
				row = append(row, nil)
			}
		}
		rows = append(rows, row)
	}

	// 使用BulkInsert方法
	return m.pool.WithConn(ctx, func(c DatabaseConn) error {
		_, err := c.BulkInsert(ctx, m.tableInfo.TableName, columns, rows)
		return err
	})
}

// Delete 根据ID删除实体
func (m *tableDataManager) Delete(ctx context.Context, id any) error {
	if m.tableInfo.PrimaryKey == "" {
		return ErrPrimaryKeyEmpty
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", m.tableInfo.TableName, m.tableInfo.PrimaryKey)
	_, err := m.Execute(ctx, sql, id)
	return err
}

// DeleteIn 根据ID列表删除实体
func (m *tableDataManager) DeleteIn(ctx context.Context, ids any) error {
	if m.tableInfo.PrimaryKey == "" {
		return ErrPrimaryKeyEmpty
	}

	idsValue := reflect.ValueOf(ids)
	if idsValue.Kind() == reflect.Ptr {
		idsValue = idsValue.Elem()
	}

	if idsValue.Kind() != reflect.Slice {
		return errors.New("ids must be a slice")
	}

	if idsValue.Len() == 0 {
		return nil
	}

	// 构建IN子句
	placeholders := make([]string, idsValue.Len())
	values := make([]any, idsValue.Len())
	for i := 0; i < idsValue.Len(); i++ {
		placeholders[i] = "?"
		values[i] = idsValue.Index(i).Interface()
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)",
		m.tableInfo.TableName,
		m.tableInfo.PrimaryKey,
		strings.Join(placeholders, ", "))

	_, err := m.Execute(ctx, sql, values...)
	return err
}

// DeleteBy 根据条件删除实体
func (m *tableDataManager) DeleteBy(ctx context.Context, condition string, args ...any) error {
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", m.tableInfo.TableName, condition)
	_, err := m.Execute(ctx, sql, args...)
	return err
}

// Update 更新实体
func (m *tableDataManager) Update(ctx context.Context, entity any) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if entityValue.Kind() != reflect.Struct {
		return ErrInvalidEntityType
	}

	if m.tableInfo.PrimaryKey == "" {
		return ErrPrimaryKeyEmpty
	}

	// 构建UPDATE SQL
	setParts := make([]string, 0)
	values := make([]any, 0)
	var primaryKeyValue any

	for _, field := range m.tableInfo.Fields {
		fieldValue := entityValue.FieldByName(field.FieldName)
		if !fieldValue.IsValid() {
			continue
		}

		if field.IsPrimaryKey {
			primaryKeyValue = fieldValue.Interface()
		} else {
			setParts = append(setParts, field.ColumnName+" = ?")
			values = append(values, fieldValue.Interface())
		}
	}

	if primaryKeyValue == nil {
		return errors.New("primary key value not found")
	}

	values = append(values, primaryKeyValue)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		m.tableInfo.TableName,
		strings.Join(setParts, ", "),
		m.tableInfo.PrimaryKey)

	_, err := m.Execute(ctx, sql, values...)
	return err
}

// UpdateIn 根据ID列表更新实体
func (m *tableDataManager) UpdateIn(ctx context.Context, ids any, updates map[string]any) error {
	if m.tableInfo.PrimaryKey == "" {
		return ErrPrimaryKeyEmpty
	}

	idsValue := reflect.ValueOf(ids)
	if idsValue.Kind() == reflect.Ptr {
		idsValue = idsValue.Elem()
	}

	if idsValue.Kind() != reflect.Slice {
		return errors.New("ids must be a slice")
	}

	if idsValue.Len() == 0 {
		return nil
	}

	// 构建SET子句
	setParts := make([]string, 0, len(updates))
	values := make([]any, 0, len(updates)+idsValue.Len())

	for column, value := range updates {
		setParts = append(setParts, column+" = ?")
		values = append(values, value)
	}

	// 构建IN子句
	placeholders := make([]string, idsValue.Len())
	for i := 0; i < idsValue.Len(); i++ {
		placeholders[i] = "?"
		values = append(values, idsValue.Index(i).Interface())
	}

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s IN (%s)",
		m.tableInfo.TableName,
		strings.Join(setParts, ", "),
		m.tableInfo.PrimaryKey,
		strings.Join(placeholders, ", "))

	_, err := m.Execute(ctx, sql, values...)
	return err
}

// UpdateBy 根据条件更新实体
func (m *tableDataManager) UpdateBy(ctx context.Context, condition string, updates map[string]any, args ...any) error {
	// 构建SET子句
	setParts := make([]string, 0, len(updates))
	values := make([]any, 0, len(updates)+len(args))

	for column, value := range updates {
		setParts = append(setParts, column+" = ?")
		values = append(values, value)
	}

	// 添加条件参数
	values = append(values, args...)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		m.tableInfo.TableName,
		strings.Join(setParts, ", "),
		condition)

	_, err := m.Execute(ctx, sql, values...)
	return err
}

// Get 根据ID查询实体
func (m *tableDataManager) Get(ctx context.Context, id any, result any) error {
	if m.tableInfo.PrimaryKey == "" {
		return ErrPrimaryKeyEmpty
	}

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", m.tableInfo.TableName, m.tableInfo.PrimaryKey)
	return m.queryOne(ctx, sql, result, id)
}

// GetBy 根据条件查询实体
func (m *tableDataManager) GetBy(ctx context.Context, condition string, result any, args ...any) error {
	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s", m.tableInfo.TableName, condition)
	return m.queryOne(ctx, sql, result, args...)
}

// GetIn 根据ID列表查询实体
func (m *tableDataManager) GetIn(ctx context.Context, ids any, result any) error {
	if m.tableInfo.PrimaryKey == "" {
		return ErrPrimaryKeyEmpty
	}

	idsValue := reflect.ValueOf(ids)
	if idsValue.Kind() == reflect.Ptr {
		idsValue = idsValue.Elem()
	}

	if idsValue.Kind() != reflect.Slice {
		return errors.New("ids must be a slice")
	}

	if idsValue.Len() == 0 {
		return nil
	}

	// 构建IN子句
	placeholders := make([]string, idsValue.Len())
	values := make([]any, idsValue.Len())
	for i := 0; i < idsValue.Len(); i++ {
		placeholders[i] = "?"
		values[i] = idsValue.Index(i).Interface()
	}

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)",
		m.tableInfo.TableName,
		m.tableInfo.PrimaryKey,
		strings.Join(placeholders, ", "))

	return m.queryMany(ctx, sql, result, values...)
}

// GetPage 分页查询实体
func (m *tableDataManager) GetPage(ctx context.Context, page, pageSize int, result any, condition string, args ...any) error {
	if page < 1 || pageSize < 1 {
		return ErrInvalidPageParams
	}

	offset := (page - 1) * pageSize

	var sql string
	if condition != "" {
		sql = fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT %d OFFSET %d",
			m.tableInfo.TableName, condition, pageSize, offset)
	} else {
		sql = fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",
			m.tableInfo.TableName, pageSize, offset)
	}

	return m.queryMany(ctx, sql, result, args...)
}

// GetAll 查询所有实体
func (m *tableDataManager) GetAll(ctx context.Context, result any, condition string, args ...any) error {
	var sql string
	if condition != "" {
		sql = fmt.Sprintf("SELECT * FROM %s WHERE %s", m.tableInfo.TableName, condition)
	} else {
		sql = fmt.Sprintf("SELECT * FROM %s", m.tableInfo.TableName)
	}

	return m.queryMany(ctx, sql, result, args...)
}

// queryOne 查询单个实体的辅助方法
func (m *tableDataManager) queryOne(ctx context.Context, sql string, result any, args ...any) error {
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	resultElem := resultValue.Elem()
	if resultElem.Kind() != reflect.Struct {
		return errors.New("result must be a pointer to struct")
	}

	return m.pool.WithConn(ctx, func(c DatabaseConn) error {
		row := c.QueryRow(ctx, sql, args...)
		return m.scanRow(row, result)
	})
}

// queryMany 查询多个实体的辅助方法
func (m *tableDataManager) queryMany(ctx context.Context, sql string, result any, args ...any) error {
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	resultElem := resultValue.Elem()
	if resultElem.Kind() != reflect.Slice {
		return errors.New("result must be a pointer to slice")
	}

	sliceType := resultElem.Type()
	elemType := sliceType.Elem()

	// 如果slice元素是指针类型，获取实际类型
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	return m.pool.WithConn(ctx, func(c DatabaseConn) error {
		rows, err := c.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		// 创建新的slice
		newSlice := reflect.MakeSlice(sliceType, 0, 0)

		for rows.Next() {
			// 创建新的实体实例
			var entity reflect.Value
			if sliceType.Elem().Kind() == reflect.Ptr {
				entity = reflect.New(elemType)
			} else {
				entity = reflect.New(elemType)
			}

			// 扫描行数据
			if err := m.scanRows(rows, entity.Interface()); err != nil {
				return err
			}

			// 添加到slice
			if sliceType.Elem().Kind() == reflect.Ptr {
				newSlice = reflect.Append(newSlice, entity)
			} else {
				newSlice = reflect.Append(newSlice, entity.Elem())
			}
		}

		// 设置结果
		resultElem.Set(newSlice)
		return rows.Err()
	})
}

// scanRow 扫描单行数据到结构体
func (m *tableDataManager) scanRow(row *sql.Row, result any) error {
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	resultElem := resultValue.Elem()
	if resultElem.Kind() != reflect.Struct {
		return errors.New("result must be a pointer to struct")
	}

	// 准备扫描目标
	scanTargets := make([]any, len(m.tableInfo.Fields))
	for i, field := range m.tableInfo.Fields {
		fieldValue := resultElem.FieldByName(field.FieldName)
		if fieldValue.IsValid() && fieldValue.CanSet() {
			scanTargets[i] = fieldValue.Addr().Interface()
		} else {
			// 如果字段不存在或不能设置，使用临时变量
			var temp any
			scanTargets[i] = &temp
		}
	}

	return row.Scan(scanTargets...)
}

// scanRows 扫描多行数据到结构体
func (m *tableDataManager) scanRows(rows *sql.Rows, result any) error {
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	resultElem := resultValue.Elem()
	if resultElem.Kind() != reflect.Struct {
		return errors.New("result must be a pointer to struct")
	}

	// 准备扫描目标
	scanTargets := make([]any, len(m.tableInfo.Fields))
	for i, field := range m.tableInfo.Fields {
		fieldValue := resultElem.FieldByName(field.FieldName)
		if fieldValue.IsValid() && fieldValue.CanSet() {
			scanTargets[i] = fieldValue.Addr().Interface()
		} else {
			// 如果字段不存在或不能设置，使用临时变量
			var temp any
			scanTargets[i] = &temp
		}
	}

	return rows.Scan(scanTargets...)
}

// 错误定义
var (
	ErrInvalidEntityType = errors.New("invalid entity type, must be struct")
	ErrTableNameEmpty    = errors.New("table name cannot be empty")
	ErrPrimaryKeyEmpty   = errors.New("primary key cannot be empty")
	ErrEntityNotFound    = errors.New("entity not found")
	ErrInvalidPageParams = errors.New("invalid page parameters")
)
