# huddlz CLI

A command-line client for humans and agents to discover huddlz and manage RSVPs.

This is an early CLI with help, version, and public search. Credentials
and profile preferences are not implemented yet.

```sh
huddlz search
huddlz search --anywhere "board games"
huddlz search "board games" --date this_week --type in_person --time-zone America/New_York
```

Search requests the first 20 matches, ordered by start time ascending, and prints
a table. It defaults to upcoming huddlz and uses `https://huddlz.com`; set
`HUDDLZ_URL` to use another server.
Omit the query to browse. Anonymous search is unrestricted geographically and
prints that scope explicitly. Options may appear before or after the quoted
query. `--anywhere` also explicitly selects that geographic scope.

| Option | Accepted values | Default |
| --- | --- | --- |
| `--date` | `upcoming`, `this_week`, `this_month`, `past`, `all` | `upcoming` |
| `--type` | `in_person`, `virtual`, `hybrid` | All types when omitted |
| `--time-zone` | Canonical IANA name, such as `America/New_York` or `Etc/UTC` | `Etc/UTC` |

The chosen calendar timezone is sent to the server and shown in the search
scope. `UTC` is accepted as shorthand for `Etc/UTC`. The API determines calendar
boundaries and applies all filters together; the CLI does not filter a downloaded
page locally. Result timestamps retain the offsets supplied by the API. Invalid
date/type values and noncanonical timezones fail before a request is sent.
Aliases such as `US/Eastern` are rejected; use `America/New_York` instead.

Search JSON output, geographic filters, and subsequent pages remain planned.

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
| Search filters and pagination | Refine discovery and fetch subsequent pages |
| `huddlz show <id>` | Look up a single huddl |
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
