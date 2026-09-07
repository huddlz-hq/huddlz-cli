package features_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

type searchScenario struct {
	server         *httptest.Server
	requests       chan *http.Request
	stdout, stderr bytes.Buffer
	exitCode       int
}

func TestFeatures(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "huddlz")
	build := exec.Command("go", "build", "-o", binary, "../cmd/huddlz")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	suite := godog.TestSuite{
		Name:    "huddlz",
		Options: &godog.Options{Format: "pretty", Paths: []string{"search.feature"}, TestingT: t, Strict: true},
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var state *searchScenario
			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				state = &searchScenario{requests: make(chan *http.Request, 10)}
				return ctx, nil
			})
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				if state.server != nil {
					state.server.Close()
				}
				return ctx, nil
			})
			sc.Step(`^the public API has upcoming board game huddlz$`, func() {
				state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					state.requests <- r.Clone(context.Background())
					w.Header().Set("Content-Type", "application/vnd.api+json")
					fmt.Fprint(w, `{"data":[{"type":"huddl","id":"11111111-1111-4111-8111-111111111111","attributes":{"title":"Board games at the library","starts_at":"2027-01-10T18:00:00Z","physical_location":"Central Library"}},{"type":"huddl","id":"22222222-2222-4222-8222-222222222222","attributes":{"title":"Sunday board games","starts_at":"2027-01-17T14:00:00Z","physical_location":"Community Center"}}]}`)
				}))
			})
			sc.Step(`^the public API has no matching huddlz$`, func() {
				state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					state.requests <- r.Clone(context.Background())
					w.Header().Set("Content-Type", "application/vnd.api+json")
					fmt.Fprint(w, `{"data":[]}`)
				}))
			})
			sc.Step(`^the public API fails with "([^"]*)"$`, func(failure string) error {
				status, body := http.StatusOK, ""
				switch failure {
				case "HTTP error":
					status, body = http.StatusServiceUnavailable, `{"errors":[{"detail":"Unavailable"}]}`
				case "disconnected":
				case "invalid JSON":
					body = "not JSON"
				case "missing data":
					body = `{}`
				case "null data":
					body = `{"data":null}`
				case "null resource":
					body = `{"data":[null]}`
				case "empty resource":
					body = `{"data":[{}]}`
				default:
					return fmt.Errorf("unknown API failure fixture: %q", failure)
				}
				state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if failure == "disconnected" {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err == nil {
							conn.Close()
						}
						return
					}
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(status)
					fmt.Fprint(w, body)
				}))
				return nil
			})
			runSearch := func(query string) error {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				args := []string{"search"}
				if query != "" {
					args = append(args, "--anywhere", query)
				}
				cmd := exec.CommandContext(ctx, binary, args...)
				// Isolate the subprocess from developer credentials and preferences.
				cmd.Env = []string{"HUDDLZ_URL=" + state.server.URL, "HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH")}
				cmd.Stdout, cmd.Stderr = &state.stdout, &state.stderr
				err := cmd.Run()
				if ctx.Err() != nil {
					return fmt.Errorf("CLI timed out: %w", ctx.Err())
				}
				if exit, ok := err.(*exec.ExitError); ok {
					state.exitCode = exit.ExitCode()
					return nil
				}
				return err
			}
			sc.Step(`^I search everywhere for "([^"]*)"$`, runSearch)
			sc.Step(`^I browse upcoming huddlz$`, func() error { return runSearch("") })
			sc.Step(`^the API receives an anonymous bounded upcoming search for "([^"]*)"$`, func(query string) error {
				select {
				case r := <-state.requests:
					if r.Method != "GET" || r.URL.Path != "/api/json/huddlz" {
						return fmt.Errorf("unexpected request: %s %s", r.Method, r.URL)
					}
					expected := map[string]string{"date_filter": "upcoming", "sort": "starts_at", "page[limit]": "20"}
					if query != "" {
						expected["query"] = query
					}
					for key, want := range expected {
						if got := r.URL.Query().Get(key); got != want {
							return fmt.Errorf("query %s: want %q, got %q", key, want, got)
						}
					}
					if len(r.URL.Query()) != len(expected) {
						return fmt.Errorf("unexpected search filters: %s", r.URL.RawQuery)
					}
					if r.Header.Get("Authorization") != "" {
						return fmt.Errorf("public search sent credentials")
					}
					if r.Header.Get("Accept") != "application/vnd.api+json" {
						return fmt.Errorf("request did not accept JSON:API")
					}
					return nil
				default:
					return fmt.Errorf("no API request received; exit=%d, stderr=%q", state.exitCode, state.stderr.String())
				}
			})
			sc.Step(`^I see the matching huddlz in a readable table$`, func() error {
				lines := strings.Split(strings.TrimSpace(state.stdout.String()), "\n")
				if len(lines) != 5 {
					return fmt.Errorf("expected scope, header, and two rows separated by a blank line; got:\n%s", state.stdout.String())
				}
				for i, cells := range map[int][]string{
					0: {"Searching everywhere"},
					2: {"ID", "TITLE", "STARTS AT", "LOCATION"},
					3: {"11111111-1111-4111-8111-111111111111", "Board games at the library", "2027-01-10T18:00:00Z", "Central Library"},
					4: {"22222222-2222-4222-8222-222222222222", "Sunday board games", "2027-01-17T14:00:00Z", "Community Center"},
				} {
					for _, cell := range cells {
						if !strings.Contains(lines[i], cell) {
							return fmt.Errorf("row %d missing %q: %s", i, cell, lines[i])
						}
					}
				}
				return nil
			})
			sc.Step(`^the command succeeds$`, func() error {
				if state.exitCode != 0 || state.stderr.Len() != 0 {
					return fmt.Errorf("exit=%d, stderr=%q", state.exitCode, state.stderr.String())
				}
				return nil
			})
			sc.Step(`^I see that no huddlz match$`, func() error {
				if !strings.Contains(state.stdout.String(), "Searching everywhere") || !strings.Contains(state.stdout.String(), "No matching huddlz.") {
					return fmt.Errorf("expected explicit empty search result, got %q", state.stdout.String())
				}
				return nil
			})
			sc.Step(`^the command fails with "([^"]*)"$`, func(message string) error {
				if state.exitCode != 1 || !strings.Contains(state.stderr.String(), message) {
					return fmt.Errorf("expected exit 1 and diagnostic %q; exit=%d, stderr=%q", message, state.exitCode, state.stderr.String())
				}
				return nil
			})
			sc.Step(`^no successful results are printed$`, func() error {
				if state.stdout.Len() != 0 {
					return fmt.Errorf("expected empty stdout on failure, got %q", state.stdout.String())
				}
				return nil
			})
		},
	}
	if suite.Run() != 0 {
		t.Fatal("Cucumber scenarios failed")
	}
}
