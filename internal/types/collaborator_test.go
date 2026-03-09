package types

import (
	"encoding/json"
	"testing"
)

func TestCollaboratorEntry_MarshalJSON_UserID(t *testing.T) {
	c := CollaboratorEntry{UserID: 12345}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "12345" {
		t.Errorf("expected 12345, got %s", b)
	}
}

func TestCollaboratorEntry_MarshalJSON_Email(t *testing.T) {
	c := CollaboratorEntry{Email: "alice@example.com"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"alice@example.com"` {
		t.Errorf("expected \"alice@example.com\", got %s", b)
	}
}

func TestCollaboratorEntry_MarshalJSON_NameEmail(t *testing.T) {
	c := CollaboratorEntry{Name: "Alice Smith", Email: "alice@example.com"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]string
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("expected JSON object, got %s", b)
	}
	if obj["name"] != "Alice Smith" {
		t.Errorf("expected name Alice Smith, got %s", obj["name"])
	}
	if obj["email"] != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", obj["email"])
	}
}

func TestCollaboratorEntry_MarshalJSON_UserIDTakesPrecedence(t *testing.T) {
	c := CollaboratorEntry{UserID: 99, Name: "Alice", Email: "alice@example.com"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "99" {
		t.Errorf("expected 99 (UserID takes precedence), got %s", b)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	collabs, ok := obj["additional_collaborators"]
	if !ok {
		t.Fatal("expected additional_collaborators in JSON")
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(collabs, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 collaborators, got %d", len(arr))
	}
	// First should be bare number
	if string(arr[0]) != "100" {
		t.Errorf("expected 100, got %s", arr[0])
	}
	// Second should be bare string
	if string(arr[1]) != `"vendor@example.com"` {
		t.Errorf("expected \"vendor@example.com\", got %s", arr[1])
	}
	// Third should be object
	var obj3 map[string]string
	if err := json.Unmarshal(arr[2], &obj3); err != nil {
		t.Errorf("expected object, got %s", arr[2])
	}
}
