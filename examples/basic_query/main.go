package main

import (
	"context"
	"fmt"
	"github.com/yggai/ygggo_mysql"
)

func main() {
	ctx := context.Background()
	pool, _ := ygggo_mysql.NewPool(ctx, ygggo_mysql.Config{})
	defer pool.Close()
	_ = pool.WithConn(ctx, func(c *ygggo_mysql.Conn) error {
		var v struct{}
		_ = ygggo_mysql.Get(ctx, c, &v, "SELECT 1")
		return nil
	})
	fmt.Println("ygggo_mysql example: basic_query")
}

