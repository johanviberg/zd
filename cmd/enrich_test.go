package cmd

import (
	"testing"

	"github.com/johanviberg/zd/internal/types"
)

func TestBuildUserMap(t *testing.T) {
	users := []types.User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	}

	m := buildUserMap(users)
	if len(m) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m))
	}
	if m[1].Name != "Alice" {
		t.Errorf("expected Alice, got %q", m[1].Name)
	}
	if m[2].Email != "bob@example.com" {
		t.Errorf("expected bob@example.com, got %q", m[2].Email)
	}
}

func TestEnrichTicket(t *testing.T) {
	ticket := types.Ticket{
		ID:          1,
		Subject:     "Test",
		Status:      "open",
		RequesterID: 100,
		AssigneeID:  200,
	}
	userMap := map[int64]types.User{
		100: {ID: 100, Name: "Jane", Email: "jane@example.com"},
		200: {ID: 200, Name: "John", Email: "john@example.com"},
	}

	result := enrichTicket(ticket, userMap)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["requester_name"] != "Jane" {
		t.Errorf("expected requester_name 'Jane', got %v", m["requester_name"])
	}
	if m["requester_email"] != "jane@example.com" {
		t.Errorf("expected requester_email 'jane@example.com', got %v", m["requester_email"])
	}
	if m["assignee_name"] != "John" {
		t.Errorf("expected assignee_name 'John', got %v", m["assignee_name"])
	}
	if m["assignee_email"] != "john@example.com" {
		t.Errorf("expected assignee_email 'john@example.com', got %v", m["assignee_email"])
	}
}

func TestEnrichTicket_NoUsers(t *testing.T) {
	ticket := types.Ticket{
		ID:      1,
		Subject: "Test",
		Status:  "open",
	}

	result := enrichTicket(ticket, nil)
	// Should return the original ticket unchanged
	if _, ok := result.(types.Ticket); !ok {
		t.Errorf("expected types.Ticket, got %T", result)
	}
}
