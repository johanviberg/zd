package types

import "time"

type Ticket struct {
	ID               int64          `json:"id"`
	URL              string         `json:"url,omitempty"`
	Subject          string         `json:"subject"`
	Description      string         `json:"description,omitempty"`
	Status           string         `json:"status"`
	Priority         string         `json:"priority,omitempty"`
	Type             string         `json:"type,omitempty"`
	RequesterID      int64          `json:"requester_id,omitempty"`
	AssigneeID       int64          `json:"assignee_id,omitempty"`
	GroupID          int64          `json:"group_id,omitempty"`
	OrganizationID   int64          `json:"organization_id,omitempty"`
	Tags             []string       `json:"tags,omitempty"`
	CustomFields     []CustomField  `json:"custom_fields,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type Comment struct {
	ID        int64     `json:"id,omitempty"`
	Body      string    `json:"body"`
	Public    *bool     `json:"public,omitempty"`
	AuthorID  int64     `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type CustomField struct {
	ID    int64       `json:"id"`
	Value interface{} `json:"value"`
}

type CreateTicketRequest struct {
	Subject        string        `json:"subject"`
	Comment        Comment       `json:"comment"`
	Priority       string        `json:"priority,omitempty"`
	Type           string        `json:"type,omitempty"`
	Status         string        `json:"status,omitempty"`
	AssigneeID     int64         `json:"assignee_id,omitempty"`
	GroupID        int64         `json:"group_id,omitempty"`
	Tags           []string      `json:"tags,omitempty"`
	CustomFields   []CustomField `json:"custom_fields,omitempty"`
	RequesterEmail string        `json:"-"`
	RequesterName  string        `json:"-"`
	Requester      *Requester    `json:"requester,omitempty"`
}

type Requester struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

type UpdateTicketRequest struct {
	Subject      string        `json:"subject,omitempty"`
	Comment      *Comment      `json:"comment,omitempty"`
	Priority     string        `json:"priority,omitempty"`
	Status       string        `json:"status,omitempty"`
	AssigneeID   *int64        `json:"assignee_id,omitempty"`
	GroupID      *int64        `json:"group_id,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	AddTags      []string      `json:"additional_tags,omitempty"`
	RemoveTags   []string      `json:"remove_tags,omitempty"`
	CustomFields []CustomField `json:"custom_fields,omitempty"`
	SafeUpdate   bool          `json:"safe_update,omitempty"`
}

type ListTicketsOptions struct {
	Limit     int
	Cursor    string
	Sort      string
	SortOrder string
	Status    string
	Assignee  int64
	Group     int64
}

type GetTicketOptions struct {
	Include string
}
