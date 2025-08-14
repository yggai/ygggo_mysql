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
	_ = pool.WithinTx(ctx, func(tx *ygggo_mysql.Tx) error { return nil })
	fmt.Println("ygggo_mysql example: tx_retry")
}

