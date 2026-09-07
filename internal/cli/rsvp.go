package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const rsvpHelp = "Usage: huddlz rsvp <id>\n       huddlz rsvp list [--help]\nRequires a saved session. Confirms attendance with the server after submitting the RSVP.\n"

func rsvp(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "list" {
		return listRSVPs(args[1:], stdout, stderr)
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
	payload, _ := json.Marshal(struct {
		Data huddlIdentity `json:"data"`
	}{huddlIdentity{ID: id, Type: "huddl"}})
	var response struct {
		Data *huddlIdentity `json:"data"`
	}
	endpoint := server.JoinPath("api/json/huddlz", id, "rsvp")
	if err := sessionRequest(endpoint, http.MethodPatch, token, payload, &response, "application/vnd.api+json"); err != nil {
		var status httpStatusError
		if errors.As(err, &status) && status == http.StatusUnauthorized {
			fmt.Fprintln(stderr, "Authentication required: this session is no longer accepted. Run 'huddlz auth login --email <email>' to log in again.")
		} else {
			fmt.Fprintln(stderr, "RSVP request failed; attendance was not confirmed:", err)
		}
		return 1
	}
	if response.Data == nil || response.Data.ID != id || response.Data.Type != "huddl" {
		fmt.Fprintln(stderr, "RSVP response was invalid; attendance was not confirmed.")
		return 1
	}
	endpoint = server.JoinPath("api/json/huddlz")
	endpoint.RawQuery = url.Values{"filter[id][eq]": {id}, "relationship": {"attending"}, "date_filter": {"all"}, "search_time_zone": {"Etc/UTC"}, "page[limit]": {"1"}}.Encode()
	var attendance struct {
		Data *[]huddlIdentity `json:"data"`
	}
	if err := sessionRequest(endpoint, http.MethodGet, token, nil, &attendance, "application/vnd.api+json"); err != nil {
		fmt.Fprintln(stderr, "RSVP was submitted, but attendance could not be verified:", err)
		return 1
	}
	if attendance.Data == nil || len(*attendance.Data) != 1 || (*attendance.Data)[0].ID != id || (*attendance.Data)[0].Type != "huddl" {
		fmt.Fprintln(stderr, "RSVP was submitted, but the server did not confirm attendance. You may still be waitlisted.")
		return 1
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
