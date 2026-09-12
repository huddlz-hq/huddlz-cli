package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const cancelRSVPHelp = "Usage: huddlz rsvp cancel <id>\nCancels confirmed attendance or leaves a waitlist, then verifies both memberships are absent.\n"

func cancelRSVP(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, cancelRSVPHelp); err != nil {
			return 1
		}
		return 0
	}
	if len(args) != 1 || !validHuddlID(args[0]) {
		fmt.Fprint(stderr, cancelRSVPHelp)
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
	if _, err := submitAttendance(server, token, id, "cancel_rsvp"); err != nil {
		var status httpStatusError
		if errors.As(err, &status) && status == http.StatusUnauthorized {
			fmt.Fprintln(stderr, "Authentication required: this session is no longer accepted. Run 'huddlz auth login --email <email>' to log in again.")
		} else if errors.Is(err, errInvalidAttendanceResponse) {
			fmt.Fprintln(stderr, "Cancellation response was invalid; cancellation was not confirmed.")
		} else {
			fmt.Fprintln(stderr, "Cancellation request failed; cancellation was not confirmed:", err)
		}
		return 1
	}
	for _, relationship := range []string{"attending", "waitlisted"} {
		member, err := attendanceMembership(server, token, id, relationship)
		if err != nil {
			fmt.Fprintln(stderr, "Cancellation was submitted, but membership removal could not be verified:", err)
			return 1
		}
		if member {
			fmt.Fprintln(stderr, "Cancellation was submitted, but the server did not confirm removal from attendance and waitlist.")
			return 1
		}
	}
	if _, err := fmt.Fprintf(stdout, "RSVP cancelled for huddl %s. No confirmed or waitlisted membership remains.\n", id); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}
