# huddlz CLI

A command-line client for humans and agents to discover huddlz and manage RSVPs.

This is an early CLI with help, version, and public search. Credentials
and profile preferences are not implemented yet.

```sh
huddlz search
huddlz search --anywhere "board games"
huddlz search "board games" --date this_week --type in_person --time-zone America/New_York
huddlz search "board games" --limit 10 --offset 10
huddlz search "board games" --lat 40.7128 --lng -74.006 --radius 25
```

Search requests one page of up to 20 matches by default, ordered by start time
ascending, and prints a table. It defaults to upcoming huddlz and uses `https://huddlz.com`; set
`HUDDLZ_URL` to use another server.
Omit the query to browse. Without coordinates, anonymous search is unrestricted
geographically and prints that scope explicitly. Options may appear before or
after the quoted query. `--anywhere` explicitly selects unrestricted search.
Provide `--lat` and `--lng` together to search near a chosen point; coordinates
of zero are valid. The radius defaults to 25 miles. This applies only to the
current search and does not save a location or change profile preferences.
Incomplete coordinate pairs, non-finite or out-of-range coordinates, and invalid
radii fail before any API request. `--radius` requires coordinates, and
`--anywhere` cannot be combined with coordinates or a radius.

| Option | Accepted values | Default |
| --- | --- | --- |
| `--date` | `upcoming`, `this_week`, `this_month`, `past`, `all` | `upcoming` |
| `--type` | `in_person`, `virtual`, `hybrid` | All types when omitted |
| `--time-zone` | Canonical IANA name, such as `America/New_York` or `Etc/UTC` | `Etc/UTC` |
| `--limit` | Integer from 1 through 100 | `20` |
| `--offset` | Nonnegative integer | `0` |
| `--lat` | Latitude from -90 through 90; requires `--lng` | None |
| `--lng` | Longitude from -180 through 180; requires `--lat` | None |
| `--radius` | Integer from 5 through 100 miles | `25` with coordinates |

The chosen calendar timezone is sent to the server and shown in the search
scope. `UTC` is accepted as shorthand for `Etc/UTC`. The API determines calendar
boundaries and applies all filters together; the CLI does not filter a downloaded
page locally. Result timestamps retain the offsets supplied by the API. Invalid
date/type values and noncanonical timezones fail before a request is sent.
Aliases such as `US/Eastern` are rejected; use `America/New_York` instead.

When the API reports another page, output includes a `Next page:` command with
the current query, filters, coordinates, radius, and page size preserved. Copy it into a POSIX shell
(such as bash or zsh) while retaining the same `HUDDLZ_URL` environment setting.
Each invocation fetches just one page. You can also repeat the search with an
explicit `--offset`; it counts results to skip, not page numbers.

An explicit null next-page link produces `No more results.` Missing pagination
metadata produces `Pagination information unavailable.`; the CLI does not guess
from the number of results. Invalid continuation links and responses exceeding
the requested limit fail without printing successful results. Offset pagination
is not a snapshot: changes to matching huddlz between requests can move results.

Search JSON output, address lookup, and profile-based defaults remain planned.

## Development

Requires Go 1.26 or later. The CLI uses pflag for option parsing and Godog for
acceptance tests. Canonical IANA timezone names are bundled in the executable;
calendar calculations remain on the server. The name list matches IANA tzdata
2026c and can be refreshed with `sh scripts/update-timezones.sh <release>` when
the backend adopts a newer release. The script requires curl, tar, awk, and sort.

```sh
go run ./cmd/huddlz help
go run ./cmd/huddlz version --json
go build -o bin/huddlz ./cmd/huddlz
go test ./...
go vet ./...
```

## Acceptance scenarios

Run the executable search scenarios with:

```sh
go test ./features -run TestFeatures -v -count=1
```

`features/pagination.feature` follows the printed next-page command through a
POSIX shell and checks that filters and literal query text survive. It also
covers page bounds and end-of-results states.

`features/location.feature` checks explicit coordinate search, valid zero and
boundary coordinates, and preservation of geographic filters across pages.
It also checks that invalid or contradictory location inputs never reach the API.

`features/search.feature` covers anonymous browsing, search by interest, combined
date/type filters, invalid inputs, no matches, and API failures. Godog builds and runs the actual CLI against an
isolated local HTTP server. Scenarios check query encoding, upcoming ordering,
the page limit, absence of credentials, readable results, stderr, and exit
status. HTTP errors, disconnected responses, and malformed JSON:API responses
are distinct from successful empty results. Tests require permission to listen
on loopback, but do not contact production or require a huddlz account.

The fixture represents the JSON:API contract; it does not prove the backend's
search/filtering behavior. Search and browse scenarios were observed failing
before implementation and passing afterward. The empty and failure scenarios
exercise handling already present in the initial search slice.

Godog runs through `go test` and reports failures by Gherkin step. Step registration,
scenario state, and assertions are explicit Go code. No assertion library is
required, but one can be added if it improves diagnostics and readability.

For a versioned build:

```sh
go build -ldflags '-X main.version=0.1.0' -o bin/huddlz ./cmd/huddlz
```

Scenario tickets and dependencies are tracked in
[GitHub Issues](https://github.com/huddlz-hq/huddlz-cli/issues).

## Initial direction

Go produces a standalone executable and provides HTTP, JSON, and testing in its
standard library. Keep the entry point small and command behavior in `internal/cli`.
Add API and credential packages when the first commands need them.

Planned additions (not implemented):

| Command | Purpose |
| --- | --- |
| Address lookup and profile defaults | Choose search locations without coordinates |
| `huddlz rsvp <id>` | RSVP to a huddl |
| `huddlz rsvp cancel <id>` | Cancel an RSVP |
| `huddlz auth login` | Authenticate |
| `huddlz auth status` | Check the current identity |
| `huddlz auth logout` | End the local session and handle server revocation |

Command conventions:

- Readable text by default; explicit `--json` for structured output.
- Results on stdout and diagnostics on stderr.
- Exit code 0 for success, 1 for execution failure, and 2 for invalid usage.
- All commands must work without interactive prompts when inputs are supplied.
- Anonymous discovery must work without login.
- Support an environment-supplied API key for agents. Do not accept passwords
  through command-line arguments or print credentials.
- Keep pagination explicit so agents can bound requests and output.

## Existing API

The adjacent huddlz backend was inspected during initialization. These routes
exist in that checkout; deployment availability still needs verification.

| Capability | Route |
| --- | --- |
| Search | `GET /api/json/huddlz` |
| Huddl details | `GET /api/json/huddlz/:id` |
| RSVP | `PATCH /api/json/huddlz/:id/rsvp` |
| Cancel RSVP | `PATCH /api/json/huddlz/:id/cancel_rsvp` |
| Password sign-in | `POST /api/auth/sign_in` |
| Current identity | `GET /api/auth/me` |
| JWT sign-out | `DELETE /api/auth/sign_out` |
| API keys | `POST /api/auth/api_keys`, `GET /api/auth/api_keys`, `DELETE /api/auth/api_keys/:id` |

Search uses JSON:API sorting such as `sort=starts_at` or `sort=-inserted_at`.
The backend advertises its schema at `/api/json/open_api`.

Before implementing authentication, decide the human login flow and credential
storage. The inspected backend supports password sign-in and API keys; a browser
or device-code flow would require checking for or adding backend support.
JWT sign-out and API-key revocation are separate operations. Current API keys
have full user permissions, so scoped agent keys are a backend follow-up.

## Next increments

1. Read-only search and details, with filters, pagination, text/JSON output,
   and HTTP integration tests against a local test server.
2. Authentication and identity, with an explicit backend URL and credentials
   tied to that origin.
3. RSVP and cancellation, including full-capacity and waitlist behavior.
4. Cross-platform release builds and installation instructions.

### Inspect a huddl

```sh
huddlz show <id>
```

Use an ID from search to read a public huddl's description, start and end times,
calendar time zone, event type, location, lifecycle state, and attendance limit.
Timestamps retain the API's offsets. Missing optional details are marked as not
disclosed; a missing attendance limit is shown as not specified.

The current JSON:API detail schema does not expose RSVP counts or remaining seats.
Virtual meeting links are undisclosed to this anonymous lookup (the API has a
protected `visible_virtual_link` field for authorized users). The CLI identifies these limitations explicitly;
a published state or attendance limit does not promise an available seat.
