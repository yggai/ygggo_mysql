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
		return c.QueryStream(ctx, "SELECT 1", func(_ []any) error { return nil })
	})
	fmt.Println("ygggo_mysql example: stream_query")
}

