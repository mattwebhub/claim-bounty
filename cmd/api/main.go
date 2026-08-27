package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mattwebhub/micro1-go-template/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
