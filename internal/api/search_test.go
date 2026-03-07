package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/johanviberg/zd/internal/types"
)

func TestSearchService_Search(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/search_results.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v2/search") {
			t.Errorf("expected /api/v2/search, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "status:open") {
			t.Errorf("expected query to contain 'status:open', got %q", query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewSearchService(client)

	page, err := svc.Search(context.Background(), "status:open", &types.SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(page.Results))
	}
	if page.Count != 2 {
		t.Errorf("expected count 2, got %d", page.Count)
	}
}

func TestSearchService_SearchExport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v2/search/export") {
			t.Errorf("expected /api/v2/search/export, got %s", r.URL.Path)
		}
		if ft := r.URL.Query().Get("filter[type]"); ft != "ticket" {
			t.Errorf("expected filter[type]=ticket, got %q", ft)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[],"meta":{"has_more":false},"count":0}`))
	})

	client := testClient(t, handler)
	svc := NewSearchService(client)

	_, err := svc.Search(context.Background(), "status:open", &types.SearchOptions{Export: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
