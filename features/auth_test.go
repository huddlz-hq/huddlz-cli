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
		s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.requests <- r.Clone(context.Background())
			switch r.URL.Path {
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
	run := func(args []string, input string) error {
		s := current()
		s.stdout.Reset()
		s.stderr.Reset()
		s.exitCode = 0
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, append([]string{"auth"}, args...)...)
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
