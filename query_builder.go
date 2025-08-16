package ygggo_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// QueryBuilder provides a fluent interface for building SQL queries
type QueryBuilder struct {
	conn DatabaseConn
	
	// Query type and basic structure
	queryType string
	
	// SELECT fields
	selectFields []string
	fromTable    string
	joins        []string
	whereConditions []whereCondition
	groupByFields   []string
	havingConditions []whereCondition
	orderByFields   []string
	limitValue      *int
	offsetValue     *int
	
	// INSERT fields
	insertTable       string
	insertColumns     []string
	insertValues      [][]any
	onDuplicateUpdate map[string]any
	
	// UPDATE fields
	updateTable string
	setFields   map[string]any
	
	// DELETE fields
	deleteTable string
	
	// Collected arguments for parameter binding
	args []any
}

// whereCondition represents a WHERE or HAVING condition
type whereCondition struct {
	condition string
	args      []any
}

// NewQueryBuilder creates a new query builder instance
func NewQueryBuilder(conn DatabaseConn) *QueryBuilder {
	return &QueryBuilder{
		conn:              conn,
		setFields:         make(map[string]any),
		onDuplicateUpdate: make(map[string]any),
	}
}

// Select starts a SELECT query with the specified columns
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.queryType = "SELECT"
	qb.selectFields = columns
	return qb
}

// From specifies the table for the SELECT query
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.fromTable = table
	return qb
}

// Join adds a JOIN clause to the query
func (qb *QueryBuilder) Join(joinClause string) *QueryBuilder {
	qb.joins = append(qb.joins, joinClause)
	return qb
}

// Where adds a WHERE condition to the query
func (qb *QueryBuilder) Where(condition string, args ...any) *QueryBuilder {
	qb.whereConditions = append(qb.whereConditions, whereCondition{
		condition: condition,
		args:      args,
	})
	return qb
}

// GroupBy adds a GROUP BY clause to the query
func (qb *QueryBuilder) GroupBy(groupBy ...string) *QueryBuilder {
	qb.groupByFields = append(qb.groupByFields, groupBy...)
	return qb
}

// Having adds a HAVING condition to the query
func (qb *QueryBuilder) Having(condition string, args ...any) *QueryBuilder {
	qb.havingConditions = append(qb.havingConditions, whereCondition{
		condition: condition,
		args:      args,
	})
	return qb
}

// OrderBy adds an ORDER BY clause to the query
func (qb *QueryBuilder) OrderBy(orderBy string) *QueryBuilder {
	qb.orderByFields = append(qb.orderByFields, orderBy)
	return qb
}

// Limit sets the LIMIT for the query
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitValue = &limit
	return qb
}

// Offset sets the OFFSET for the query
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetValue = &offset
	return qb
}

// Insert starts an INSERT query for the specified table
func (qb *QueryBuilder) Insert(table string) *QueryBuilder {
	qb.queryType = "INSERT"
	qb.insertTable = table
	return qb
}

// Values sets the values for an INSERT query using a map
func (qb *QueryBuilder) Values(values map[string]any) *QueryBuilder {
	if len(values) == 0 {
		return qb
	}

	// Extract columns and values in consistent order
	columns := make([]string, 0, len(values))
	vals := make([]any, 0, len(values))

	for col, val := range values {
		columns = append(columns, col)
		vals = append(vals, val)
	}

	qb.insertColumns = columns
	qb.insertValues = [][]any{vals}
	return qb
}

// Update starts an UPDATE query for the specified table
func (qb *QueryBuilder) Update(table string) *QueryBuilder {
	qb.queryType = "UPDATE"
	qb.updateTable = table
	return qb
}

// Set adds a column=value assignment for UPDATE queries
func (qb *QueryBuilder) Set(column string, value any) *QueryBuilder {
	qb.setFields[column] = value
	return qb
}

// Delete starts a DELETE query for the specified table
func (qb *QueryBuilder) Delete(table string) *QueryBuilder {
	qb.queryType = "DELETE"
	qb.deleteTable = table
	return qb
}

// Query executes the built SELECT query and returns rows
func (qb *QueryBuilder) Query(ctx context.Context) (*sql.Rows, error) {
	if qb.queryType != "SELECT" {
		return nil, fmt.Errorf("Query() can only be called on SELECT queries")
	}

	query, args := qb.buildSelectQuery()
	return qb.conn.Query(ctx, query, args...)
}

// Exec executes INSERT, UPDATE, or DELETE queries and returns the result
func (qb *QueryBuilder) Exec(ctx context.Context) (sql.Result, error) {
	var query string
	var args []any

	switch qb.queryType {
	case "INSERT":
		query, args = qb.buildInsertQuery()
	case "UPDATE":
		query, args = qb.buildUpdateQuery()
	case "DELETE":
		query, args = qb.buildDeleteQuery()
	default:
		return nil, fmt.Errorf("Exec() can only be called on INSERT, UPDATE, or DELETE queries")
	}

	return qb.conn.Exec(ctx, query, args...)
}

// buildSelectQuery constructs the SELECT SQL query and collects arguments
func (qb *QueryBuilder) buildSelectQuery() (string, []any) {
	var query strings.Builder
	var args []any
	
	// SELECT clause
	query.WriteString("SELECT ")
	if len(qb.selectFields) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString(strings.Join(qb.selectFields, ", "))
	}
	
	// FROM clause
	if qb.fromTable != "" {
		query.WriteString(" FROM ")
		query.WriteString(qb.fromTable)
	}
	
	// JOIN clauses
	for _, join := range qb.joins {
		query.WriteString(" ")
		query.WriteString(join)
	}
	
	// WHERE clause
	if len(qb.whereConditions) > 0 {
		query.WriteString(" WHERE ")
		for i, condition := range qb.whereConditions {
			if i > 0 {
				query.WriteString(" AND ")
			}
			query.WriteString("(")
			query.WriteString(condition.condition)
			query.WriteString(")")
			args = append(args, condition.args...)
		}
	}
	
	// GROUP BY clause
	if len(qb.groupByFields) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(qb.groupByFields, ", "))
	}
	
	// HAVING clause
	if len(qb.havingConditions) > 0 {
		query.WriteString(" HAVING ")
		for i, condition := range qb.havingConditions {
			if i > 0 {
				query.WriteString(" AND ")
			}
			query.WriteString("(")
			query.WriteString(condition.condition)
			query.WriteString(")")
			args = append(args, condition.args...)
		}
	}
	
	// ORDER BY clause
	if len(qb.orderByFields) > 0 {
		query.WriteString(" ORDER BY ")
		query.WriteString(strings.Join(qb.orderByFields, ", "))
	}
	
	// LIMIT clause
	if qb.limitValue != nil {
		query.WriteString(" LIMIT ")
		query.WriteString(strconv.Itoa(*qb.limitValue))
	}
	
	// OFFSET clause
	if qb.offsetValue != nil {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(*qb.offsetValue))
	}
	
	return query.String(), args
}

// buildInsertQuery constructs the INSERT SQL query and collects arguments
func (qb *QueryBuilder) buildInsertQuery() (string, []any) {
	var query strings.Builder
	var args []any

	query.WriteString("INSERT INTO ")
	query.WriteString(qb.insertTable)

	if len(qb.insertColumns) > 0 && len(qb.insertValues) > 0 {
		// Column names
		query.WriteString(" (")
		query.WriteString(strings.Join(qb.insertColumns, ", "))
		query.WriteString(") VALUES ")

		// Values
		for i, row := range qb.insertValues {
			if i > 0 {
				query.WriteString(", ")
			}
			query.WriteString("(")
			placeholders := make([]string, len(row))
			for j := range row {
				placeholders[j] = "?"
			}
			query.WriteString(strings.Join(placeholders, ", "))
			query.WriteString(")")
			args = append(args, row...)
		}
	}

	return query.String(), args
}

// buildUpdateQuery constructs the UPDATE SQL query and collects arguments
func (qb *QueryBuilder) buildUpdateQuery() (string, []any) {
	var query strings.Builder
	var args []any

	query.WriteString("UPDATE ")
	query.WriteString(qb.updateTable)

	if len(qb.setFields) > 0 {
		query.WriteString(" SET ")
		setParts := make([]string, 0, len(qb.setFields))
		for col, val := range qb.setFields {
			setParts = append(setParts, col+" = ?")
			args = append(args, val)
		}
		query.WriteString(strings.Join(setParts, ", "))
	}

	// WHERE clause
	if len(qb.whereConditions) > 0 {
		query.WriteString(" WHERE ")
		for i, condition := range qb.whereConditions {
			if i > 0 {
				query.WriteString(" AND ")
			}
			query.WriteString("(")
			query.WriteString(condition.condition)
			query.WriteString(")")
			args = append(args, condition.args...)
		}
	}

	return query.String(), args
}

// buildDeleteQuery constructs the DELETE SQL query and collects arguments
func (qb *QueryBuilder) buildDeleteQuery() (string, []any) {
	var query strings.Builder
	var args []any

	query.WriteString("DELETE FROM ")
	query.WriteString(qb.deleteTable)

	// WHERE clause
	if len(qb.whereConditions) > 0 {
		query.WriteString(" WHERE ")
		for i, condition := range qb.whereConditions {
			if i > 0 {
				query.WriteString(" AND ")
			}
			query.WriteString("(")
			query.WriteString(condition.condition)
			query.WriteString(")")
			args = append(args, condition.args...)
		}
	}

	return query.String(), args
}
