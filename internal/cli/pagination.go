package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// nextPageMessage reads server pagination metadata without following a server URL.
func nextPageMessage(next json.RawMessage, options searchOptions, endpoint *url.URL) (string, error) {
	if len(next) == 0 {
		return "Pagination information unavailable.", nil
	}
	if string(next) == "null" {
		return "No more results.", nil
	}
	var link string
	if err := json.Unmarshal(next, &link); err != nil {
		var object struct {
			Href string `json:"href"`
		}
		if err := json.Unmarshal(next, &object); err != nil {
			return "", fmt.Errorf("invalid next-page link")
		}
		link = object.Href
	}
	parsed, err := url.Parse(link)
	if err != nil || link == "" {
		return "", fmt.Errorf("invalid next-page link")
	}
	parsed = endpoint.ResolveReference(parsed)
	if parsed.Scheme != endpoint.Scheme || parsed.Host != endpoint.Host || parsed.Path != endpoint.Path || parsed.User != nil || parsed.Fragment != "" {
		return "", fmt.Errorf("next-page link does not refer to this search endpoint")
	}
	params, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(params["page[offset]"]) != 1 {
		return "", fmt.Errorf("invalid next-page offset")
	}
	offset, err := strconv.Atoi(params.Get("page[offset]"))
	if err != nil || offset <= options.offset {
		return "", fmt.Errorf("next-page offset must advance the search")
	}
	command := "huddlz search --anywhere"
	if location := options.location; location != nil {
		command = fmt.Sprintf("huddlz search --lat %g --lng %g --radius %d", location.latitude, location.longitude, location.radius)
	}
	command += " --date " + options.date
	if options.kind != "" {
		command += " --type " + options.kind
	}
	command += " --time-zone " + options.timeZone
	command += fmt.Sprintf(" --limit %d --offset %d", options.limit, offset)
	if options.query != "" {
		// End options before the query; quote it for POSIX shells.
		command += " -- '" + strings.ReplaceAll(options.query, "'", `'\''`) + "'"
	}
	return "Next page: " + command, nil
}
