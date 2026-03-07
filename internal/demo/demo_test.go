package demo

import (
	"context"
	"testing"

	"github.com/johanviberg/zd/internal/types"
)

func TestStoreGenerates100Tickets(t *testing.T) {
	s := NewStore()
	if got := len(s.Tickets); got != 100 {
		t.Fatalf("expected 100 tickets, got %d", got)
	}
}

func TestStoreGenerates10Users(t *testing.T) {
	s := NewStore()
	if got := len(s.Users); got != 10 {
		t.Fatalf("expected 10 users, got %d", got)
	}
}

func TestStoreDeterministic(t *testing.T) {
	s1 := NewStore()
	s2 := NewStore()

	for id := int64(1); id <= 100; id++ {
		t1 := s1.Tickets[id]
		t2 := s2.Tickets[id]
		if t1.Subject != t2.Subject || t1.Status != t2.Status || t1.Priority != t2.Priority {
			t.Fatalf("ticket %d differs between stores", id)
		}
	}
}

func TestTicketServiceList(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page, err := svc.List(ctx, &types.ListTicketsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Tickets) != 10 {
		t.Fatalf("expected 10 tickets, got %d", len(page.Tickets))
	}
	if !page.Meta.HasMore {
		t.Fatal("expected HasMore=true")
	}
	if page.Count != 100 {
		t.Fatalf("expected count=100, got %d", page.Count)
	}
}

func TestTicketServiceListPagination(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page1, err := svc.List(ctx, &types.ListTicketsOptions{Limit: 25})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !page1.Meta.HasMore {
		t.Fatal("expected HasMore after first page")
	}

	page2, err := svc.List(ctx, &types.ListTicketsOptions{Limit: 25, Cursor: page1.Meta.AfterCursor})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ensure no overlap
	ids := make(map[int64]bool)
	for _, ticket := range page1.Tickets {
		ids[ticket.ID] = true
	}
	for _, ticket := range page2.Tickets {
		if ids[ticket.ID] {
			t.Fatalf("duplicate ticket ID %d across pages", ticket.ID)
		}
	}
}

func TestTicketServiceListFilterStatus(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page, err := svc.List(ctx, &types.ListTicketsOptions{Status: "open", Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ticket := range page.Tickets {
		if ticket.Status != "open" {
			t.Fatalf("expected status=open, got %s", ticket.Status)
		}
	}
	if len(page.Tickets) == 0 {
		t.Fatal("expected at least one open ticket")
	}
}

func TestTicketServiceListSort(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page, err := svc.List(ctx, &types.ListTicketsOptions{Sort: "created_at", SortOrder: "asc", Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(page.Tickets); i++ {
		if page.Tickets[i].CreatedAt.Before(page.Tickets[i-1].CreatedAt) {
			t.Fatal("tickets not sorted by created_at ascending")
		}
	}
}

func TestTicketServiceGet(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	result, err := svc.Get(ctx, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ticket.ID != 1 {
		t.Fatalf("expected ticket ID=1, got %d", result.Ticket.ID)
	}
}

func TestTicketServiceGetNotFound(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	_, err := svc.Get(ctx, 999, nil)
	if err == nil {
		t.Fatal("expected error for non-existent ticket")
	}
	appErr, ok := err.(*types.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "not_found" {
		t.Fatalf("expected not_found, got %s", appErr.Code)
	}
}

func TestTicketServiceGetWithUsers(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	result, err := svc.Get(ctx, 1, &types.GetTicketOptions{Include: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Users) == 0 {
		t.Fatal("expected users to be populated")
	}
}

func TestTicketServiceCreate(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	ticket, err := svc.Create(ctx, &types.CreateTicketRequest{
		Subject:  "Test ticket",
		Comment:  types.Comment{Body: "Initial comment"},
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Subject != "Test ticket" {
		t.Fatalf("expected subject 'Test ticket', got '%s'", ticket.Subject)
	}
	if ticket.Status != "new" {
		t.Fatalf("expected status 'new', got '%s'", ticket.Status)
	}
	if ticket.Priority != "high" {
		t.Fatalf("expected priority 'high', got '%s'", ticket.Priority)
	}

	// Verify stored
	result, err := svc.Get(ctx, ticket.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ticket.Subject != "Test ticket" {
		t.Fatal("created ticket not found in store")
	}
}

func TestTicketServiceUpdate(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	updated, err := svc.Update(ctx, 1, &types.UpdateTicketRequest{
		Status:   "solved",
		Priority: "urgent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "solved" {
		t.Fatalf("expected status 'solved', got '%s'", updated.Status)
	}
	if updated.Priority != "urgent" {
		t.Fatalf("expected priority 'urgent', got '%s'", updated.Priority)
	}
}

func TestTicketServiceUpdateWithComment(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	commentsBefore := len(s.Comments[1])
	pub := true
	_, err := svc.Update(ctx, 1, &types.UpdateTicketRequest{
		Comment: &types.Comment{Body: "New comment", Public: &pub},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Comments[1]) != commentsBefore+1 {
		t.Fatal("expected comment to be added")
	}
}

func TestTicketServiceDelete(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	err := svc.Delete(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Get(ctx, 1, nil)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestTicketServiceDeleteNotFound(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	err := svc.Delete(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent ticket")
	}
}

func TestTicketServiceListComments(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page, err := svc.ListComments(ctx, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Comments) == 0 {
		t.Fatal("expected at least one comment")
	}
}

func TestSearchServiceSubstring(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "billing", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Results) == 0 {
		t.Fatal("expected results for 'billing'")
	}
	if page.Count != len(page.Results) {
		t.Fatalf("count mismatch: %d vs %d", page.Count, len(page.Results))
	}
}

func TestSearchServiceFieldPrefix(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "status:open", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range page.Results {
		if r.Status != "open" {
			t.Fatalf("expected status=open, got %s", r.Status)
		}
	}
	if len(page.Results) == 0 {
		t.Fatal("expected at least one open ticket")
	}
}

func TestSearchServiceCombined(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "status:open priority:high", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range page.Results {
		if r.Status != "open" || r.Priority != "high" {
			t.Fatalf("expected status=open priority=high, got status=%s priority=%s", r.Status, r.Priority)
		}
	}
}

func TestSearchServiceTagFilter(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "tags:billing", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Results) == 0 {
		t.Fatal("expected results for tags:billing")
	}
	for _, r := range page.Results {
		found := false
		for _, tag := range r.Tags {
			if tag == "billing" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ticket %d missing billing tag", r.ID)
		}
	}
}

func TestSearchServicePagination(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page1, err := svc.Search(ctx, "status:open", &types.SearchOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page1.Count <= 5 && page1.Meta.HasMore {
		t.Fatal("HasMore should be false when count <= limit")
	}
	if page1.Count > 5 && !page1.Meta.HasMore {
		t.Fatal("HasMore should be true when count > limit")
	}
}

func TestUserServiceGetMe(t *testing.T) {
	s := NewStore()
	svc := NewUserService(s)
	ctx := context.Background()

	user, err := svc.GetMe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != "agent" {
		t.Fatalf("expected agent role, got %s", user.Role)
	}
	if user.Name != "Sarah Chen" {
		t.Fatalf("expected Sarah Chen, got %s", user.Name)
	}
}
