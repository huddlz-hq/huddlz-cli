package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func huddlEndpoint() (*url.URL, error) {
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
	return endpoint.JoinPath("api/json/huddlz"), nil
}

// getJSONAPI applies the shared transport policy for public reads.
func getJSONAPI(endpoint *url.URL) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.api+json")
	client := &http.Client{Timeout: 15 * time.Second}
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
