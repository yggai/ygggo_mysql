package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
)

func TestWithinTx_Retry_DeadlockThenSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db, retry: RetryPolicy{MaxAttempts: 3}}

	// First attempt deadlock, rollback; second attempt ok, commit
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE t SET a=\? WHERE id=\?`).WithArgs(2,1).WillReturnError(&mysql.MySQLError{Number:1213, Message:"deadlock"})
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE t SET a=\? WHERE id=\?`).WithArgs(2,1).WillReturnResult(sqlmock.NewResult(0,1))
	mock.ExpectCommit()

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "UPDATE t SET a=? WHERE id=?", 2, 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestWithinTx_Retry_ReadOnlyThenSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db, retry: RetryPolicy{MaxAttempts: 2}}

	// First attempt read-only error, rollback; second attempt ok, commit
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnError(&mysql.MySQLError{Number:1290, Message:"read only"})
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO t\(a\) VALUES\(\?\)`).WithArgs(1).WillReturnResult(sqlmock.NewResult(1,1))
	mock.ExpectCommit()

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "INSERT INTO t(a) VALUES(?)", 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

