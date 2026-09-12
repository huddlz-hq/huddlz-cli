package features_test

import (
	"context"
	"encoding/json"
	"errors"
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

func registerAuth(sc *godog.ScenarioContext, current func() *searchScenario, binary, root string) {
	var home string
	var serverOverride string
	var redirectLogin bool
	var accountStatus int
	var revocationFailure string
	var rsvpStatus int
	var attendanceConfirmed bool
	var listMode, nextListCommand string
	var cancellationMembership string
	var cancelled bool
	var cancelMode string
	var profileMode string
	var waitlistState string
	sc.Step(`^the authentication API accepts my credentials$`, func() error {
		var err error
		home, err = os.MkdirTemp(root, "account-")
		if err != nil {
			return err
		}
		s := current()
		serverOverride = ""
		redirectLogin = false
		accountStatus = http.StatusOK
		revocationFailure = ""
		rsvpStatus = http.StatusOK
		attendanceConfirmed = true
		listMode = ""
		cancellationMembership = ""
		cancelled = false
		cancelMode = ""
		profileMode = ""
		waitlistState = "waitlisted"
		s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.requests <- r.Clone(context.Background())
			switch r.URL.Path {
			case "/api/json/huddlz/11111111-1111-4111-8111-111111111111/join_waitlist":
				var body struct{ Data struct{ ID, Type string } }
				if r.Method != "PATCH" || r.Header.Get("Authorization") != "Bearer test-session-secret" || json.NewDecoder(r.Body).Decode(&body) != nil || body.Data.ID != "11111111-1111-4111-8111-111111111111" {
					w.WriteHeader(400)
					return
				}
				fmt.Fprintf(w, `{"data":{"id":"11111111-1111-4111-8111-111111111111","type":"huddl","attributes":{"attendance_state":%q}}}`, waitlistState)
			case "/api/json/profile":
				if profileMode == "unavailable" {
					w.WriteHeader(503)
					return
				}
				if profileMode == "unset" || profileMode == "incomplete" {
					fmt.Fprint(w, `{"search_defaults":{"home_location":null,"distance_miles":25}}`)
					return
				}
				if r.Header.Get("Authorization") != "Bearer test-session-secret" {
					w.WriteHeader(401)
					return
				}
				fmt.Fprint(w, `{"search_defaults":{"home_location":{"label":"Austin","latitude":30.2672,"longitude":-97.7431,"time_zone":"America/Chicago"},"distance_miles":25}}`)
			case "/api/auth/sign_in":
				if redirectLogin {
					w.Header().Set("Location", s.server.URL+"/redirect-destination")
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				var credentials struct{ Email, Password string }
				if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&credentials) != nil || credentials.Email != "person@example.com" || credentials.Password != "test-password" {
					w.WriteHeader(401)
					fmt.Fprint(w, `{"error":"wrong-password test-session-secret"}`)
					return
				}
				fmt.Fprint(w, `{"token":"test-session-secret","user":{"id":"account-id","email":"person@example.com","display_name":"Test Person"}}`)
			case "/api/json/huddlz/11111111-1111-4111-8111-111111111111/cancel_rsvp":
				var body struct{ Data struct{ ID, Type string } }
				if r.Method != "PATCH" || r.Header.Get("Authorization") != "Bearer test-session-secret" || r.Header.Get("Content-Type") != "application/vnd.api+json" || json.NewDecoder(r.Body).Decode(&body) != nil || body.Data.ID != "11111111-1111-4111-8111-111111111111" || body.Data.Type != "huddl" || cancellationMembership == "" {
					w.WriteHeader(400)
					return
				}
				if cancelMode == "rejected" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{"errors":[{"detail":"test-session-secret"}]}`)
					return
				}
				cancelled = cancelMode != "retained"
				fmt.Fprint(w, `{"data":{"type":"huddl","id":"11111111-1111-4111-8111-111111111111"}}`)
			case "/api/json/huddlz/11111111-1111-4111-8111-111111111111/rsvp":
				if rsvpStatus != http.StatusOK {
					w.WriteHeader(rsvpStatus)
					fmt.Fprint(w, `{"errors":[{"detail":"test-session-secret"}]}`)
					return
				}
				var body struct {
					Data struct {
						ID   string
						Type string
					}
				}
				if r.Method != "PATCH" || r.Header.Get("Authorization") != "Bearer test-session-secret" || r.Header.Get("Content-Type") != "application/vnd.api+json" || json.NewDecoder(r.Body).Decode(&body) != nil || body.Data.ID != "11111111-1111-4111-8111-111111111111" || body.Data.Type != "huddl" {
					w.WriteHeader(400)
					return
				}
				fmt.Fprint(w, `{"data":{"type":"huddl","id":"11111111-1111-4111-8111-111111111111"}}`)
			case "/api/json/huddlz":
				q := r.URL.Query()
				if q.Get("search_latitude") != "" || q.Get("relationship") == "" {
					fmt.Fprint(w, `{"data":[],"links":{"next":null}}`)
					return
				}

				if cancellationMembership != "" && q.Get("filter[id][eq]") != "" {
					if r.Header.Get("Authorization") != "Bearer test-session-secret" || r.Method != "GET" || q.Get("filter[id][eq]") != "11111111-1111-4111-8111-111111111111" || q.Get("date_filter") != "all" || (q.Get("relationship") != "attending" && q.Get("relationship") != "waitlisted") {
						w.WriteHeader(400)
						return
					}
					if cancelMode == "unavailable" && q.Get("relationship") == "waitlisted" {
						w.WriteHeader(503)
						return
					}
					if cancelled || q.Get("relationship") != cancellationMembership {
						fmt.Fprint(w, `{"data":[]}`)
					} else {
						fmt.Fprint(w, `{"data":[{"type":"huddl","id":"11111111-1111-4111-8111-111111111111"}]}`)
					}
					return
				}

				if q.Get("filter[id][eq]") == "" {
					if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-session-secret" || q.Get("date_filter") != "all" || q.Get("sort") != "starts_at" || q.Get("page[limit]") != "20" || (q.Get("page[offset]") != "0" && q.Get("page[offset]") != "1") {
						w.WriteHeader(400)
						return
					}
					title, id := "Confirmed games", "11111111-1111-4111-8111-111111111111"
					if q.Get("relationship") == "waitlisted" {
						title, id = "Waitlisted games", "22222222-2222-4222-8222-222222222222"
					} else if q.Get("relationship") != "attending" {
						w.WriteHeader(400)
						return
					}

					if listMode == "failed" && q.Get("relationship") == "waitlisted" {
						w.WriteHeader(503)
						return
					}
					if listMode == "empty" {
						fmt.Fprint(w, `{"data":[],"links":{"next":null}}`)
						return
					}
					next := "null"
					if listMode == "next" && q.Get("relationship") == "waitlisted" && q.Get("page[offset]") == "0" {
						next = `"?page[offset]=1"`
					}
					fmt.Fprintf(w, `{"data":[{"type":"huddl","id":%q,"attributes":{"title":%q,"starts_at":"2027-01-10T18:00:00Z","physical_location":"Central Library"}}],"links":{"next":%s}}`, id, title, next)

					return
				}

				if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-session-secret" || r.Header.Get("Accept") != "application/vnd.api+json" || q.Get("filter[id][eq]") != "11111111-1111-4111-8111-111111111111" || q.Get("relationship") != "attending" || q.Get("date_filter") != "all" || q.Get("page[limit]") != "1" {
					w.WriteHeader(400)
					return
				}
				if !attendanceConfirmed {
					fmt.Fprint(w, `{"data":[]}`)
					return
				}
				fmt.Fprint(w, `{"data":[{"type":"huddl","id":"11111111-1111-4111-8111-111111111111"}]}`)
			case "/api/auth/sign_out":
				switch revocationFailure {
				case "expired":
					w.WriteHeader(401)
					fmt.Fprint(w, `{"error":"test-session-secret"}`)
					return
				case "unavailable":
					w.WriteHeader(503)
					return
				case "disconnected":
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						conn.Close()
					}
					return
				}

				if r.Method != "DELETE" || r.Header.Get("Authorization") != "Bearer test-session-secret" {
					w.WriteHeader(401)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			case "/api/auth/me":
				if accountStatus != http.StatusOK {
					w.WriteHeader(accountStatus)
					fmt.Fprint(w, `{"error":"test-session-secret test-password"}`)
					return
				}
				if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-session-secret" {
					w.WriteHeader(401)
					return
				}
				fmt.Fprint(w, `{"user":{"id":"account-id","email":"person@example.com","display_name":"Test Person"}}`)
			default:
				w.WriteHeader(404)
			}
		}))
		return nil
	})
	runCLI := func(args []string, input string) error {
		s := current()
		s.stdout.Reset()
		s.stderr.Reset()
		s.exitCode = 0
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		server := s.server.URL
		if serverOverride != "" {
			server = serverOverride
		}
		cmd.Env = []string{"HUDDLZ_URL=" + server, "HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, "config"), "PATH=" + os.Getenv("PATH")}
		cmd.Stdin = strings.NewReader(input)
		cmd.Stdout = &s.stdout
		cmd.Stderr = &s.stderr
		err := cmd.Run()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			s.exitCode = exitErr.ExitCode()
			return nil
		}
		return err
	}
	run := func(args []string, input string) error { return runCLI(append([]string{"auth"}, args...), input) }
	sc.Step(`^the RSVP endpoint returns HTTP (\d+)$`, func(status int) { rsvpStatus = status })
	sc.Step(`^the server cannot confirm my attendance$`, func() { attendanceConfirmed = false })
	sc.Step(`^only one RSVP attempt was made$`, func() error {
		s := current()
		if len(s.requests) != 1 {
			return fmt.Errorf("expected one RSVP request, got %d", len(s.requests))
		}
		r := <-s.requests
		if r.Method != "PATCH" || strings.Contains(s.stderr.String(), "test-session-secret") {
			return fmt.Errorf("unexpected RSVP failure")
		}
		return nil
	})
	sc.Step(`^the RSVP was submitted only once before checking attendance$`, func() error {
		s := current()
		if len(s.requests) != 2 {
			return fmt.Errorf("expected RSVP then confirmation")
		}
		for _, method := range []string{"PATCH", "GET"} {
			r := <-s.requests
			if r.Method != method {
				return fmt.Errorf("unexpected request sequence")
			}
		}
		return nil
	})

	sc.Step(`^my RSVP list has another waitlisted page$`, func() { listMode = "next" })
	sc.Step(`^my RSVP lists are empty$`, func() { listMode = "empty" })
	sc.Step(`^the waitlisted lookup fails$`, func() { listMode = "failed" })
	sc.Step(`^I list my RSVPs as JSON$`, func() error { return runCLI([]string{"rsvp", "list", "--json"}, "") })
	sc.Step(`^JSON distinguishes membership and provides a waitlisted continuation$`, func() error {
		var out struct {
			Groups []struct {
				Status string
				Data   []struct {
					ID         string
					Attributes struct {
						Title    string
						StartsAt string `json:"starts_at"`
					}
				}
				Pagination struct {
					Known       bool
					NextCommand *string `json:"next_command"`
				}
			}
		}
		if err := json.Unmarshal(current().stdout.Bytes(), &out); err != nil {
			return err
		}
		if len(out.Groups) != 2 || out.Groups[0].Status != "confirmed" || out.Groups[1].Status != "waitlisted" || len(out.Groups[0].Data) != 1 || len(out.Groups[1].Data) != 1 || out.Groups[0].Data[0].Attributes.Title != "Confirmed games" || out.Groups[1].Data[0].Attributes.StartsAt != "2027-01-10T18:00:00Z" || out.Groups[0].Pagination.NextCommand != nil || out.Groups[1].Pagination.NextCommand == nil {
			return fmt.Errorf("unexpected membership JSON")
		}
		nextListCommand = *out.Groups[1].Pagination.NextCommand
		if nextListCommand != "huddlz rsvp list --status waitlisted --limit 20 --offset 1 --json" {
			return fmt.Errorf("incorrect continuation")
		}
		return nil
	})
	sc.Step(`^I run the RSVP continuation command$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI(strings.Fields(nextListCommand)[1:], "")
	})
	sc.Step(`^the continuation lists only the next waitlisted page$`, func() error {
		var out struct {
			Groups []struct {
				Status     string
				Pagination struct {
					Offset      int
					Known       bool
					NextCommand *string `json:"next_command"`
				}
			}
		}
		s := current()
		if err := json.Unmarshal(s.stdout.Bytes(), &out); err != nil {
			return err
		}
		if len(out.Groups) != 1 || out.Groups[0].Status != "waitlisted" || out.Groups[0].Pagination.Offset != 1 || !out.Groups[0].Pagination.Known || out.Groups[0].Pagination.NextCommand != nil || len(s.requests) != 1 {
			return fmt.Errorf("unexpected continuation result")
		}
		r := <-s.requests
		if r.URL.Query().Get("relationship") != "waitlisted" || r.URL.Query().Get("page[offset]") != "1" {
			return fmt.Errorf("incorrect continuation request")
		}
		return nil
	})
	sc.Step(`^JSON contains empty membership groups$`, func() error {
		var out struct {
			Groups []struct{ Data json.RawMessage }
		}
		if err := json.Unmarshal(current().stdout.Bytes(), &out); err != nil {
			return err
		}
		if len(out.Groups) != 2 {
			return fmt.Errorf("expected two membership groups")
		}
		for _, g := range out.Groups {
			if string(g.Data) != "[]" {
				return fmt.Errorf("expected empty array")
			}
		}
		return nil
	})

	sc.Step(`^I list my RSVPs$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI([]string{"rsvp", "list"}, "")
	})
	sc.Step(`^my RSVP list identifies confirmed and waitlisted huddlz$`, func() error {
		s := current()
		for _, value := range []string{"confirmed", "waitlisted", "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "Confirmed games", "Waitlisted games", "2027-01-10T18:00:00Z", "Central Library"} {
			if !strings.Contains(s.stdout.String(), value) {
				return fmt.Errorf("missing %q in list", value)
			}
		}
		if len(s.requests) != 2 {
			return fmt.Errorf("expected two membership searches")
		}
		return nil
	})

	sc.Step(`^cancellation is rejected$`, func() { cancelMode = "rejected" })
	sc.Step(`^cancellation leaves membership unchanged$`, func() { cancelMode = "retained" })
	sc.Step(`^cancellation verification is unavailable$`, func() { cancelMode = "unavailable" })
	sc.Step(`^cancellation was attempted only once$`, func() error {
		s := current()
		if len(s.requests) != 1 || strings.Contains(s.stderr.String(), "test-session-secret") {
			return fmt.Errorf("expected safe single cancellation attempt")
		}
		r := <-s.requests
		if r.Method != "PATCH" {
			return fmt.Errorf("expected cancellation")
		}
		return nil
	})
	sc.Step(`^cancellation was submitted once and checked twice$`, func() error {
		s := current()
		if len(s.requests) != 3 {
			return fmt.Errorf("expected three requests")
		}
		for _, method := range []string{"PATCH", "GET", "GET"} {
			r := <-s.requests
			if r.Method != method {
				return fmt.Errorf("unexpected cancellation sequence")
			}
		}
		return nil
	})

	sc.Step(`^I am (attending|waitlisted) for a huddl$`, func(status string) { cancellationMembership = status })
	sc.Step(`^I cancel my RSVP$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI([]string{"rsvp", "cancel", "11111111-1111-4111-8111-111111111111"}, "")
	})
	sc.Step(`^the backend confirms I am no longer attending or waitlisted$`, func() error {
		s := current()
		if !cancelled || s.stdout.String() != "RSVP cancelled for huddl 11111111-1111-4111-8111-111111111111. No confirmed or waitlisted membership remains.\n" || len(s.requests) != 3 {
			return fmt.Errorf("expected verified cancellation")
		}
		r := <-s.requests
		if r.Method != "PATCH" {
			return fmt.Errorf("expected cancellation first")
		}
		for _, status := range []string{"attending", "waitlisted"} {
			r := <-s.requests
			if r.Method != "GET" || r.URL.Query().Get("relationship") != status {
				return fmt.Errorf("expected both membership checks")
			}
		}
		return nil
	})

	sc.Step(`^I search everywhere despite my profile$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI([]string{"search", "--anywhere"}, "")
	})
	sc.Step(`^the profile is bypassed without a geographic restriction$`, func() error {
		s := current()
		if len(s.requests) != 1 || !strings.Contains(s.stdout.String(), "Searching everywhere") {
			return fmt.Errorf("expected unrestricted search only")
		}
		r := <-s.requests
		if r.Method != "GET" || r.URL.Path != "/api/json/huddlz" || r.URL.Query().Get("search_latitude") != "" || r.URL.Query().Get("search_longitude") != "" {
			return fmt.Errorf("unexpected search restriction or profile update")
		}
		return nil
	})

	sc.Step(`^my profile location is (unset|incomplete|unavailable)$`, func(mode string) { profileMode = mode })
	sc.Step(`^the profile lookup is followed by an unrestricted search$`, func() error {
		s := current()
		if len(s.requests) != 2 || !strings.Contains(s.stdout.String(), "Searching everywhere") {
			return fmt.Errorf("expected profile and unrestricted search")
		}
		<-s.requests
		r := <-s.requests
		if r.URL.Query().Get("search_latitude") != "" || r.URL.Query().Get("search_longitude") != "" {
			return fmt.Errorf("unexpected location")
		}
		return nil
	})

	sc.Step(`^I search using my saved profile$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI([]string{"search"}, "")
	})
	sc.Step(`^the search uses my current profile location$`, func() error {
		s := current()
		if len(s.requests) != 2 || !strings.Contains(s.stdout.String(), "within 25 miles of 30.2672, -97.7431") || !strings.Contains(s.stdout.String(), "America/Chicago") {
			return fmt.Errorf("expected profile scope")
		}
		r := <-s.requests
		if r.URL.Path != "/api/json/profile" {
			return fmt.Errorf("expected profile read")
		}
		r = <-s.requests
		q := r.URL.Query()
		if q.Get("search_latitude") != "30.2672" || q.Get("search_longitude") != "-97.7431" || q.Get("distance_miles") != "25" || q.Get("search_time_zone") != "America/Chicago" {
			return fmt.Errorf("incorrect profile search")
		}
		return nil
	})

	sc.Step(`^the waitlist API reports (waitlisted|confirmed|none)$`, func(state string) { waitlistState = state })
	sc.Step(`^I request waitlist membership$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI([]string{"rsvp", "waitlist", "11111111-1111-4111-8111-111111111111"}, "")
	})
	sc.Step(`^the waitlist result reports (waitlisted|confirmed)$`, func(state string) error {
		s := current()
		want := "Waitlisted for huddl"
		if state == "confirmed" {
			want = "Attendance confirmed for huddl"
		}
		if !strings.HasPrefix(s.stdout.String(), want) || len(s.requests) != 1 {
			return fmt.Errorf("expected backend state %s", state)
		}
		return nil
	})

	sc.Step(`^I RSVP to a huddl with space available$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return runCLI([]string{"rsvp", "11111111-1111-4111-8111-111111111111"}, "")
	})
	sc.Step(`^attendance is confirmed by the backend$`, func() error {
		s := current()
		if s.stdout.String() != "Attendance confirmed for huddl 11111111-1111-4111-8111-111111111111.\n" {
			return fmt.Errorf("expected confirmed attendance")
		}
		if len(s.requests) != 2 {
			return fmt.Errorf("expected RSVP and confirmation lookup")
		}
		for _, method := range []string{"PATCH", "GET"} {
			r := <-s.requests
			if r.Method != method {
				return fmt.Errorf("unexpected RSVP sequence")
			}
		}
		return nil
	})

	sc.Step(`^I check authentication status for another server$`, func() error { serverOverride = "http://127.0.0.1:1"; return run([]string{"status"}, "") })
	sc.Step(`^only a private session token is saved$`, func() error {
		count := 0
		err := filepath.Walk(home, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), "test-password") {
				return fmt.Errorf("password was persisted")
			}
			if string(data) != "test-session-secret" || info.Mode().Perm() != 0600 {
				return fmt.Errorf("expected user-only token file")
			}
			count++
			return nil
		})
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("expected one saved token, got %d", count)
		}
		return nil
	})

	sc.Step(`^the login endpoint redirects to another destination$`, func() { redirectLogin = true })
	sc.Step(`^no redirected credential request is made$`, func() error {
		s := current()
		if len(s.requests) != 1 {
			return fmt.Errorf("expected only initial login request, got %d", len(s.requests))
		}
		r := <-s.requests
		if r.URL.Path != "/api/auth/sign_in" {
			return fmt.Errorf("unexpected request path")
		}
		return nil
	})
	sc.Step(`^I attempt login over remote plaintext HTTP$`, func() error {
		serverOverride = "http://example.invalid"
		return run([]string{"login", "--email", "person@example.com", "--password-stdin"}, "test-password\n")
	})
	sc.Step(`^authentication rejects the insecure server without a request$`, func() error {
		s := current()
		if s.exitCode != 2 || s.stdout.Len() != 0 || !strings.Contains(s.stderr.String(), "authentication requires HTTPS") || len(s.requests) != 0 {
			return fmt.Errorf("expected insecure-server usage rejection")
		}
		return nil
	})

	sc.Step(`^account verification returns HTTP (\d+)$`, func(status int) { accountStatus = status })
	sc.Step(`^my saved credentials have expired$`, func() {
		accountStatus = http.StatusUnauthorized
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
	})
	sc.Step(`^authentication is required without prompting or exposing credentials$`, func() error {
		s := current()
		if s.exitCode != 1 || s.stdout.Len() != 0 || s.stderr.String() != "Authentication required: this session is no longer accepted. Run 'huddlz auth login --email <email>' to log in again.\n" {
			return fmt.Errorf("expected safe authentication-required diagnostic, exit=%d", s.exitCode)
		}
		return nil
	})
	sc.Step(`^only the rejected account verification was attempted$`, func() error {
		s := current()
		if len(s.requests) != 1 {
			return fmt.Errorf("expected one account verification, got %d", len(s.requests))
		}
		r := <-s.requests
		if r.Method != "GET" || r.URL.Path != "/api/auth/me" {
			return fmt.Errorf("unexpected authenticated operation")
		}
		return nil
	})

	sc.Step(`^server revocation fails with "([^"]*)"$`, func(failure string) { revocationFailure = failure })
	sc.Step(`^local sign-out reports unconfirmed server revocation$`, func() error {
		s := current()
		if s.exitCode != 1 || s.stdout.Len() != 0 || !strings.HasPrefix(s.stderr.String(), "Saved session removed locally; server revocation could not be confirmed:") || strings.Contains(s.stderr.String(), "test-session-secret") {
			return fmt.Errorf("expected safe partial sign-out failure")
		}
		if len(s.requests) != 1 {
			return fmt.Errorf("expected one revocation attempt")
		}
		r := <-s.requests
		if r.Method != "DELETE" || r.URL.Path != "/api/auth/sign_out" {
			return fmt.Errorf("unexpected revocation request")
		}
		return nil
	})
	sc.Step(`^I am already signed out without a server request$`, func() error {
		s := current()
		if s.stdout.String() != "Already signed out of this server.\n" || len(s.requests) != 0 {
			return fmt.Errorf("expected local already-signed-out result")
		}
		return nil
	})

	sc.Step(`^I sign out$`, func() error {
		s := current()
		for len(s.requests) > 0 {
			<-s.requests
		}
		return run([]string{"logout"}, "")
	})
	sc.Step(`^sign-out revokes the saved session without exposing it$`, func() error {
		s := current()
		if s.stdout.String() != "Signed out. Saved session removed and server token revoked.\n" {
			return fmt.Errorf("expected confirmed sign-out")
		}
		if len(s.requests) != 1 {
			return fmt.Errorf("expected one sign-out request, got %d", len(s.requests))
		}
		r := <-s.requests
		if r.Method != "DELETE" || r.URL.Path != "/api/auth/sign_out" {
			return fmt.Errorf("unexpected sign-out request")
		}
		return nil
	})
	sc.Step(`^no authenticated request follows sign-out$`, func() error {
		if len(current().requests) != 0 {
			return fmt.Errorf("unexpected request after sign-out")
		}
		return nil
	})

	sc.Step(`^I log in with an incorrect password$`, func() error {
		return run([]string{"login", "--email", "person@example.com", "--password-stdin"}, "wrong-password\n")
	})
	sc.Step(`^the rejected login is reported without credentials$`, func() error {
		s := current()
		if s.exitCode != 1 || s.stdout.Len() != 0 || s.stderr.String() != "Login failed: email or password was not accepted. Saved sessions have not been changed.\n" {
			return fmt.Errorf("expected safe rejected-login diagnostic, exit=%d", s.exitCode)
		}
		return nil
	})
	sc.Step(`^no account verification followed the rejected login$`, func() error {
		s := current()
		if len(s.requests) != 1 {
			return fmt.Errorf("expected just one sign-in request, got %d", len(s.requests))
		}
		r := <-s.requests
		if r.URL.Path != "/api/auth/sign_in" {
			return fmt.Errorf("unexpected request path")
		}
		return nil
	})

	sc.Step(`^I log in with my password on stdin$`, func() error {
		return run([]string{"login", "--email", "person@example.com", "--password-stdin"}, "test-password\n")
	})
	sc.Step(`^I check authentication status in a new process$`, func() error { return run([]string{"status"}, "") })
	sc.Step(`^I see my account without credentials$`, func() error {
		s := current()
		out := s.stdout.String()
		if !strings.Contains(out, "person@example.com") || !strings.Contains(out, "Test Person") || !strings.Contains(out, "account-id") || strings.Contains(out+s.stderr.String(), "test-password") || strings.Contains(out+s.stderr.String(), "test-session-secret") {
			return fmt.Errorf("incorrect account output")
		}
		return nil
	})
	sc.Step(`^the server verified my saved session$`, func() error {
		s := current()
		if len(s.requests) != 3 {
			return fmt.Errorf("expected sign-in and two account verifications, got %d requests", len(s.requests))
		}
		for _, path := range []string{"/api/auth/sign_in", "/api/auth/me", "/api/auth/me"} {
			r := <-s.requests
			if r.URL.Path != path {
				return fmt.Errorf("unexpected auth request %s", r.URL.Path)
			}
		}
		return nil
	})
}
