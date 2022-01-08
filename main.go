package main

import (
	"context"
	"os"
)

func main() {
	ctx := context.Background()
	if err := New().Cmd(ctx, os.Args); err != nil {
		os.Exit(1)
	}
}
