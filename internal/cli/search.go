package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/pflag"
)

// Anonymous searches have no profile defaults, so their scope is everywhere.
func search(args []string, stdout, stderr io.Writer) int {
	options, err := parseSearchOptions(args)
	if errors.Is(err, pflag.ErrHelp) {
		if _, err := io.WriteString(stdout, searchHelp); err != nil {
			fmt.Fprintln(stderr, "Could not write output:", err)
			return 1
		}
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprint(stderr, searchHelp)
		return 2
	}
	endpoint, err := huddlEndpoint()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := applyProfileDefaults(&options); err != nil {
		fmt.Fprintln(stderr, "Could not read search defaults:", err)
		return 1
	}
	params := url.Values{"date_filter": {options.date}, "search_time_zone": {options.timeZone}, "sort": {"starts_at"}, "page[limit]": {strconv.Itoa(options.limit)}, "page[offset]": {strconv.Itoa(options.offset)}}
	if options.query != "" {
		params.Set("query", options.query)
	}
	if options.kind != "" {
		params.Set("event_type", options.kind)
	}
	if location := options.location; location != nil {
		params.Set("search_latitude", fmt.Sprint(location.latitude))
		params.Set("search_longitude", fmt.Sprint(location.longitude))
		params.Set("distance_miles", strconv.Itoa(location.radius))
	}
	endpoint.RawQuery = params.Encode()
	body, err := getJSONAPI(endpoint)
	if err != nil {
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			fmt.Fprintln(stderr, "Search request failed:", err)
		} else {
			fmt.Fprintln(stderr, "Search failed:", err)
		}
		return 1
	}
	var document struct {
		Links struct {
			Next json.RawMessage `json:"next"`
		} `json:"links"`
		Data *[]searchHuddl `json:"data"`
	}
	if json.Unmarshal(body, &document) != nil || document.Data == nil {
		fmt.Fprintln(stderr, "Search failed: invalid or oversized JSON:API response.")
		return 1
	}
	for _, huddl := range *document.Data {
		if !huddl.valid() {
			fmt.Fprintln(stderr, "Search failed: invalid or oversized JSON:API response.")
			return 1
		}
	}
	if len(*document.Data) > options.limit {
		fmt.Fprintln(stderr, "Search failed: API returned more results than requested.")
		return 1
	}
	page, err := nextPage(document.Links.Next, options, endpoint)
	if err != nil {
		fmt.Fprintln(stderr, "Search failed:", err)
		return 1
	}
	if options.jsonOutput {
		var location *searchJSONLocation
		if l := options.location; l != nil {
			location = &searchJSONLocation{l.latitude, l.longitude, l.radius}
		}
		result := searchJSONResult{*document.Data, searchJSONContext{options.query, options.date, options.kind, options.timeZone, location}, page}
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, "Could not write output:", err)
			return 1
		}
		return 0
	}
	var output strings.Builder
	kind := options.kind
	if kind == "" {
		kind = "all types"
	}
	scope := "Searching everywhere"
	if location := options.location; location != nil {
		scope = fmt.Sprintf("Searching within %d miles of %g, %g", location.radius, location.latitude, location.longitude)
	}
	fmt.Fprintf(&output, "%s · %s · %s · calendar timezone: %s · up to %d results · offset %d\n", scope, options.date, kind, options.timeZone, options.limit, options.offset)
	fmt.Fprintln(&output)
	if len(*document.Data) == 0 {
		fmt.Fprintln(&output, "No matching huddlz.")
	} else {
		writeHuddlTable(&output, *document.Data)
	}
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, page.message())
	if _, err := io.WriteString(stdout, output.String()); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}

func tableCell(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, value)
}

type searchHuddl struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Title            string `json:"title"`
		StartsAt         string `json:"starts_at"`
		PhysicalLocation string `json:"physical_location"`
	} `json:"attributes"`
}
