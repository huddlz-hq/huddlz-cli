package cli

type searchJSONResult struct {
	Data       []searchHuddl     `json:"data"`
	Search     searchJSONContext `json:"search"`
	Pagination searchPage        `json:"pagination"`
}

type searchJSONContext struct {
	Query    string              `json:"query"`
	Date     string              `json:"date"`
	Type     string              `json:"type"`
	TimeZone string              `json:"time_zone"`
	Location *searchJSONLocation `json:"location"`
}

type searchJSONLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    int     `json:"radius_miles"`
}
