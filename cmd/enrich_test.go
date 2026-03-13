package cmd

import (
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUserMap(t *testing.T) {
	users := []types.User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	}

	m := buildUserMap(users)
	require.Len(t, m, 2)
	assert.Equal(t, "Alice", m[1].Name)
	assert.Equal(t, "bob@example.com", m[2].Email)
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
	require.True(t, ok, "expected map, got %T", result)
	assert.Equal(t, "Jane", m["requester_name"])
	assert.Equal(t, "jane@example.com", m["requester_email"])
	assert.Equal(t, "John", m["assignee_name"])
	assert.Equal(t, "john@example.com", m["assignee_email"])
}

func TestEnrichTicket_NoUsers(t *testing.T) {
	ticket := types.Ticket{
		ID:      1,
		Subject: "Test",
		Status:  "open",
	}

	result := enrichTicket(ticket, nil)
	// Should return the original ticket unchanged
	_, ok := result.(types.Ticket)
	assert.True(t, ok, "expected types.Ticket, got %T", result)
}
