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
)

func TestTicketService_List(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/tickets_list.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/tickets" {
			t.Errorf("expected /api/v2/tickets, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	page, err := svc.List(context.Background(), &types.ListTicketsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Tickets) != 3 {
		t.Errorf("expected 3 tickets, got %d", len(page.Tickets))
	}
	if !page.Meta.HasMore {
		t.Error("expected has_more to be true")
	}
	if page.Tickets[0].ID != 1 {
		t.Errorf("expected first ticket ID 1, got %d", page.Tickets[0].ID)
	}
}

func TestTicketService_Get(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/ticket.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/tickets/12345" {
			t.Errorf("expected /api/v2/tickets/12345, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	ticket, err := svc.Get(context.Background(), 12345, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", ticket.ID)
	}
	if ticket.Subject != "Test Ticket" {
		t.Errorf("expected subject 'Test Ticket', got %q", ticket.Subject)
	}
	if ticket.Status != "open" {
		t.Errorf("expected status 'open', got %q", ticket.Status)
	}
}

func TestTicketService_Get_WithInclude(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		include := r.URL.Query().Get("include")
		if include != "users" {
			t.Errorf("expected include=users, got %q", include)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":1,"subject":"Test","status":"open","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z"}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	_, err := svc.Get(context.Background(), 1, &types.GetTicketOptions{Include: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTicketService_Create(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/tickets" {
			t.Errorf("expected /api/v2/tickets, got %s", r.URL.Path)
		}

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

		if req.Ticket.Subject != "New Ticket" {
			t.Errorf("expected subject 'New Ticket', got %q", req.Ticket.Subject)
		}

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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != 999 {
		t.Errorf("expected ID 999, got %d", ticket.ID)
	}
}

func TestTicketService_Update(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/tickets/100" {
			t.Errorf("expected /api/v2/tickets/100, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":100,"subject":"Updated","status":"pending","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-02T00:00:00Z"}}`))
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	ticket, err := svc.Update(context.Background(), 100, &types.UpdateTicketRequest{
		Status: "pending",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", ticket.Status)
	}
}

func TestTicketService_Delete(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/tickets/100" {
			t.Errorf("expected /api/v2/tickets/100, got %s", r.URL.Path)
		}
		w.WriteHeader(204)
	})

	client := testClient(t, handler)
	svc := NewTicketService(client)

	err := svc.Delete(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p1.Meta.HasMore {
		t.Error("expected has_more true on first page")
	}

	// Second page
	p2, err := svc.List(context.Background(), &types.ListTicketsOptions{Limit: 1, Cursor: p1.Meta.AfterCursor})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p2.Meta.HasMore {
		t.Error("expected has_more false on second page")
	}
}
