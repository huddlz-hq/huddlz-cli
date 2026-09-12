package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const showHelp = "Usage: huddlz show <id>\nShows details visible to the current caller. Seat availability and undisclosed links are identified explicitly.\n"

func show(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, showHelp); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" || strings.ContainsAny(args[0], "/\\?#%") || args[0] == "." || args[0] == ".." || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, showHelp)
		return 2
	}
	endpoint, err := huddlEndpoint()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	endpoint = endpoint.JoinPath(args[0])
	endpoint.RawQuery = url.Values{"include": {"group"}, "fields[group]": {"name,slug"}, "fields[huddl]": {"title,description,starts_at,ends_at,time_zone,event_type,physical_location,max_attendees,lifecycle_state,visible_virtual_link,attendance_state,group"}}.Encode()
	body, err := discoveryGet(endpoint)
	if err != nil {
		var status httpStatusError
		if errors.As(err, &status) && (status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound) {
			fmt.Fprintln(stderr, "Huddl unavailable: it does not exist or is not visible to you.")
		} else {
			fmt.Fprintln(stderr, "Huddl lookup failed:", err)
		}
		return 1
	}
	var document struct {
		Included []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"attributes"`
		} `json:"included"`
		Data *struct {
			Relationships struct {
				Group struct {
					Data *huddlIdentity `json:"data"`
				} `json:"group"`
			} `json:"relationships"`
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				VirtualLink      string `json:"visible_virtual_link"`
				AttendanceState  string `json:"attendance_state"`
				Title            string `json:"title"`
				Description      string `json:"description"`
				StartsAt         string `json:"starts_at"`
				EndsAt           string `json:"ends_at"`
				TimeZone         string `json:"time_zone"`
				EventType        string `json:"event_type"`
				PhysicalLocation string `json:"physical_location"`
				MaxAttendees     *int   `json:"max_attendees"`
				LifecycleState   string `json:"lifecycle_state"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &document) != nil || document.Data == nil {
		fmt.Fprintln(stderr, "Huddl lookup failed: invalid or oversized JSON:API response.")
		return 1
	}
	h := document.Data
	a := h.Attributes
	_, startErr := time.Parse(time.RFC3339, a.StartsAt)
	_, endErr := time.Parse(time.RFC3339, a.EndsAt)
	if h.Type != "huddl" || h.ID != args[0] || strings.TrimSpace(a.Title) == "" || startErr != nil || endErr != nil || (a.MaxAttendees != nil && *a.MaxAttendees < 1) {
		fmt.Fprintln(stderr, "Huddl lookup failed: invalid JSON:API resource.")
		return 1
	}
	limit := "Not specified"
	if a.MaxAttendees != nil {
		limit = fmt.Sprint(*a.MaxAttendees)
	}
	groupName, groupSlug := "", ""
	if ref := h.Relationships.Group.Data; ref != nil && ref.Type == "group" {
		for _, g := range document.Included {
			if g.ID == ref.ID && g.Type == "group" {
				groupName, groupSlug = g.Attributes.Name, g.Attributes.Slug
				break
			}
		}
	}
	var output strings.Builder
	for _, field := range [][2]string{{"ID", h.ID}, {"Title", a.Title}, {"Description", a.Description}, {"Starts at", a.StartsAt}, {"Ends at", a.EndsAt}, {"Time zone", a.TimeZone}, {"Type", a.EventType}, {"Location", a.PhysicalLocation}, {"State", a.LifecycleState}, {"Attendance limit", limit}, {"Attendance", a.AttendanceState}, {"Hosting group", groupName}, {"Group slug", groupSlug}, {"Virtual link", a.VirtualLink}} {
		value := tableCell(field[1])
		if strings.TrimSpace(value) == "" {
			value = "Not disclosed"
		}
		fmt.Fprintf(&output, "%s: %s\n", field[0], value)
	}
	fmt.Fprintln(&output, "Seat availability: Not exposed by the API")
	if _, err := io.WriteString(stdout, output.String()); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}
