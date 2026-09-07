package main

import (
	"os"

	"github.com/huddlz-hq/huddlz-cli/internal/cli"
)

// Set at build time with -ldflags '-X main.version=...'.
var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
