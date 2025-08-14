package ygggo_mysql

import (
	"context"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type row struct {
	A int    `db:"a"`
	B string `db:"b"`
}

func TestNamedExec_WithStruct(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\)`).WithArgs(1, "x").WillReturnResult(sqlmock.NewResult(1,1))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		_, err := c.NamedExec(context.Background(), "INSERT INTO t (a,b) VALUES (:a,:b)", row{A:1, B:"x"})
		return err
	})
	if err != nil { t.Fatalf("NamedExec err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestNamedExec_WithSliceStructs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\)`).WithArgs(1, "x").WillReturnResult(sqlmock.NewResult(1,1))
	mock.ExpectExec(`INSERT INTO t \(a,b\) VALUES \(\?,\?\)`).WithArgs(2, "y").WillReturnResult(sqlmock.NewResult(1,1))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		rows := []row{{1,"x"},{2,"y"}}
		_, err := c.NamedExec(context.Background(), "INSERT INTO t (a,b) VALUES (:a,:b)", rows)
		return err
	})
	if err != nil { t.Fatalf("NamedExec slice err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestNamedQuery_WithMap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectQuery(`SELECT \* FROM t WHERE id=\?`).WithArgs(42).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		rs, err := c.NamedQuery(context.Background(), "SELECT * FROM t WHERE id=:id", map[string]any{"id": 42})
		if err != nil { return err }
		defer rs.Close()
		var ids []int
		for rs.Next() {
			var id int
			if err := rs.Scan(&id); err != nil { return err }
			ids = append(ids, id)
		}
		if !reflect.DeepEqual(ids, []int{42}) { t.Fatalf("ids=%v", ids) }
		return nil
	})
	if err != nil { t.Fatalf("NamedQuery err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestIn_Helper_ExpandsSlice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()
	p := &Pool{db: db}

	mock.ExpectQuery(`SELECT \* FROM t WHERE id IN \(\?,\?,\?\) AND kind=\?`).WithArgs(1,2,3,"a").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))

	err = p.WithConn(context.Background(), func(c *Conn) error {
		q, args, err := BuildIn("SELECT * FROM t WHERE id IN (?) AND kind=?", []int{1,2,3}, "a")
		if err != nil { return err }
		rs, err := c.Query(context.Background(), q, args...)
		if err != nil { return err }
		defer rs.Close()
		return nil
	})
	if err != nil { t.Fatalf("BuildIn err: %v", err) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

