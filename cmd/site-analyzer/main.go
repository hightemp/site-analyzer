// Command site-analyzer detects technologies used by one or more websites.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"site-analyzer/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(app.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
