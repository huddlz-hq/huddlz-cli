package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/term"
)

const authHelp = `Usage:
  huddlz auth login --email <email> [--password-stdin]
  huddlz auth status
  huddlz auth logout

Login prompts for a hidden password; --password-stdin reads it from standard input.
Only the session token is saved, scoped to this server, in a user-only file.
All auth commands accept --json for structured results.
Status verifies the saved session with the server.
Logout removes the local session and attempts server revocation.
`

type authAccount struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func (inv *invocation) auth(args []string, stdout, stderr io.Writer) int {
	args, jsonOutput := outputArgs(args)
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		fmt.Fprint(stdout, authHelp)
		return 0
	}
	if len(args) == 0 || len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, authHelp); err != nil {
			return 1
		}
		return 0
	}
	var email string
	var passwordStdin bool
	switch args[0] {
	case "login":
		flags := pflag.NewFlagSet("login", pflag.ContinueOnError)
		flags.SetOutput(io.Discard)
		flags.StringVar(&email, "email", "", "Account email")
		flags.BoolVar(&passwordStdin, "password-stdin", false, "Read password from stdin")
		err := flags.Parse(args[1:])
		if errors.Is(err, pflag.ErrHelp) {
			fmt.Fprint(stdout, authHelp)
			return 0
		}
		if err != nil || flags.NArg() != 0 || strings.TrimSpace(email) == "" {
			fmt.Fprint(stderr, authHelp)
			return 2
		}
	case "status", "logout":
		if len(args) != 1 {
			fmt.Fprint(stderr, authHelp)
			return 2
		}
	default:
		fmt.Fprint(stderr, authHelp)
		return 2
	}
	server, err := authServer()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if args[0] == "logout" {
		return inv.logout(server, stdout, stderr, jsonOutput)
	}
	var token string
	if args[0] == "login" {
		password, err := inv.readPassword(passwordStdin, stderr)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		payload, _ := json.Marshal(struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}{email, password})
		var result struct {
			Token string `json:"token"`
		}
		if err := inv.authRequest(server, http.MethodPost, "sign_in", "", payload, &result); err != nil {
			var status httpStatusError
			if errors.As(err, &status) && status == http.StatusUnauthorized {
				fmt.Fprintln(stderr, "Login failed: email or password was not accepted. Saved sessions have not been changed.")
			} else {
				fmt.Fprintln(stderr, "Login failed:", err)
			}
			return 1
		}
		token = result.Token
		if token == "" || len(token) > 64<<10 || strings.ContainsAny(token, "\r\n\t ") {
			fmt.Fprintln(stderr, "Login failed: invalid session response.")
			return 1
		}
	} else {
		token, err = loadSession(server)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	var result struct {
		User *authAccount `json:"user"`
	}
	if err := inv.authRequest(server, http.MethodGet, "me", token, nil, &result); err != nil {
		var status httpStatusError
		if errors.As(err, &status) && status == http.StatusUnauthorized {
			fmt.Fprintln(stderr, "Authentication required: this session is no longer accepted. Run 'huddlz auth login --email <email>' to log in again.")
		} else {
			fmt.Fprintln(stderr, "Account verification failed:", err)
		}
		return 1
	}
	if result.User == nil || strings.TrimSpace(result.User.ID) == "" || strings.TrimSpace(result.User.Email) == "" {
		fmt.Fprintln(stderr, "Account verification failed: invalid account response.")
		return 1
	}
	if args[0] == "login" {
		if err := saveSession(server, token); err != nil {
			fmt.Fprintln(stderr, "Could not save session:", err)
			return 1
		}
	}
	if jsonOutput {
		return writeJSON(stdout, stderr, result)
	}
	_, err = fmt.Fprintf(stdout, "Account: %s\nEmail: %s\nID: %s\n", tableCell(result.User.DisplayName), tableCell(result.User.Email), tableCell(result.User.ID))
	if err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}

// Reading stdin may block indefinitely; cancellation must also release the CLI.
func (inv *invocation) readPassword(fromStdin bool, stderr io.Writer) (string, error) {
	type result struct {
		password string
		err      error
	}
	done := make(chan result, 1)
	go func() {
		password, err := readPassword(fromStdin, stderr)
		done <- result{password, err}
	}()
	select {
	case r := <-done:
		return r.password, r.err
	case <-inv.ctx.Done():
		return "", inv.ctx.Err()
	}
}

func readPassword(fromStdin bool, stderr io.Writer) (string, error) {
	var data []byte
	var err error
	if fromStdin {
		data, err = io.ReadAll(io.LimitReader(os.Stdin, 64<<10+1))
	} else {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return "", fmt.Errorf("use --password-stdin when no terminal is available")
		}
		fmt.Fprint(stderr, "Password: ")
		data, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(stderr)
	}
	if err != nil || len(data) > 64<<10 {
		return "", fmt.Errorf("could not read password")
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if password == "" {
		return "", fmt.Errorf("password must not be empty")
	}
	return password, nil
}

func authServer() (*url.URL, error) {
	server, err := serverURL()
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(server.Hostname())
	if server.Scheme != "https" && server.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("authentication requires HTTPS (HTTP is allowed for loopback development servers)")
	}
	return server, nil
}

// Authentication never follows redirects or prints response bodies and transport errors.
func (inv *invocation) authRequest(server *url.URL, method, action, token string, payload []byte, result any) error {
	return inv.sessionRequest(server.JoinPath("api/auth", action), method, token, payload, result, "application/json")
}

func (inv *invocation) sessionRequest(endpoint *url.URL, method, token string, payload []byte, result any, mediaType string) error {
	request, err := http.NewRequestWithContext(inv.ctx, method, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("could not create authentication request")
	}
	request.Header.Set("Accept", mediaType)
	if payload != nil {
		request.Header.Set("Content-Type", mediaType)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := http.Client{Timeout: inv.timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("authentication request could not be completed")
	}
	defer response.Body.Close()
	if result == nil {
		if response.StatusCode == http.StatusNoContent {
			return nil
		}
		return httpStatusError(response.StatusCode)
	}
	if response.StatusCode != http.StatusOK {
		return httpStatusError(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil || len(body) > 1<<20 || json.Unmarshal(body, result) != nil {
		return fmt.Errorf("invalid authentication response")
	}
	return nil
}
