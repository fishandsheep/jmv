package main

import (
	"context"
	"fmt"
	"os"

	"jmv/internal/jmv"
)

func main() {
	if err := jmv.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "jmv:", err)
		os.Exit(1)
	}
}
