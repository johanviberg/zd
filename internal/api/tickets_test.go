package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketService_List(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/tickets_list.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v2/tickets", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	page, err := svc.List(context.Background(), &types.ListTicketsOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Tickets, 3)
	assert.True(t, page.Meta.HasMore, "expected has_more to be true")
	assert.Equal(t, int64(1), page.Tickets[0].ID)
}

func TestTicketService_Get(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/ticket.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/tickets/12345", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	result, err := svc.Get(context.Background(), 12345, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(12345), result.Ticket.ID)
	assert.Equal(t, "Test Ticket", result.Ticket.Subject)
	assert.Equal(t, "open", result.Ticket.Status)
	assert.Empty(t, result.Users)
}

func TestTicketService_Get_WithInclude(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "users", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":1,"subject":"Test","status":"open","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	_, err := svc.Get(context.Background(), 1, &types.GetTicketOptions{Include: "users"})
	require.NoError(t, err)
}

func TestTicketService_Get_WithUsers(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/ticket_with_users.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	result, err := svc.Get(context.Background(), 12345, &types.GetTicketOptions{Include: "users"})
	require.NoError(t, err)
	assert.Equal(t, int64(12345), result.Ticket.ID)
	require.Len(t, result.Users, 2)
	assert.Equal(t, "Jane Requester", result.Users[0].Name)
	assert.Equal(t, "john@example.com", result.Users[1].Email)
}

func TestTicketService_List_WithInclude(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/tickets_list_with_users.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "users", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	page, err := svc.List(context.Background(), &types.ListTicketsOptions{Include: "users"})
	require.NoError(t, err)
	assert.Len(t, page.Tickets, 2)
	require.Len(t, page.Users, 3)
	assert.Equal(t, "Jane Requester", page.Users[0].Name)
}

func TestTicketService_Create(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/tickets", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req struct {
			Ticket struct {
				Subject string `json:"subject"`
				Comment struct {
					Body string `json:"body"`
				} `json:"comment"`
			} `json:"ticket"`
		}
		json.Unmarshal(body, &req)

		assert.Equal(t, "New Ticket", req.Ticket.Subject)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write([]byte(`{"ticket":{"id":999,"subject":"New Ticket","status":"new","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	ticket, err := svc.Create(context.Background(), &types.CreateTicketRequest{
		Subject: "New Ticket",
		Comment: types.Comment{Body: "Test body"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(999), ticket.ID)
}

func TestTicketService_Update(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/v2/tickets/100", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":100,"subject":"Updated","status":"pending","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	ticket, err := svc.Update(context.Background(), 100, &types.UpdateTicketRequest{
		Status: "pending",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", ticket.Status)
}

func TestTicketService_Delete(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v2/tickets/100", r.URL.Path)
		w.WriteHeader(204)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	err := svc.Delete(context.Background(), 100)
	require.NoError(t, err)
}

func TestTicketService_ListComments(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/comments.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v2/tickets/42/comments", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	page, err := svc.ListComments(context.Background(), 42, nil)
	require.NoError(t, err)
	require.Len(t, page.Comments, 2)
	assert.Equal(t, "First comment", page.Comments[0].Body)
	assert.Equal(t, int64(10), page.Comments[0].AuthorID)
	assert.Equal(t, "Internal note", page.Comments[1].Body)
	assert.True(t, page.Meta.HasMore, "expected has_more to be true")
}

func TestTicketService_ListComments_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("page[size]"))
		assert.Equal(t, "desc", r.URL.Query().Get("sort_order"))
		assert.Equal(t, "users", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"comments":[],"meta":{"has_more":false}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	_, err := svc.ListComments(context.Background(), 1, &types.ListCommentsOptions{
		Limit:     10,
		SortOrder: "desc",
		Include:   "users",
	})
	require.NoError(t, err)
}

func TestTicketService_ListComments_WithUsers(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/comments_with_users.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	page, err := svc.ListComments(context.Background(), 42, &types.ListCommentsOptions{
		Include: "users",
	})
	require.NoError(t, err)
	require.Len(t, page.Comments, 2)
	require.Len(t, page.Users, 2)
	assert.Equal(t, "Alice Agent", page.Users[0].Name)
	assert.Equal(t, "bob@example.com", page.Users[1].Email)
}

func TestTicketService_ListAudits(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/audits.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v2/tickets/42/audits", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	page, err := svc.ListAudits(context.Background(), 42, nil)
	require.NoError(t, err)
	require.Len(t, page.Audits, 3)
	assert.Equal(t, int64(1001), page.Audits[0].ID)
	assert.Equal(t, int64(42), page.Audits[0].TicketID)
	assert.Len(t, page.Audits[0].Events, 2)
	assert.Equal(t, "Comment", page.Audits[0].Events[0].Type)
	assert.Equal(t, "Initial description of the issue", page.Audits[0].Events[0].Body)
	assert.Len(t, page.Audits[0].Events[0].Attachments, 1)
	assert.True(t, page.Meta.HasMore, "expected has_more to be true")
}

func TestTicketService_ListAudits_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("page[size]"))
		assert.Equal(t, "asc", r.URL.Query().Get("sort_order"))
		assert.Equal(t, "users", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"audits":[],"meta":{"has_more":false}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	_, err := svc.ListAudits(context.Background(), 1, &types.ListAuditsOptions{
		Limit:     10,
		SortOrder: "asc",
		Include:   "users",
	})
	require.NoError(t, err)
}

func TestTicketService_List_Pagination(t *testing.T) {
	page := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		w.Header().Set("Content-Type", "application/json")
		if page == 1 {
			w.Write([]byte(`{"tickets":[{"id":1,"subject":"T1","status":"open","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}],"meta":{"has_more":true,"after_cursor":"cursor1"}}`))
		} else {
			w.Write([]byte(`{"tickets":[{"id":2,"subject":"T2","status":"open","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}],"meta":{"has_more":false}}`))
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := &Client{HTTPClient: server.Client(), BaseURL: server.URL}
	svc := NewTicketService(client)

	// First page
	p1, err := svc.List(context.Background(), &types.ListTicketsOptions{Limit: 1})
	require.NoError(t, err)
	assert.True(t, p1.Meta.HasMore, "expected has_more true on first page")

	// Second page
	p2, err := svc.List(context.Background(), &types.ListTicketsOptions{Limit: 1, Cursor: p1.Meta.AfterCursor})
	require.NoError(t, err)
	assert.False(t, p2.Meta.HasMore, "expected has_more false on second page")
}
