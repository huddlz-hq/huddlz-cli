package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var errInvalidAttendanceResponse = errors.New("invalid attendance mutation response")

func (inv *invocation) submitAttendance(server *url.URL, token, id, action string) (string, error) {
	payload, _ := json.Marshal(struct {
		Data huddlIdentity `json:"data"`
	}{huddlIdentity{ID: id, Type: "huddl"}})
	var response struct {
		Data *struct {
			huddlIdentity
			Attributes struct {
				State string `json:"attendance_state"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := inv.sessionRequest(server.JoinPath("api/json/huddlz", id, action), http.MethodPatch, token, payload, &response, "application/vnd.api+json"); err != nil {
		return "", err
	}
	if response.Data == nil || response.Data.ID != id || response.Data.Type != "huddl" {
		return "", errInvalidAttendanceResponse
	}
	return response.Data.Attributes.State, nil
}

func (inv *invocation) attendanceMembership(server *url.URL, token, id, relationship string) (bool, error) {
	endpoint := server.JoinPath("api/json/huddlz")
	endpoint.RawQuery = url.Values{"filter[id][eq]": {id}, "relationship": {relationship}, "date_filter": {"all"}, "search_time_zone": {"Etc/UTC"}, "page[limit]": {"1"}}.Encode()
	var response struct {
		Data *[]huddlIdentity `json:"data"`
	}
	if err := inv.sessionRequest(endpoint, http.MethodGet, token, nil, &response, "application/vnd.api+json"); err != nil {
		return false, err
	}
	if response.Data == nil || len(*response.Data) > 1 {
		return false, fmt.Errorf("invalid membership response")
	}
	if len(*response.Data) == 0 {
		return false, nil
	}
	h := (*response.Data)[0]
	if h.ID != id || h.Type != "huddl" {
		return false, fmt.Errorf("invalid membership resource")
	}
	return true, nil
}
