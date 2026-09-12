package main

import (
	"context"
	"golang.org/x/term"
	"os"
	"os/signal"

	"github.com/huddlz-hq/huddlz-cli/internal/cli"
)

// Set at build time with -ldflags '-X main.version=...'.
var version = "dev"

func main() {
	state, _ := term.GetState(int(os.Stdin.Fd()))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr, version)
	stop()
	if state != nil {
		_ = term.Restore(int(os.Stdin.Fd()), state)
	}
	os.Exit(code)
}
