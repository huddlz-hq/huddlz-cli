package features_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

func registerDetails(sc *godog.ScenarioContext, current func() *searchScenario, binary, home string) {
	var undisclosed bool
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
		cmd.Env = []string{"HUDDLZ_URL=" + s.server.URL, "HOME=" + home, "PATH=" + os.Getenv("PATH")}
		cmd.Stdout, cmd.Stderr = &s.stdout, &s.stderr
		return cmd.Run()
	})
	sc.Step(`^I see its public details and availability limits$`, func() error {
		s := current()
		for _, value := range []string{"Board games", "Bring your favorite game.", "2027-01-10T18:00:00Z", "2027-01-10T20:00:00Z", "America/New_York", "hybrid", "Central Library", "Attendance limit: 12", "Seat availability: Not exposed by the API", "Virtual link: Not disclosed", "published"} {
			if !strings.Contains(s.stdout.String(), value) {
				return fmt.Errorf("missing %q in %s", value, s.stdout.String())
			}
		}
		if s.stderr.Len() != 0 {
			return fmt.Errorf("unexpected stderr: %s", s.stderr.String())
		}
		select {
		case r := <-s.requests:
			if r.Method != "GET" || r.URL.Path != "/api/json/huddlz/11111111-1111-4111-8111-111111111111" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "" || r.Header.Get("Accept") != "application/vnd.api+json" {
				return fmt.Errorf("unexpected request: %v", r)
			}
		default:
			return fmt.Errorf("no detail request")
		}
		return nil
	})
}
