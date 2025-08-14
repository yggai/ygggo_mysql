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
		case 1213, 1205, 1290:
			return ErrClassRetryable
		case 1062:
			return ErrClassConflict
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

