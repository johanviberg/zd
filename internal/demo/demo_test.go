package demo

import (
	"context"
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreGenerates100Tickets(t *testing.T) {
	s := NewStore()
	assert.Len(t, s.Tickets, 100)
}

func TestStoreGenerates10Users(t *testing.T) {
	s := NewStore()
	assert.Len(t, s.Users, 10)
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
	require.NoError(t, err)
	require.Len(t, page.Tickets, 10)
	assert.True(t, page.Meta.HasMore, "expected HasMore=true")
	assert.Equal(t, 100, page.Count)
}

func TestTicketServiceListPagination(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page1, err := svc.List(ctx, &types.ListTicketsOptions{Limit: 25})
	require.NoError(t, err)
	assert.True(t, page1.Meta.HasMore, "expected HasMore after first page")

	page2, err := svc.List(ctx, &types.ListTicketsOptions{Limit: 25, Cursor: page1.Meta.AfterCursor})
	require.NoError(t, err)

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
	require.NoError(t, err)
	for _, ticket := range page.Tickets {
		assert.Equal(t, "open", ticket.Status)
	}
	assert.NotEmpty(t, page.Tickets, "expected at least one open ticket")
}

func TestTicketServiceListSort(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page, err := svc.List(ctx, &types.ListTicketsOptions{Sort: "created_at", SortOrder: "asc", Limit: 100})
	require.NoError(t, err)
	for i := 1; i < len(page.Tickets); i++ {
		assert.False(t, page.Tickets[i].CreatedAt.Before(page.Tickets[i-1].CreatedAt), "tickets not sorted by created_at ascending")
	}
}

func TestTicketServiceGet(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	result, err := svc.Get(ctx, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Ticket.ID)
}

func TestTicketServiceGetNotFound(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	_, err := svc.Get(ctx, 999, nil)
	require.Error(t, err)
	appErr, ok := err.(*types.AppError)
	require.True(t, ok, "expected AppError, got %T", err)
	assert.Equal(t, "not_found", appErr.Code)
}

func TestTicketServiceGetWithUsers(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	result, err := svc.Get(ctx, 1, &types.GetTicketOptions{Include: "users"})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Users, "expected users to be populated")
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
	require.NoError(t, err)
	assert.Equal(t, "Test ticket", ticket.Subject)
	assert.Equal(t, "new", ticket.Status)
	assert.Equal(t, "high", ticket.Priority)

	// Verify stored
	result, err := svc.Get(ctx, ticket.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "Test ticket", result.Ticket.Subject, "created ticket not found in store")
}

func TestTicketServiceUpdate(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	updated, err := svc.Update(ctx, 1, &types.UpdateTicketRequest{
		Status:   "solved",
		Priority: "urgent",
	})
	require.NoError(t, err)
	assert.Equal(t, "solved", updated.Status)
	assert.Equal(t, "urgent", updated.Priority)
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
	require.NoError(t, err)
	assert.Len(t, s.Comments[1], commentsBefore+1, "expected comment to be added")
}

func TestTicketServiceDelete(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	err := svc.Delete(ctx, 1)
	require.NoError(t, err)

	_, err = svc.Get(ctx, 1, nil)
	assert.Error(t, err, "expected error after delete")
}

func TestTicketServiceDeleteNotFound(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	err := svc.Delete(ctx, 999)
	assert.Error(t, err, "expected error for non-existent ticket")
}

func TestTicketServiceListComments(t *testing.T) {
	s := NewStore()
	svc := NewTicketService(s)
	ctx := context.Background()

	page, err := svc.ListComments(ctx, 1, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, page.Comments, "expected at least one comment")
}

func TestSearchServiceSubstring(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "billing", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, page.Results, "expected results for 'billing'")
	assert.GreaterOrEqual(t, page.Count, len(page.Results))
}

func TestSearchServiceFieldPrefix(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "status:open", nil)
	require.NoError(t, err)
	for _, r := range page.Results {
		assert.Equal(t, "open", r.Status)
	}
	assert.NotEmpty(t, page.Results, "expected at least one open ticket")
}

func TestSearchServiceCombined(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "status:open priority:high", nil)
	require.NoError(t, err)
	for _, r := range page.Results {
		assert.Equal(t, "open", r.Status)
		assert.Equal(t, "high", r.Priority)
	}
}

func TestSearchServiceTagFilter(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page, err := svc.Search(ctx, "tags:billing", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, page.Results, "expected results for tags:billing")
	for _, r := range page.Results {
		found := false
		for _, tag := range r.Tags {
			if tag == "billing" {
				found = true
				break
			}
		}
		assert.True(t, found, "ticket %d missing billing tag", r.ID)
	}
}

func TestSearchServicePagination(t *testing.T) {
	s := NewStore()
	svc := NewSearchService(s)
	ctx := context.Background()

	page1, err := svc.Search(ctx, "status:open", &types.SearchOptions{Limit: 5})
	require.NoError(t, err)
	if page1.Count <= 5 {
		assert.False(t, page1.Meta.HasMore, "HasMore should be false when count <= limit")
	}
	if page1.Count > 5 {
		assert.True(t, page1.Meta.HasMore, "HasMore should be true when count > limit")
	}
}

func TestUserServiceAutocomplete(t *testing.T) {
	s := NewStore()
	svc := NewUserService(s)
	ctx := context.Background()

	users, err := svc.AutocompleteUsers(ctx, "sarah")
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "Sarah Chen", users[0].Name)
}

func TestUserServiceAutocompleteEmail(t *testing.T) {
	s := NewStore()
	svc := NewUserService(s)
	ctx := context.Background()

	users, err := svc.AutocompleteUsers(ctx, "customer.com")
	require.NoError(t, err)
	assert.Len(t, users, 6, "expected 6 end-users")
}

func TestUserServiceAutocompleteEmpty(t *testing.T) {
	s := NewStore()
	svc := NewUserService(s)
	ctx := context.Background()

	users, err := svc.AutocompleteUsers(ctx, "zzznomatch")
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserServiceGetMe(t *testing.T) {
	s := NewStore()
	svc := NewUserService(s)
	ctx := context.Background()

	user, err := svc.GetMe(ctx)
	require.NoError(t, err)
	assert.Equal(t, "agent", user.Role)
	assert.Equal(t, "Sarah Chen", user.Name)
}
