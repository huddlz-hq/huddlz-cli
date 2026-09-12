package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const rsvpHelp = "Usage: huddlz rsvp <id>\n       huddlz rsvp list [--help]\n       huddlz rsvp cancel <id>\n       huddlz rsvp waitlist <id>\nAll RSVP commands accept --json.\nRequires a saved session. Confirms attendance with the server after submitting the RSVP.\n"

func rsvp(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "cancel" {
		return cancelRSVP(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "list" {
		return listRSVPs(args[1:], stdout, stderr)
	}
	args, jsonOutput := outputArgs(args)
	action := "rsvp"
	if len(args) > 0 && args[0] == "waitlist" {
		action = "join_waitlist"
		args = args[1:]
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, rsvpHelp); err != nil {
			return 1
		}
		return 0
	}
	if len(args) != 1 || !validHuddlID(args[0]) {
		fmt.Fprint(stderr, rsvpHelp)
		return 2
	}
	server, err := authServer()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	token, err := loadSession(server)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	id := strings.ToLower(args[0])
	state, err := submitAttendance(server, token, id, action)
	if err != nil {
		var status httpStatusError
		if errors.As(err, &status) && status == http.StatusUnauthorized {
			fmt.Fprintln(stderr, "Authentication required: this session is no longer accepted. Run 'huddlz auth login --email <email>' to log in again.")
		} else if errors.Is(err, errInvalidAttendanceResponse) {
			fmt.Fprintln(stderr, "RSVP response was invalid; attendance was not confirmed.")
		} else {
			fmt.Fprintln(stderr, "RSVP request failed; attendance was not confirmed:", err)
		}
		return 1
	}
	if state != "" || action == "join_waitlist" {
		message := ""
		switch state {
		case "confirmed":
			message = "Attendance confirmed for huddl"
		case "waitlisted":
			message = "Waitlisted for huddl"
		default:
			fmt.Fprintln(stderr, "Attendance state was not confirmed by the API.")
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, stderr, membershipResult{id, state})
		}
		if _, err := fmt.Fprintf(stdout, "%s %s.\n", message, id); err != nil {
			fmt.Fprintln(stderr, "Could not write output:", err)
			return 1
		}
		return 0
	}
	attending, err := attendanceMembership(server, token, id, "attending")
	if err != nil {
		fmt.Fprintln(stderr, "RSVP was submitted, but attendance could not be verified:", err)
		return 1
	}
	if !attending {
		fmt.Fprintln(stderr, "RSVP was submitted, but the server did not confirm attendance. You may still be waitlisted.")
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, stderr, membershipResult{id, "confirmed"})
	}
	if _, err := fmt.Fprintf(stdout, "Attendance confirmed for huddl %s.\n", id); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}

type huddlIdentity struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func validHuddlID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}
