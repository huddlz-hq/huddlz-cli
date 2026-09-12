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

Address lookup remains planned.

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
| Address lookup | Choose search locations without coordinates |

Command conventions:

- Readable text by default; explicit `--json` for structured output.
- Results on stdout and diagnostics on stderr.
- Exit code 0 for success, 1 for execution failure, and 2 for invalid usage.
- All commands must work without interactive prompts when inputs are supplied.
- Anonymous discovery must work without login.
- Do not accept passwords through command-line arguments or print credentials.
  Environment-supplied API keys remain planned; password stdin is available to every caller.
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

The initial login uses password sign-in and saves a session token in a private
local file. The backend also supports API keys; a browser or device-code flow
would require checking for or adding backend support.
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
Virtual meeting links are shown only when the API discloses its protected
`visible_virtual_link` field to the current caller. The CLI identifies these limitations explicitly;
a published state or attendance limit does not promise an available seat.

A missing or inaccessible huddl produces the same “Huddl unavailable” diagnostic
on stderr and exits with status 1, leaving stdout empty. The CLI does not expose
API error bodies or try alternate lookups to determine whether a hidden huddl
exists. Server outages remain lookup errors rather than unavailable results.

### Structured search output

```sh
huddlz search "board games" --json
```

`--json` changes presentation only. Successful searches emit one JSON document
and no diagnostics on stderr. Failures leave stdout empty and report errors on
stderr with a nonzero exit status. Help remains readable text.

The document contains:

- `data`: an array (including `[]` for no matches) of resources with `id`, `type`,
  and `attributes` containing `title`, `starts_at`, and `physical_location`.
- `search`: `query`, `date`, `type` (empty string means all types), `time_zone`,
  and `location` (`null` means everywhere; otherwise `latitude`, `longitude`,
  and `radius_miles`).
- `pagination`: `limit`, `offset`, `known`, `next_offset`, and `next_command`.
  If `known` is false, the API supplied no pagination information. If true and
  `next_offset` is null, there are no more results. A next command preserves
  the filters and `--json`, using the same POSIX shell quoting as readable output.

Strings retain their original values in JSON; table formatting does not alter
structured data. This initial JSON convention covers search; `show` remains
readable output for now.

### Authenticate

```sh
huddlz auth login --email you@example.com
huddlz auth status
```

Login reads a hidden password from the terminal. For noninteractive use, add
`--password-stdin` and pipe the password from your secret manager. A trailing
line ending is removed; other password characters are preserved. The CLI never
accepts a password argument and never stores the password.

Login verifies the returned token with `/api/auth/me` before saving it. Status
also calls that endpoint, so it reports the current server-verified account,
including ID, email, and display name. Tokens are never printed.

Sessions are scoped to the full `HUDDLZ_URL` server and base path. Tokens are
stored unencrypted in mode-0600 files beneath the OS user configuration directory
(`~/Library/Application Support/huddlz/sessions` on macOS,
`$XDG_CONFIG_HOME/huddlz/sessions` or `~/.config/huddlz/sessions` on Linux).
The file name is a hash of the server URL. New session directories use mode 0700;
saves replace the token atomically. The OS keychain is not used in this version.
Authentication requires HTTPS except for loopback development servers and does
not follow redirects. Search and show use the saved session when present, as do profile defaults.
RSVP uses the saved
session for the selected server.

Use `huddlz auth logout` to remove the saved session for the selected server and
revoke its JWT through `DELETE /api/auth/sign_out`. Local removal happens first,
so subsequent commands stop using the token even if the server cannot be reached.
A confirmed revocation exits 0. If revocation cannot be confirmed (including a
rejected or expired token), logout exits 1 and explicitly reports that local
removal succeeded; it does not claim the server token was revoked. With no saved
session, logout succeeds without a network request. If local removal fails, the
command reports failure and does not claim to be signed out.

Logout affects only this server's saved session. API-key and environment-based
credentials are not supported yet; this command does not modify the caller's
environment or revoke API keys.

A rejected email/password reports a clear login failure on stderr, exits with
status 1, and leaves stdout empty. It creates no saved session and preserves any
existing session for that server. Diagnostics never include the server's error
body, password, or token.

If the server rejects a saved session with HTTP 401, `auth status` exits with
status 1 and explains how to log in again. It prints no account data, exposes no
token, and never opens a login prompt automatically. The server may reject an
expired or revoked token; the CLI does not claim to distinguish those causes.
Other failures, such as HTTP 403 or 503, remain account-verification errors.
Failed verification leaves the saved token unchanged.

### RSVP

```sh
huddlz rsvp <id>
```

Use a huddl ID from search after logging in. The CLI sends one authenticated
`PATCH /api/json/huddlz/<id>/rsvp` and reports the returned attendance state.
For older responses without that field, it verifies the exact huddl through
`relationship=attending` and `date_filter=all`. A successful mutation alone does
not establish attendance: a repeated RSVP may leave an existing waitlist entry.

Backend rejections remain failures. If submission succeeds but verification fails
or finds no confirmed attendance, the CLI reports uncertainty and exits 1. It
does not automatically repeat the mutation or join a waitlist. Missing or rejected
credentials fail without an interactive login prompt. Use `huddlz rsvp waitlist <id>` to explicitly request a waitlist entry.

### List my RSVPs

```sh
huddlz rsvp list
huddlz rsvp list --status waitlisted --json
```

Lists currently visible huddlz across all dates, separated into `confirmed` and
`waitlisted` groups. Each row includes the huddl ID, title, start timestamp, and
physical location. The API applies the current user's membership filters;
confirmed attendance excludes waitlisted entries.

`--status` accepts `all` (default), `confirmed`, or `waitlisted`. `--limit` is
1–100, default 20 **per selected status**, and `--offset` defaults to 0 for each
selected status. Each group is sorted by start time and has its own next-page
command. An `all` request makes two bounded reads; a status-specific request
makes one. The two reads are not a single snapshot, so membership may change
between them. An error in either read leaves stdout empty.

JSON contains `groups`, each with `status`, `data` (the same huddl resource shape
as search), and `pagination` (`known`, `limit`, `offset`, `next_offset`, and
`next_command`). Continuation commands select just that group's status and
preserve `--json`. Missing pagination metadata is distinguished from the last page.

### Cancel an RSVP or leave a waitlist

```sh
huddlz rsvp cancel <id>
```

Uses the saved session to submit one `PATCH /api/json/huddlz/<id>/cancel_rsvp`.
The backend removes either confirmed attendance or an existing waitlist entry.
The CLI validates the mutation response, then checks the exact huddl ID against
both `attending` and `waitlisted` membership. It reports success only after both
checks return empty results. These checks reflect the API's current visibility
and are separate reads, not a transactional snapshot.

A rejected mutation, remaining membership, invalid response, or verification
failure exits nonzero without claiming cancellation succeeded. When submission
succeeded but verification did not, the diagnostic states that distinction. The
CLI never automatically repeats the mutation. No new waitlist-join API is needed
to leave an existing waitlist entry.

Signed-in searches now fetch current home search defaults from `/api/json/profile`
when no explicit coordinates or `--anywhere` are supplied. The returned coordinates,
radius, and location timezone are used for that request; an explicit `--time-zone`
continues to take precedence. No separate home-location copy is persisted locally.

### Join a waitlist

`huddlz rsvp waitlist <id>` submits one request to the full huddl's waitlist API.
It reports the returned `attendance_state` (`waitlisted` or `confirmed`), without
inventing a queue position or remaining-seat count. Normal RSVP now also honors
this field, including a repeated RSVP by someone already waitlisted. An unknown
or absent waitlist outcome fails. The CLI does not infer a full-huddl error or
silently switch a normal RSVP into a waitlist request.

The newer API returns attendance state directly. For ordinary RSVP responses
from older servers that omit it, the CLI retains its attending-only verification.


Discovery uses a saved session when available and stays anonymous otherwise.
A rejected or unreadable saved session is a failure, never an anonymous retry.
`show` requests the hosting group's name and slug and the caller-visible virtual
link. Missing or protected details remain marked as undisclosed. The API owns
visibility decisions; the CLI does not attempt alternate lookups.
