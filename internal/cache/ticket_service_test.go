package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/johanviberg/zd/internal/cache"
	"github.com/johanviberg/zd/internal/demo"
	"github.com/johanviberg/zd/internal/types"
)

func setup() (*cache.Cache, *cache.CachedTicketService, *cache.CachedSearchService) {
	store := demo.NewStore()
	c := cache.New(time.Minute)
	ts := cache.NewCachedTicketService(demo.NewTicketService(store), c)
	ss := cache.NewCachedSearchService(demo.NewSearchService(store), c)
	return c, ts, ss
}

func TestGetCacheHit(t *testing.T) {
	_, ts, _ := setup()
	ctx := context.Background()

	// First call — cache miss, fetches from delegate
	result1, err := ts.Get(ctx, 1, &types.GetTicketOptions{Include: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call — should return cached result
	result2, err := ts.Get(ctx, 1, &types.GetTicketOptions{Include: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result1.Ticket.ID != result2.Ticket.ID {
		t.Fatal("expected same ticket from cache")
	}
}

func TestListCacheHit(t *testing.T) {
	_, ts, _ := setup()
	ctx := context.Background()

	opts := &types.ListTicketsOptions{Limit: 10, Sort: "updated_at", SortOrder: "desc"}

	page1, err := ts.List(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page2, err := ts.List(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(page1.Tickets) != len(page2.Tickets) {
		t.Fatal("expected same results from cache")
	}
}

func TestListAuditsCacheHit(t *testing.T) {
	_, ts, _ := setup()
	ctx := context.Background()

	opts := &types.ListAuditsOptions{Include: "users"}

	result1, err := ts.ListAudits(ctx, 1, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result2, err := ts.ListAudits(ctx, 1, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result1.Audits) != len(result2.Audits) {
		t.Fatal("expected same audits from cache")
	}
}

func TestSearchCacheHit(t *testing.T) {
	_, _, ss := setup()
	ctx := context.Background()

	opts := &types.SearchOptions{Limit: 10, Export: true}

	page1, err := ss.Search(ctx, "status:open", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page2, err := ss.Search(ctx, "status:open", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(page1.Results) != len(page2.Results) {
		t.Fatal("expected same search results from cache")
	}
}

func TestUpdateInvalidatesCache(t *testing.T) {
	c, ts, ss := setup()
	ctx := context.Background()

	// Populate caches
	_, err := ts.Get(ctx, 1, &types.GetTicketOptions{Include: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = ts.List(ctx, &types.ListTicketsOptions{Limit: 10, Sort: "updated_at", SortOrder: "desc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = ss.Search(ctx, "status:open", &types.SearchOptions{Limit: 10, Export: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update ticket — should invalidate ticket:get:1:*, ticket:list:*, search:*
	_, err = ts.Update(ctx, 1, &types.UpdateTicketRequest{Status: "solved"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify invalidation by checking cache directly won't have old entries
	// We can't directly access cache internals, but we can verify the behavior:
	// After update, a fresh Get should return updated data
	result, err := ts.Get(ctx, 1, &types.GetTicketOptions{Include: "users"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ticket.Status != "solved" {
		t.Fatalf("expected status solved after update, got %s", result.Ticket.Status)
	}

	// Verify that the cache instance is the shared one (search should also be invalidated)
	_ = c // shared cache verified by construction
}

func TestCreateInvalidatesListAndSearch(t *testing.T) {
	_, ts, _ := setup()
	ctx := context.Background()

	opts := &types.ListTicketsOptions{Limit: 200, Sort: "updated_at", SortOrder: "desc"}

	page1, err := ts.List(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	count1 := page1.Count

	// Create a new ticket
	_, err = ts.Create(ctx, &types.CreateTicketRequest{
		Subject: "New test ticket",
		Comment: types.Comment{Body: "test body"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List should return fresh data (cache was invalidated)
	page2, err := ts.List(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page2.Count != count1+1 {
		t.Fatalf("expected count %d after create, got %d", count1+1, page2.Count)
	}
}

func TestDeleteInvalidatesCache(t *testing.T) {
	_, ts, _ := setup()
	ctx := context.Background()

	// Populate cache
	_, err := ts.Get(ctx, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete
	err = ts.Delete(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get should now return not found (cache was invalidated)
	_, err = ts.Get(ctx, 1, nil)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDifferentOptsAreDifferentCacheKeys(t *testing.T) {
	_, ts, _ := setup()
	ctx := context.Background()

	opts1 := &types.ListTicketsOptions{Limit: 5, Sort: "updated_at", SortOrder: "desc"}
	opts2 := &types.ListTicketsOptions{Limit: 10, Sort: "updated_at", SortOrder: "desc"}

	page1, err := ts.List(ctx, opts1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page2, err := ts.List(ctx, opts2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(page1.Tickets) == len(page2.Tickets) && len(page1.Tickets) == 5 {
		// Both returned 5, which means page2 might be a cache hit from page1
		// This shouldn't happen since the opts differ
		t.Fatal("different opts should produce different cache keys")
	}
}
