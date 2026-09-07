package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

const listRSVPHelp = `Usage: huddlz rsvp list [options]
  --status <status>   all (default), confirmed, or waitlisted
  --limit <number>    Results per status, 1–100 (default: 20)
  --offset <number>   Offset per selected status (default: 0)
  --json             Return structured output
  -h, --help         Show help
Lists visible huddlz across all dates, sorted by starts_at within each status.
`

type rsvpListGroup struct {
	Status     string        `json:"status"`
	Data       []searchHuddl `json:"data"`
	Pagination searchPage    `json:"pagination"`
}

func listRSVPs(args []string, stdout, stderr io.Writer) int {
	var status string
	var limit, offset int
	var jsonOutput bool
	flags := pflag.NewFlagSet("rsvp list", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&status, "status", "all", "Membership status")
	flags.IntVar(&limit, "limit", 20, "Results per status")
	flags.IntVar(&offset, "offset", 0, "Offset per status")
	flags.BoolVar(&jsonOutput, "json", false, "Return JSON")
	err := flags.Parse(args)
	if errors.Is(err, pflag.ErrHelp) {
		if _, err := io.WriteString(stdout, listRSVPHelp); err != nil {
			return 1
		}
		return 0
	}
	if err != nil || flags.NArg() != 0 || limit < 1 || limit > 100 || offset < 0 || (status != "all" && status != "confirmed" && status != "waitlisted") {
		fmt.Fprint(stderr, listRSVPHelp)
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
	statuses := []string{status}
	if status == "all" {
		statuses = []string{"confirmed", "waitlisted"}
	}
	groups := make([]rsvpListGroup, 0, len(statuses))
	for _, status := range statuses {
		group, err := fetchRSVPGroup(server, token, status, limit, offset, jsonOutput)
		if err != nil {
			var code httpStatusError
			if errors.As(err, &code) && code == http.StatusUnauthorized {
				fmt.Fprintln(stderr, "Authentication required: this session is no longer accepted. Run 'huddlz auth login --email <email>' to log in again.")
			} else {
				fmt.Fprintln(stderr, "Could not list RSVPs:", err)
			}
			return 1
		}
		groups = append(groups, group)
	}
	if jsonOutput {
		err = json.NewEncoder(stdout).Encode(struct {
			Groups []rsvpListGroup `json:"groups"`
		}{groups})
	} else {
		var output strings.Builder
		fmt.Fprintln(&output, "My RSVPs · all dates · visible huddlz")
		for _, group := range groups {
			fmt.Fprintf(&output, "\n%s · up to %d results · offset %d\n", group.Status, limit, offset)
			if len(group.Data) == 0 {
				fmt.Fprintln(&output, "No matching RSVPs.")
			} else {
				writeHuddlTable(&output, group.Data)
			}
			fmt.Fprintln(&output, group.Pagination.message())
		}
		_, err = io.WriteString(stdout, output.String())
	}
	if err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}

func fetchRSVPGroup(server *url.URL, token, status string, limit, offset int, jsonOutput bool) (rsvpListGroup, error) {
	group := rsvpListGroup{Status: status}
	relationship := status
	if status == "confirmed" {
		relationship = "attending"
	}
	endpoint := server.JoinPath("api/json/huddlz")
	endpoint.RawQuery = url.Values{"relationship": {relationship}, "date_filter": {"all"}, "search_time_zone": {"Etc/UTC"}, "sort": {"starts_at"}, "page[limit]": {strconv.Itoa(limit)}, "page[offset]": {strconv.Itoa(offset)}}.Encode()
	var document struct {
		Data  *[]searchHuddl `json:"data"`
		Links struct {
			Next json.RawMessage `json:"next"`
		} `json:"links"`
	}
	if err := sessionRequest(endpoint, http.MethodGet, token, nil, &document, "application/vnd.api+json"); err != nil {
		return group, err
	}
	if document.Data == nil || len(*document.Data) > limit {
		return group, fmt.Errorf("invalid RSVP search response")
	}
	for _, h := range *document.Data {
		if !h.valid() {
			return group, fmt.Errorf("invalid RSVP search resource")
		}
	}
	page, err := readNextPage(document.Links.Next, limit, offset, endpoint)
	if err != nil {
		return group, err
	}
	if page.NextOffset != nil {
		command := fmt.Sprintf("huddlz rsvp list --status %s --limit %d --offset %d", status, limit, *page.NextOffset)
		if jsonOutput {
			command += " --json"
		}
		page.NextCommand = &command
	}
	group.Data = *document.Data
	group.Pagination = page
	return group, nil
}
