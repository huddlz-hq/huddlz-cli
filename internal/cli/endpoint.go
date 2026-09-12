package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func huddlEndpoint() (*url.URL, error) {
	server, err := serverURL()
	if err != nil {
		return nil, err
	}
	return server.JoinPath("api/json/huddlz"), nil
}

func serverURL() (*url.URL, error) {
	base := os.Getenv("HUDDLZ_URL")
	if base == "" {
		base = "https://huddlz.com"
	}
	endpoint, err := url.Parse(base)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("HUDDLZ_URL must be an HTTP(S) server URL without credentials, query, or fragment.")
	}
	if endpoint.Path == "" {
		endpoint.Path = "/"
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/"
	return endpoint, nil
}

// getJSONAPI applies the shared transport policy for public reads.
func (inv *invocation) getJSONAPI(endpoint *url.URL) ([]byte, error) {
	request, err := http.NewRequestWithContext(inv.ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.api+json")
	client := &http.Client{Timeout: inv.timeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, httpStatusError(response.StatusCode)
	}
	const maxResponseBytes = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return nil, fmt.Errorf("invalid or oversized JSON:API response")
	}
	return body, nil
}

// httpStatusError retains the status without exposing a server error body.
type httpStatusError int

func (status httpStatusError) Error() string {
	return fmt.Sprintf("HTTP %d %s", status, http.StatusText(int(status)))
}
