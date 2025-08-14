package ygggo_mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestConn_Exec_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectExec(`INSERT INTO t\(a,b\) VALUES\(\?,\?\)`).WithArgs(1, "x").WillReturnResult(sqlmock.NewResult(1, 1))
	err = p.WithConn(context.Background(), func(c *Conn) error {
		res, err := c.Exec(context.Background(), "INSERT INTO t(a,b) VALUES(?,?)", 1, "x")
		if err != nil { return err }
		affected, _ := res.RowsAffected()
		if affected != 1 { t.Fatalf("affected=%d", affected) }
		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestConn_Query_And_Stream(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	rows := sqlmock.NewRows([]string{"id","name"}).AddRow(1, "a").AddRow(2, "b")
	mock.ExpectQuery("SELECT id,name FROM t").WillReturnRows(rows)

	err = p.WithConn(context.Background(), func(c *Conn) error {
		// Query and read all
		rs, err := c.Query(context.Background(), "SELECT id,name FROM t")
		if err != nil { return err }
		defer rs.Close()
		cnt := 0
		for rs.Next() { cnt++ }
		if cnt != 2 { t.Fatalf("rows cnt=%d", cnt) }
		return nil
	})
	if err != nil { t.Fatalf("WithConn err: %v", err) }

	// Reuse query for streaming
	rows2 := sqlmock.NewRows([]string{"id","name"}).AddRow(1, "a").AddRow(2, "b")
	mock.ExpectQuery("SELECT id,name FROM t").WillReturnRows(rows2)
	callbacks := 0
	err = p.WithConn(context.Background(), func(c *Conn) error {
		return c.QueryStream(context.Background(), "SELECT id,name FROM t", func(_ []any) error {
			callbacks++
			return nil
		})
	})
	if err != nil { t.Fatalf("stream err: %v", err) }
	if callbacks != 2 { t.Fatalf("callbacks=%d", callbacks) }

	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

