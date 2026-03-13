package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollaboratorEntry_MarshalJSON_UserID(t *testing.T) {
	c := CollaboratorEntry{UserID: 12345}
	b, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Equal(t, "12345", string(b))
}

func TestCollaboratorEntry_MarshalJSON_Email(t *testing.T) {
	c := CollaboratorEntry{Email: "alice@example.com"}
	b, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Equal(t, `"alice@example.com"`, string(b))
}

func TestCollaboratorEntry_MarshalJSON_NameEmail(t *testing.T) {
	c := CollaboratorEntry{Name: "Alice Smith", Email: "alice@example.com"}
	b, err := json.Marshal(c)
	require.NoError(t, err)
	var obj map[string]string
	err = json.Unmarshal(b, &obj)
	require.NoError(t, err, "expected JSON object, got %s", b)
	assert.Equal(t, "Alice Smith", obj["name"])
	assert.Equal(t, "alice@example.com", obj["email"])
}

func TestCollaboratorEntry_MarshalJSON_UserIDTakesPrecedence(t *testing.T) {
	c := CollaboratorEntry{UserID: 99, Name: "Alice", Email: "alice@example.com"}
	b, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Equal(t, "99", string(b))
}

func TestUpdateTicketRequest_WithCollaborators(t *testing.T) {
	req := UpdateTicketRequest{
		Status: "open",
		AdditionalCollaborators: []CollaboratorEntry{
			{UserID: 100},
			{Email: "vendor@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	err = json.Unmarshal(b, &obj)
	require.NoError(t, err)
	collabs, ok := obj["additional_collaborators"]
	require.True(t, ok, "expected additional_collaborators in JSON")
	var arr []json.RawMessage
	err = json.Unmarshal(collabs, &arr)
	require.NoError(t, err)
	require.Len(t, arr, 3)
	// First should be bare number
	assert.Equal(t, "100", string(arr[0]))
	// Second should be bare string
	assert.Equal(t, `"vendor@example.com"`, string(arr[1]))
	// Third should be object
	var obj3 map[string]string
	err = json.Unmarshal(arr[2], &obj3)
	require.NoError(t, err, "expected object, got %s", arr[2])
}
