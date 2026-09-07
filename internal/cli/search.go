package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"
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
	base := os.Getenv("HUDDLZ_URL")
	if base == "" {
		base = "https://huddlz.com"
	}
	endpoint, err := url.Parse(base)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		fmt.Fprintln(stderr, "HUDDLZ_URL must be an HTTP(S) server URL without credentials, query, or fragment.")
		return 2
	}
	endpoint = endpoint.JoinPath("api/json/huddlz")
	params := url.Values{"date_filter": {options.date}, "search_time_zone": {options.timeZone}, "sort": {"starts_at"}, "page[limit]": {"20"}}
	if options.query != "" {
		params.Set("query", options.query)
	}
	if options.kind != "" {
		params.Set("event_type", options.kind)
	}
	endpoint.RawQuery = params.Encode()
	request, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		fmt.Fprintln(stderr, "Could not create search request:", err)
		return 1
	}
	request.Header.Set("Accept", "application/vnd.api+json")
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		fmt.Fprintln(stderr, "Search request failed:", err)
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Fprintf(stderr, "Search failed: HTTP %d %s\n", response.StatusCode, http.StatusText(response.StatusCode))
		return 1
	}
	var document struct {
		Data *[]struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Title            string `json:"title"`
				StartsAt         string `json:"starts_at"`
				PhysicalLocation string `json:"physical_location"`
			} `json:"attributes"`
		} `json:"data"`
	}
	const maxResponseBytes = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes || json.Unmarshal(body, &document) != nil || document.Data == nil {
		fmt.Fprintln(stderr, "Search failed: invalid or oversized JSON:API response.")
		return 1
	}
	for _, huddl := range *document.Data {
		_, dateErr := time.Parse(time.RFC3339, huddl.Attributes.StartsAt)
		if huddl.Type != "huddl" || strings.TrimSpace(huddl.ID) == "" || strings.TrimSpace(huddl.Attributes.Title) == "" || dateErr != nil {
			fmt.Fprintln(stderr, "Search failed: invalid or oversized JSON:API response.")
			return 1
		}
	}
	var output strings.Builder
	kind := options.kind
	if kind == "" {
		kind = "all types"
	}
	fmt.Fprintf(&output, "Searching everywhere · %s · %s · calendar timezone: %s · up to 20 results\n", options.date, kind, options.timeZone)
	fmt.Fprintln(&output)
	if len(*document.Data) == 0 {
		fmt.Fprintln(&output, "No matching huddlz.")
	} else {
		table := tabwriter.NewWriter(&output, 0, 4, 2, ' ', 0)
		fmt.Fprintln(table, "ID\tTITLE\tSTARTS AT\tLOCATION")
		for _, huddl := range *document.Data {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", tableCell(huddl.ID), tableCell(huddl.Attributes.Title), tableCell(huddl.Attributes.StartsAt), tableCell(huddl.Attributes.PhysicalLocation))
		}
		table.Flush()
	}
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
