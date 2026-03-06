package types

type PageMeta struct {
	HasMore      bool   `json:"has_more"`
	AfterCursor  string `json:"after_cursor,omitempty"`
	BeforeCursor string `json:"before_cursor,omitempty"`
}

type PageLinks struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

type TicketPage struct {
	Tickets []Ticket  `json:"tickets"`
	Meta    PageMeta  `json:"meta"`
	Links   PageLinks `json:"links"`
	Count   int       `json:"count,omitempty"`
}
