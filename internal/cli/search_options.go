package cli

import (
	_ "embed"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
)

//go:embed timezones.txt
var canonicalTimeZones string

const searchHelp = `Usage: huddlz search [<query>] [options]

Options:
  --anywhere          Search without a geographic restriction (current default)
  --date <filter>     upcoming (default), this_week, this_month, past, all
  --type <type>       in_person, virtual, hybrid (omitted: all types)
  --time-zone <zone>  Canonical IANA calendar timezone (default: Etc/UTC)
  -h, --help          Show search help

Options may appear before or after the quoted query.
The server applies date filters in the selected calendar timezone.
Result timestamps retain the offsets returned by the server.
`

type searchOptions struct {
	query, date, kind, timeZone string
}

func parseSearchOptions(args []string) (searchOptions, error) {
	var options searchOptions
	flags := pflag.NewFlagSet("search", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Bool("anywhere", false, "Search without a geographic restriction")
	flags.StringVar(&options.date, "date", "upcoming", "Date filter")
	flags.StringVar(&options.kind, "type", "", "Huddl type")
	flags.StringVar(&options.timeZone, "time-zone", "Etc/UTC", "Calendar timezone")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() > 1 || flags.NArg() == 1 && strings.TrimSpace(flags.Arg(0)) == "" {
		return options, fmt.Errorf("supply at most one nonempty query; quote multiword searches")
	}
	options.query = flags.Arg(0)
	switch options.date {
	case "upcoming", "this_week", "this_month", "past", "all":
	default:
		return options, fmt.Errorf("--date must be upcoming, this_week, this_month, past, or all")
	}
	if flags.Changed("type") {
		switch options.kind {
		case "in_person", "virtual", "hybrid":
		default:
			return options, fmt.Errorf("--type must be in_person, virtual, or hybrid; omit it for all types")
		}
	}
	if options.timeZone == "UTC" {
		options.timeZone = "Etc/UTC"
	}
	if !isCanonicalTimeZone(options.timeZone) {
		return options, fmt.Errorf("--time-zone must be a canonical IANA timezone such as America/New_York or Etc/UTC; timezone aliases are not supported")
	}
	return options, nil
}

func isCanonicalTimeZone(name string) bool {
	for _, zone := range strings.Split(canonicalTimeZones, "\n") {
		if zone != "" && !strings.HasPrefix(zone, "#") && zone == name {
			return true
		}
	}
	return false
}
