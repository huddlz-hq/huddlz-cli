package cli

import (
	"embed"
	"fmt"
	"io"
)

//go:embed completions/*
var completionScripts embed.FS

func completion(args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: huddlz completion bash|zsh|fish\nPrints a script to source in the selected shell.\n"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if len(args) != 1 || (args[0] != "bash" && args[0] != "zsh" && args[0] != "fish") {
		fmt.Fprint(stderr, usage)
		return 2
	}
	script, err := completionScripts.ReadFile("completions/" + args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := stdout.Write(script); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}
