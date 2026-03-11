package types

import "time"

// AuditEvent represents a single event within a ticket audit.
type AuditEvent struct {
	ID            int64        `json:"id"`
	Type          string       `json:"type"`
	FieldName     string       `json:"field_name,omitempty"`
	Value         interface{}  `json:"value"`
	PreviousValue interface{}  `json:"previous_value"`
	Body          string       `json:"body,omitempty"`
	HTMLBody      string       `json:"html_body,omitempty"`
	Public        *bool        `json:"public,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
	AuthorID      int64        `json:"author_id,omitempty"`
}

// Audit represents a single audit entry for a ticket.
type Audit struct {
	ID        int64        `json:"id"`
	TicketID  int64        `json:"ticket_id"`
	AuthorID  int64        `json:"author_id"`
	CreatedAt time.Time    `json:"created_at"`
	Events    []AuditEvent `json:"events"`
}

// AuditPage represents a paginated response of audits.
type AuditPage struct {
	Audits []Audit   `json:"audits"`
	Users  []User    `json:"users,omitempty"`
	Meta   PageMeta  `json:"meta"`
	Links  PageLinks `json:"links"`
	Count  int       `json:"count,omitempty"`
}

// ListAuditsOptions configures the ListAudits request.
type ListAuditsOptions struct {
	Limit     int
	Cursor    string
	SortOrder string
	Include   string
}
