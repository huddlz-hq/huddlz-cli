package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func logout(server *url.URL, stdout, stderr io.Writer, jsonOutput bool) int {
	token, loadErr := loadSession(server)
	if err := removeSession(server); err != nil {
		fmt.Fprintln(stderr, "Logout failed:", err)
		return 1
	}
	message := "Already signed out of this server."
	if !errors.Is(loadErr, errNoSession) {
		if loadErr != nil {
			fmt.Fprintln(stderr, "Saved session removed locally; server revocation could not be confirmed:", loadErr)
			return 1
		}
		if err := authRequest(server, http.MethodDelete, "sign_out", token, nil, nil); err != nil {
			fmt.Fprintln(stderr, "Saved session removed locally; server revocation could not be confirmed:", err)
			return 1
		}
		message = "Signed out. Saved session removed and server token revoked."
	}
	if jsonOutput {
		return writeJSON(stdout, stderr, struct {
			LocalRemoved  bool `json:"local_removed"`
			ServerRevoked bool `json:"server_revoked"`
		}{true, loadErr == nil})
	}
	if _, err := fmt.Fprintln(stdout, message); err != nil {
		fmt.Fprintln(stderr, "Could not write output:", err)
		return 1
	}
	return 0
}
