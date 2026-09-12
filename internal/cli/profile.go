package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
)

func applyProfileDefaults(options *searchOptions) error {
	if options.anywhere || options.location != nil {
		return nil
	}
	server, err := serverURL()
	if err != nil {
		return err
	}
	token, err := loadSession(server)
	if errors.Is(err, errNoSession) {
		return nil
	}
	if err != nil {
		return err
	}
	server, err = authServer()
	if err != nil {
		return err
	}
	var profile struct {
		Defaults *struct {
			Home   json.RawMessage `json:"home_location"`
			Radius int             `json:"distance_miles"`
		} `json:"search_defaults"`
	}
	if err := sessionRequest(server.JoinPath("api/json/profile"), http.MethodGet, token, nil, &profile, "application/json"); err != nil {
		return err
	}
	if profile.Defaults == nil {
		return fmt.Errorf("invalid profile search defaults")
	}
	if len(profile.Defaults.Home) == 0 {
		return fmt.Errorf("invalid profile search defaults")
	}
	if strings.TrimSpace(string(profile.Defaults.Home)) == "null" {
		return nil
	}
	var home struct {
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		TimeZone  string   `json:"time_zone"`
	}
	if json.Unmarshal(profile.Defaults.Home, &home) != nil {
		return fmt.Errorf("invalid profile home location")
	}
	if home.Latitude == nil || home.Longitude == nil || math.IsNaN(*home.Latitude) || math.IsNaN(*home.Longitude) || math.Abs(*home.Latitude) > 90 || math.Abs(*home.Longitude) > 180 || !isCanonicalTimeZone(home.TimeZone) || profile.Defaults.Radius < 5 || profile.Defaults.Radius > 100 {
		return fmt.Errorf("invalid profile home location")
	}
	options.location = &searchLocation{*home.Latitude, *home.Longitude, profile.Defaults.Radius}
	if !options.timeZoneSet {
		options.timeZone = home.TimeZone
	}
	return nil
}
