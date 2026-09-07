// Package cli owns command parsing, output, and process exit codes.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

const help = `huddlz — a CLI for humans and agents

Usage:
  huddlz rsvp <id>
  huddlz auth login --email <email> [--password-stdin]
  huddlz auth status
  huddlz auth logout
  huddlz show <id>
  huddlz help
  huddlz version [--json]
  huddlz search [<query>] [--date <filter>] [--type <type>] [--time-zone <zone>]

Options:
  -h, --help     Show help
  --version      Show version

Search shows one page as a table, defaulting to 20 upcoming huddlz.
Run 'huddlz search --help' for supported filters and options.
Set HUDDLZ_URL to override the default server, https://huddlz.com.
RSVP requires a saved session.
`

// Run executes a command. Exit codes are 0 for success, 1 for execution
// failures, and 2 for invalid usage. Diagnostics go to stderr.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	var err error
	switch {
	case len(args) > 0 && args[0] == "rsvp":
		return rsvp(args[1:], stdout, stderr)
	case len(args) > 0 && args[0] == "auth":
		return auth(args[1:], stdout, stderr)
	case len(args) > 0 && args[0] == "show":
		return show(args[1:], stdout, stderr)
	case len(args) > 0 && args[0] == "search":
		return search(args[1:], stdout, stderr)
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
