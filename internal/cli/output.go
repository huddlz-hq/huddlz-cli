package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// outputArgs handles the shared presentation flag for commands without queries.
func outputArgs(args []string) ([]string, bool) {
	result := make([]string, 0, len(args))
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json", "--json=true":
			jsonOutput = true
		case "--json=false":
			jsonOutput = false
		default:
			result = append(result, arg)
		}
	}
	return result, jsonOutput
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	if err := json.NewEncoder(stdout).Encode(value); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}

type membershipResult struct {
	HuddlID string `json:"huddl_id"`
	State   string `json:"attendance_state"`
}
