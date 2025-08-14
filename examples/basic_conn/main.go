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
	fmt.Println("ygggo_mysql example: basic_conn", ygggo_mysql.Version())
}

