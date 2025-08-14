package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBulkInsert_Simple(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\)`).
		WithArgs(1, "x", 2, "y").
		WillReturnResult(sqlmock.NewResult(0, 2))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		rows := [][]any{{1, "x"}, {2, "y"}}
		res, err := c.BulkInsert(context.Background(), "t", []string{"a", "b"}, rows)
		if err != nil { return err }
		aff, _ := res.RowsAffected()
		if aff != 2 { t.Fatalf("affected=%d", aff) }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestInsertOnDuplicate_Simple(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\),\(\?,\?\) ON DUPLICATE KEY UPDATE b=VALUES\(b\)`).
		WithArgs(1, "x", 2, "y").
		WillReturnResult(sqlmock.NewResult(0, 2))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		rows := [][]any{{1, "x"}, {2, "y"}}
		res, err := c.InsertOnDuplicate(context.Background(), "t", []string{"a", "b"}, rows, []string{"b"})
		if err != nil { return err }
		aff, _ := res.RowsAffected()
		if aff != 2 { t.Fatalf("affected=%d", aff) }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestBulkInsert_EmptyRows_Error(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	err = p.WithConn(context.Background(), func(c *Conn) error {
		_, err := c.BulkInsert(context.Background(), "t", []string{"a"}, nil)
		if err == nil { t.Fatalf("expected error for empty rows") }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
}

func TestBulkInsert_ColumnMismatch_Error(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	err = p.WithConn(context.Background(), func(c *Conn) error {
		rows := [][]any{{1, "x"}, {2}} // mismatch for second row
		_, err := c.BulkInsert(context.Background(), "t", []string{"a", "b"}, rows)
		if err == nil { t.Fatalf("expected error for mismatch columns") }
		return nil
	})
	if err != nil { t.Fatalf("WithConn: %v", err) }
}

