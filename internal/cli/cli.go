// Package cli owns command parsing, output, and process exit codes.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

const help = `huddlz — a CLI for humans and agents

Usage:
  huddlz help
  huddlz version [--json]

Options:
  -h, --help     Show help
  --version      Show version

This initial scaffold supports help and version.
Search, huddl details, RSVP, and authentication are planned.
`

// Run executes a command. Exit codes are 0 for success, 1 for execution
// failures, and 2 for invalid usage. Diagnostics go to stderr.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	var err error
	switch {
	case len(args) == 0 || len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h"):
		_, err = io.WriteString(stdout, help)
	case len(args) == 1 && (args[0] == "version" || args[0] == "--version"):
		_, err = fmt.Fprintf(stdout, "huddlz %s\n", version)
	case len(args) == 2 && args[0] == "version" && args[1] == "--json":
		err = json.NewEncoder(stdout).Encode(struct {
			Version string `json:"version"`
		}{Version: version})
	default:
		fmt.Fprintln(stderr, "Invalid command or arguments. Run 'huddlz help' for usage.")
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}
