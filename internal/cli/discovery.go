package cli

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

func (inv *invocation) discoveryGet(endpoint *url.URL) ([]byte, error) {
	server, err := serverURL()
	if err != nil {
		return nil, err
	}
	token, err := loadSession(server)
	if errors.Is(err, errNoSession) {
		return inv.getJSONAPI(endpoint)
	}
	if err != nil {
		return nil, err
	}
	if _, err := authServer(); err != nil {
		return nil, err
	}
	var body json.RawMessage
	if err := inv.sessionRequest(endpoint, http.MethodGet, token, nil, &body, "application/vnd.api+json"); err != nil {
		return nil, err
	}
	return body, nil
}
