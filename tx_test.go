package ygggo_mysql

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	mysql "github.com/go-sql-driver/mysql"
)

func TestWithinTx_CommitOnSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

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

func TestWithinTx_RollbackOnFnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectBegin()
	mock.ExpectRollback()

	sentinel := errors.New("boom")
	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error { return sentinel })
	if !errors.Is(err, sentinel) { t.Fatalf("expected sentinel, got %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestWithinTx_RetryOnDeadlock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db, retry: RetryPolicy{MaxAttempts: 2}}

	// First attempt: deadlock error -> rollback
	mock.ExpectBegin()
	deadlock := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
	mock.ExpectExec(`UPDATE t SET a=\? WHERE id=\?`).WithArgs(2, 1).WillReturnError(deadlock)
	mock.ExpectRollback()
	// Second attempt: success -> commit
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE t SET a=\? WHERE id=\?`).WithArgs(2, 1).WillReturnResult(sqlmock.NewResult(0,1))
	mock.ExpectCommit()

	err = p.WithinTx(context.Background(), func(tx DatabaseTx) error {
		_, err := tx.Exec(context.Background(), "UPDATE t SET a=? WHERE id=?", 2, 1)
		return err
	})
	if err != nil { t.Fatalf("WithinTx err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

