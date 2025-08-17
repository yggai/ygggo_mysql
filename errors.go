package ygggo_mysql

import (
	"errors"
	mysql "github.com/go-sql-driver/mysql"
)

// ErrorClass is a placeholder for error classification.
type ErrorClass int

const (
	ErrClassUnknown ErrorClass = iota
	ErrClassRetryable
	ErrClassConflict
	ErrClassReadonly
	ErrClassConstraint
)

// Classify classifies error into a high-level class.
func Classify(err error) ErrorClass {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		switch me.Number {
		// Retryable transient errors
		case 1213, // ER_LOCK_DEADLOCK
			1205: // ER_LOCK_WAIT_TIMEOUT
			return ErrClassRetryable
		// Readonly mode
		case 1290: // ER_OPTION_PREVENTS_STATEMENT (often read-only mode)
			return ErrClassReadonly
		// Conflicts (duplicates)
		case 1062, // ER_DUP_ENTRY
			1022: // ER_DUP_KEY
			return ErrClassConflict
		// Constraints (not-null, foreign key, check)
		case 1048, // ER_BAD_NULL_ERROR
			1452, // ER_NO_REFERENCED_ROW_2
			1451, // ER_ROW_IS_REFERENCED_2
			3819: // ER_CHECK_CONSTRAINT_VIOLATED
			return ErrClassConstraint
		}
	}
	return ErrClassUnknown
}

// adapt wraps driver error into local mysqlMySQLError for decoupled checks.
func adapt(err error) error {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return &mysqlMySQLError{Number: me.Number}
	}
	return err
}
