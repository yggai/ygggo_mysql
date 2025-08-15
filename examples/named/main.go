package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yggai/ygggo_mysql"
)

type User struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

func main() {
	ctx := context.Background()

	pool, mock, err := ygggo_mysql.NewPoolWithMock(ctx, ygggo_mysql.Config{}, true)
	if err != nil { log.Fatalf("NewPoolWithMock: %v", err) }
	defer pool.Close()

	if mock != nil {
		// NamedExec with struct
		mock.ExpectExec(`INSERT INTO users \(id,name\) VALUES \(\?,\?\)`).WithArgs(1, "Alice").WillReturnResult(ygggo_mysql.NewResult(1, 1))

		// NamedExec with slice of structs (executed individually)
		mock.ExpectExec(`INSERT INTO users \(id,name\) VALUES \(\?,\?\)`).WithArgs(2, "Bob").WillReturnResult(ygggo_mysql.NewResult(2, 1))
		mock.ExpectExec(`INSERT INTO users \(id,name\) VALUES \(\?,\?\)`).WithArgs(3, "Charlie").WillReturnResult(ygggo_mysql.NewResult(3, 1))

		// NamedQuery with map
		rows1 := ygggo_mysql.NewRows([]string{"id", "name"})
		rows1 = ygggo_mysql.AddRow(rows1, 1, "Alice")
		mock.ExpectQuery(`SELECT \* FROM users WHERE id=\?`).WithArgs(1).WillReturnRows(rows1)

		// BuildIn helper
		rows2 := ygggo_mysql.NewRows([]string{"id", "name"})
		rows2 = ygggo_mysql.AddRow(rows2, 1, "Alice")
		rows2 = ygggo_mysql.AddRow(rows2, 2, "Bob")
		mock.ExpectQuery(`SELECT \* FROM users WHERE id IN \(\?,\?,\?\) AND active=\?`).WithArgs(1, 2, 3, true).WillReturnRows(rows2)
	}

	err = pool.WithConn(ctx, func(c *ygggo_mysql.Conn) error {
		// Single struct
		_, err := c.NamedExec(ctx, "INSERT INTO users (id,name) VALUES (:id,:name)", User{ID: 1, Name: "Alice"})
		if err != nil { return err }

		// Slice of structs
		users := []User{{ID: 2, Name: "Bob"}, {ID: 3, Name: "Charlie"}}
		_, err = c.NamedExec(ctx, "INSERT INTO users (id,name) VALUES (:id,:name)", users)
		if err != nil { return err }

		// Map query
		rs, err := c.NamedQuery(ctx, "SELECT * FROM users WHERE id=:id", map[string]any{"id": 1})
		if err != nil { return err }
		defer rs.Close()
		for rs.Next() {
			var id int
			var name string
			if err := rs.Scan(&id, &name); err != nil { return err }
			fmt.Printf("Found user: %d, %s\n", id, name)
		}

		// BuildIn helper
		query, args, err := ygggo_mysql.BuildIn("SELECT * FROM users WHERE id IN (?) AND active=?", []int{1, 2, 3}, true)
		if err != nil { return err }
		rs, err = c.Query(ctx, query, args...)
		if err != nil { return err }
		defer rs.Close()
		count := 0
		for rs.Next() { count++ }
		fmt.Printf("BuildIn found %d users\n", count)

		return nil
	})
	if err != nil { log.Fatalf("WithConn: %v", err) }

	if mock != nil {
		if err := mock.ExpectationsWereMet(); err != nil { log.Fatalf("expectations: %v", err) }
	}
	fmt.Println("ygggo_mysql example: named parameters & BuildIn")
}
