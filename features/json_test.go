package features_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cucumber/godog"
	"net/http"
	"net/http/httptest"
	"strings"
)

func registerJSON(sc *godog.ScenarioContext, current func() *searchScenario) {
	sc.Step(`^the JSON search API provides pagination "([^"]*)"$`, func(next string) {
		s := current()
		s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.requests <- r.Clone(context.Background())
			links := ""
			switch next {
			case "next":
				links = `,"links":{"next":"?page[offset]=1"}`
			case "end":
				links = `,"links":{"next":null}`
			case "invalid":
				links = `,"links":{"next":"https://invalid.example/api/json/huddlz?page[offset]=1"}`
			}
			fmt.Fprint(w, `{"data":[]`+links+`}`)
		}))
	})
	sc.Step(`^JSON reports pagination "([^"]*)"$`, func(next string) error {
		var result struct {
			Search     struct{ Type string }
			Pagination struct {
				Known         bool
				Limit, Offset int
				NextOffset    *int    `json:"next_offset"`
				NextCommand   *string `json:"next_command"`
			}
		}
		if err := json.Unmarshal(current().stdout.Bytes(), &result); err != nil {
			return err
		}
		p := result.Pagination
		if result.Search.Type != "virtual" || p.Limit != 1 || p.Offset != 0 || p.Known != (next != "unknown") {
			return fmt.Errorf("unexpected pagination: %s", current().stdout.String())
		}
		if next == "next" {
			if p.NextOffset == nil || *p.NextOffset != 1 || p.NextCommand == nil || !strings.Contains(*p.NextCommand, "--json") || !strings.Contains(*p.NextCommand, "--type virtual") || !strings.Contains(*p.NextCommand, "--limit 1 --offset 1") {
				return fmt.Errorf("invalid next page: %s", current().stdout.String())
			}
		} else if p.NextOffset != nil || p.NextCommand != nil {
			return fmt.Errorf("unexpected next page: %s", current().stdout.String())
		}
		return nil
	})

	sc.Step(`^JSON preserves the search results and context$`, func() error {
		var result struct {
			Data []struct {
				ID         string
				Attributes struct {
					Title    string
					StartsAt string `json:"starts_at"`
					Location string `json:"physical_location"`
				}
			}
			Search struct {
				Query    string
				Date     string
				TimeZone string `json:"time_zone"`
				Location struct {
					Latitude, Longitude float64
					Radius              int `json:"radius_miles"`
				}
			}
			Pagination struct {
				Limit, Offset int
				Known         bool
			}
		}
		if err := json.Unmarshal(current().stdout.Bytes(), &result); err != nil {
			return err
		}
		if len(result.Data) != 2 || result.Data[0].ID != "11111111-1111-4111-8111-111111111111" || result.Data[0].Attributes.Title != "Board games at the library" || result.Data[0].Attributes.StartsAt != "2027-01-10T18:00:00Z" || result.Data[0].Attributes.Location != "Central Library" || result.Search.Query != "board games" || result.Search.Date != "upcoming" || result.Search.TimeZone != "Etc/UTC" || result.Search.Location.Latitude != 0 || result.Search.Location.Longitude != 0 || result.Search.Location.Radius != 25 || result.Pagination.Limit != 20 || result.Pagination.Offset != 0 || result.Pagination.Known {
			return fmt.Errorf("unexpected JSON: %s", current().stdout.String())
		}
		return nil
	})
	sc.Step(`^JSON contains an empty results array$`, func() error {
		var result struct{ Data json.RawMessage }
		if err := json.Unmarshal(current().stdout.Bytes(), &result); err != nil {
			return err
		}
		if string(result.Data) != "[]" {
			return fmt.Errorf("expected [], got %s", result.Data)
		}
		return nil
	})
}
