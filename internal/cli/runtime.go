package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type invocation struct {
	ctx     context.Context
	timeout time.Duration
}

// Run executes a CLI invocation without an external cancellation signal.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	return RunContext(context.Background(), args, stdout, stderr, version)
}

// RunContext cancels in-flight HTTP requests when the caller cancels ctx.
func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	value := os.Getenv("HUDDLZ_TIMEOUT")
	if value == "" {
		value = "15s"
	}
	cleaned := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			cleaned = append(cleaned, args[i:]...)
			break
		}
		if arg == "--timeout" {
			i++
			if i == len(args) {
				fmt.Fprintln(stderr, "--timeout requires a positive duration, such as 15s")
				return 2
			}
			value = args[i]
		} else if strings.HasPrefix(arg, "--timeout=") {
			value = strings.TrimPrefix(arg, "--timeout=")
		} else {
			cleaned = append(cleaned, arg)
		}
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		fmt.Fprintln(stderr, "--timeout / HUDDLZ_TIMEOUT must be a positive duration, such as 15s")
		return 2
	}
	inv := invocation{ctx: ctx, timeout: timeout}
	code := inv.run(cleaned, stdout, stderr, version)
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "Interrupted. A submitted operation may have completed; check its status before repeating it.")
		return 130
	}
	return code
}
