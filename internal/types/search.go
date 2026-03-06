package types

type SearchOptions struct {
	Limit     int
	Cursor    string
	SortBy    string
	SortOrder string
	Export    bool
	Include   string
}

type SearchPage struct {
	Results []SearchResult `json:"results"`
	Users   []User         `json:"users,omitempty"`
	Meta    PageMeta       `json:"meta"`
	Links   PageLinks      `json:"links"`
	Count   int            `json:"count"`
}

type SearchResult struct {
	Ticket
	ResultType string `json:"result_type"`
}
