package features_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

func registerDetails(sc *godog.ScenarioContext, current func() *searchScenario, binary, home string) {
	var undisclosed bool
	sc.Step(`^the huddl lookup returns HTTP (\d+)$`, func(status int) {
		s := current()
		s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.requests <- r.Clone(context.Background())
			w.WriteHeader(status)
			fmt.Fprint(w, `{"errors":[{"detail":"Protected title and private address"}]}`)
		}))
	})
	sc.Step(`^the huddl is reported unavailable without disclosing details$`, func() error {
		s := current()
		if s.exitCode != 1 || s.stdout.Len() != 0 || s.stderr.String() != "Huddl unavailable: it does not exist or is not visible to you.\n" {
			return fmt.Errorf("expected unavailable result, got exit=%d stdout=%q stderr=%q", s.exitCode, s.stdout.String(), s.stderr.String())
		}
		return nil
	})
	sc.Step(`^no alternate huddl lookup is attempted$`, func() error {
		s := current()
		if len(s.requests) != 1 {
			return fmt.Errorf("expected one lookup, got %d", len(s.requests))
		}
		r := <-s.requests
		if r.Method != "GET" || r.URL.Path != "/api/json/huddlz/11111111-1111-4111-8111-111111111111" || r.URL.Query().Get("include") != "group" {
			return fmt.Errorf("unexpected lookup: %v", r)
		}
		return nil
	})
	sc.Step(`^its optional details are undisclosed$`, func() { undisclosed = true })
	sc.Step(`^missing details are reported honestly$`, func() error {
		s := current()
		for _, v := range []string{"Description: Not disclosed", "Location: Not disclosed", "Attendance limit: Not specified", "Seat availability: Not exposed by the API"} {
			if !strings.Contains(s.stdout.String(), v) {
				return fmt.Errorf("missing %q in %s", v, s.stdout.String())
			}
		}
		return nil
	})
	sc.Step(`^a public huddl has published details$`, func() {
		s := current()
		undisclosed = false
		s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.requests <- r.Clone(context.Background())
			body := `{"data":{"type":"huddl","id":"11111111-1111-4111-8111-111111111111","attributes":{"title":"Board games","description":"Bring your favorite game.","starts_at":"2027-01-10T18:00:00Z","ends_at":"2027-01-10T20:00:00Z","time_zone":"America/New_York","event_type":"hybrid","physical_location":"Central Library","max_attendees":12,"lifecycle_state":"published"}}}`
			if undisclosed {
				var doc map[string]any
				json.Unmarshal([]byte(body), &doc)
				a := doc["data"].(map[string]any)["attributes"].(map[string]any)
				for _, key := range []string{"description", "physical_location", "max_attendees"} {
					a[key] = nil
				}
				json.NewEncoder(w).Encode(doc)
			} else {
				fmt.Fprint(w, body)
			}
		}))
	})
	sc.Step(`^I look up the huddl by its identifier$`, func() error {
		s := current()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "show", "11111111-1111-4111-8111-111111111111")
		cmd.Env = testEnvironment(s.server.URL, home)
		cmd.Stdout, cmd.Stderr = &s.stdout, &s.stderr
		err := cmd.Run()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			s.exitCode = exitErr.ExitCode()
			return nil
		}
		return err
	})
	sc.Step(`^I see its public details and availability limits$`, func() error {
		s := current()
		for _, value := range []string{"Board games", "Bring your favorite game.", "2027-01-10T18:00:00Z", "2027-01-10T20:00:00Z", "America/New_York", "hybrid", "Central Library", "Attendance limit: 12", "Seat availability: Not exposed by the API", "Virtual link: Not disclosed", "published"} {
			if !strings.Contains(s.stdout.String(), value) {
				return fmt.Errorf("missing %q in %s", value, s.stdout.String())
			}
		}
		if s.exitCode != 0 || s.stderr.Len() != 0 {
			return fmt.Errorf("unexpected stderr: %s", s.stderr.String())
		}
		select {
		case r := <-s.requests:
			if r.Method != "GET" || r.URL.Path != "/api/json/huddlz/11111111-1111-4111-8111-111111111111" || r.URL.Query().Get("include") != "group" || r.Header.Get("Authorization") != "" || r.Header.Get("Accept") != "application/vnd.api+json" {
				return fmt.Errorf("unexpected request: %v", r)
			}
		default:
			return fmt.Errorf("no detail request")
		}
		return nil
	})
}
