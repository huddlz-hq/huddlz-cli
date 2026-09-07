# huddlz CLI

A command-line client for humans and agents to discover huddlz and manage RSVPs.

This is the initial scaffold. Only help and version work; no backend requests
or credential storage are implemented yet.

## Development

Requires Go 1.26 or later. The scaffold uses only the standard library.

```sh
go run ./cmd/huddlz help
go run ./cmd/huddlz version --json
go build -o bin/huddlz ./cmd/huddlz
go test ./...
go vet ./...
```

For a versioned build:

```sh
go build -ldflags '-X main.version=0.1.0' -o bin/huddlz ./cmd/huddlz
```

The module path assumes the future repository will be
`github.com/huddlz-hq/huddlz-cli`; update it and imports if a different name is chosen.

## Initial direction

Go produces a standalone executable and provides HTTP, JSON, and testing in its
standard library. Keep the entry point small and command behavior in `internal/cli`.
Add API and credential packages when the first commands need them.

Planned commands (not implemented):

| Command | Purpose |
| --- | --- |
| `huddlz search [query]` | Discover public huddlz with filters and pagination |
| `huddlz show <id>` | Look up a single huddl |
| `huddlz rsvp <id>` | RSVP to a huddl |
| `huddlz rsvp cancel <id>` | Cancel an RSVP |
| `huddlz auth login` | Sign in as a human |
| `huddlz auth status` | Check the current identity |
| `huddlz auth logout` | End the local session and handle server revocation |

Command conventions:

- Readable text by default; explicit `--json` for structured output.
- Results on stdout and diagnostics on stderr.
- Exit code 0 for success, 1 for execution failure, and 2 for invalid usage.
- Agent commands must work without interactive prompts when inputs are supplied.
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
4. Cross-platform release builds and installation instructions once the GitHub
   repository exists.
