package features_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

func registerPagination(sc *godog.ScenarioContext, current func() *searchScenario, binary, home string) {
	var query string
	var commands []string
	sc.Step(`^the public API has an empty final page$`, func() {
		state := current()
		state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			fmt.Fprint(w, `{"data":[],"links":{"next":null}}`)
		}))
	})
	sc.Step(`^the public API has two pages of matching huddlz$`, func() {
		state := current()
		state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state.requests <- r.Clone(context.Background())
			id, title := "first-id", "First matching huddl"
			var next any
			if r.URL.Query().Get("page[offset]") == "1" {
				id, title = "second-id", "Second matching huddl"
			} else {
				params := r.URL.Query()
				params.Set("page[offset]", "1")
				next = state.server.URL + "/api/json/huddlz?" + params.Encode()
			}
			w.Header().Set("Content-Type", "application/vnd.api+json")
			json.NewEncoder(w).Encode(map[string]any{
				"data":  []any{map[string]any{"type": "huddl", "id": id, "attributes": map[string]any{"title": title, "starts_at": "2027-01-10T18:00:00Z", "physical_location": "Library"}}},
				"links": map[string]any{"next": next},
			})
		}))
	})
	run := func(args []string, shell bool) error {
		state := current()
		state.stdout.Reset()
		state.stderr.Reset()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		program := binary
		if shell {
			program = "/bin/sh"
		}
		cmd := exec.CommandContext(ctx, program, args...)
		cmd.Env = []string{"HUDDLZ_URL=" + state.server.URL, "HOME=" + home, "PATH=" + filepath.Dir(binary) + string(os.PathListSeparator) + os.Getenv("PATH")}
		cmd.Stdout, cmd.Stderr = &state.stdout, &state.stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("CLI failed: %v; stderr=%s", err, state.stderr.String())
		}
		if state.stderr.Len() != 0 {
			return fmt.Errorf("unexpected stderr: %s", state.stderr.String())
		}
		return nil
	}
	sc.Step(`^I search the first page for "([^"]*)" with filters$`, func(input string) error {
		query = input
		return run([]string{"search", "--date", "this_week", "--type", "in_person", "--time-zone", "America/New_York", "--limit", "1", "--", query}, false)
	})
	sc.Step(`^I see the first matching huddl and a next-page command$`, func() error {
		output := current().stdout.String()
		if !strings.Contains(output, "First matching huddl") || strings.Contains(output, "Second matching huddl") {
			return fmt.Errorf("unexpected first page: %s", output)
		}
		commands = nil
		for _, line := range strings.Split(output, "\n") {
			if command, ok := strings.CutPrefix(line, "Next page: "); ok {
				commands = append(commands, command)
			}
		}
		if len(commands) != 1 {
			return fmt.Errorf("expected one next-page command: %s", output)
		}
		if len(current().requests) != 1 {
			return fmt.Errorf("expected exactly one request before continuing")
		}
		return nil
	})
	sc.Step(`^I run the suggested next-page command$`, func() error {
		// Execute exactly what the user would paste, including a query with shell metacharacters.
		return run([]string{"-c", commands[0]}, true)
	})
	sc.Step(`^I see the second matching huddl$`, func() error {
		output := current().stdout.String()
		if !strings.Contains(output, "Second matching huddl") || strings.Contains(output, "First matching huddl") {
			return fmt.Errorf("unexpected second page: %s", output)
		}
		return nil
	})
	sc.Step(`^both requests preserve the query and filters$`, func() error {
		state := current()
		if len(state.requests) != 2 {
			return fmt.Errorf("expected exactly two requests, got %d", len(state.requests))
		}
		for _, offset := range []string{"0", "1"} {
			expected := map[string]string{"query": query, "date_filter": "this_week", "event_type": "in_person", "search_time_zone": "America/New_York", "sort": "starts_at", "page[limit]": "1", "page[offset]": offset}
			if err := state.expectSearch(expected); err != nil {
				return err
			}
		}
		return nil
	})
	sc.Step(`^I am told there are no more results$`, func() error {
		output := current().stdout.String()
		if !strings.Contains(output, "No more results.") || strings.Contains(output, "Next page:") {
			return fmt.Errorf("unexpected final page status: %s", output)
		}
		return nil
	})
	sc.Step(`^the pagination status is unavailable$`, func() error {
		output := current().stdout.String()
		if !strings.Contains(output, "Pagination information unavailable.") || strings.Contains(output, "No more results.") || strings.Contains(output, "Next page:") {
			return fmt.Errorf("unexpected unknown pagination status: %s", output)
		}
		return nil
	})
}
