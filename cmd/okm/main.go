package main

import (
	"context"
	"fmt"
	"os"

	"okm/internal/okm"
)

func main() {
	if err := okm.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "okm:", err)
		os.Exit(1)
	}
}
