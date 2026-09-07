package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type searchPage struct {
	Known       bool    `json:"known"`
	Limit       int     `json:"limit"`
	Offset      int     `json:"offset"`
	NextOffset  *int    `json:"next_offset"`
	NextCommand *string `json:"next_command"`
}

func (page searchPage) message() string {
	if !page.Known {
		return "Pagination information unavailable."
	}
	if page.NextCommand == nil {
		return "No more results."
	}
	return "Next page: " + *page.NextCommand
}

// nextPage reads server pagination metadata without following a server URL.
func readNextPage(next json.RawMessage, limit, currentOffset int, endpoint *url.URL) (searchPage, error) {
	page := searchPage{Limit: limit, Offset: currentOffset}
	if len(next) == 0 {
		return page, nil
	}
	if string(next) == "null" {
		page.Known = true
		return page, nil
	}
	var link string
	if err := json.Unmarshal(next, &link); err != nil {
		var object struct {
			Href string `json:"href"`
		}
		if err := json.Unmarshal(next, &object); err != nil {
			return page, fmt.Errorf("invalid next-page link")
		}
		link = object.Href
	}
	parsed, err := url.Parse(link)
	if err != nil || link == "" {
		return page, fmt.Errorf("invalid next-page link")
	}
	parsed = endpoint.ResolveReference(parsed)
	if parsed.Scheme != endpoint.Scheme || parsed.Host != endpoint.Host || parsed.Path != endpoint.Path || parsed.User != nil || parsed.Fragment != "" {
		return page, fmt.Errorf("next-page link does not refer to this search endpoint")
	}
	params, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(params["page[offset]"]) != 1 {
		return page, fmt.Errorf("invalid next-page offset")
	}
	offset, err := strconv.Atoi(params.Get("page[offset]"))
	if err != nil || offset <= currentOffset {
		return page, fmt.Errorf("next-page offset must advance the search")
	}
	page.Known = true
	page.NextOffset = &offset
	return page, nil
}

func nextPage(next json.RawMessage, options searchOptions, endpoint *url.URL) (searchPage, error) {
	page, err := readNextPage(next, options.limit, options.offset, endpoint)
	if err != nil || page.NextOffset == nil {
		return page, err
	}
	offset := *page.NextOffset
	command := "huddlz search --anywhere"
	if location := options.location; location != nil {
		command = fmt.Sprintf("huddlz search --lat %g --lng %g --radius %d", location.latitude, location.longitude, location.radius)
	}
	if options.jsonOutput {
		command += " --json"
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
	page.Known = true
	page.NextOffset = &offset
	page.NextCommand = &command
	return page, nil
}
