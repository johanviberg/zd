package api

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/johanviberg/zd/internal/types"
)

func TestArticleService_List(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/articles_list.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/help_center/articles" {
			t.Errorf("expected /api/v2/help_center/articles, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	page, err := svc.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Articles) != 3 {
		t.Errorf("expected 3 articles, got %d", len(page.Articles))
	}
	if !page.Meta.HasMore {
		t.Error("expected has_more to be true")
	}
	if page.Articles[0].ID != 101 {
		t.Errorf("expected first article ID 101, got %d", page.Articles[0].ID)
	}
	if page.Articles[0].Title != "Getting Started Guide" {
		t.Errorf("expected title 'Getting Started Guide', got %q", page.Articles[0].Title)
	}
}

func TestArticleService_List_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page[size]"); got != "10" {
			t.Errorf("expected page[size]=10, got %q", got)
		}
		if got := r.URL.Query().Get("sort_by"); got != "updated_at" {
			t.Errorf("expected sort_by=updated_at, got %q", got)
		}
		if got := r.URL.Query().Get("sort_order"); got != "asc" {
			t.Errorf("expected sort_order=asc, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"articles":[],"meta":{"has_more":false}}`))
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	_, err := svc.List(context.Background(), &types.ListArticlesOptions{
		Limit:     10,
		SortBy:    "updated_at",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArticleService_Get(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/article.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/help_center/articles/101" {
			t.Errorf("expected /api/v2/help_center/articles/101, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	result, err := svc.Get(context.Background(), 101)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Article.ID != 101 {
		t.Errorf("expected ID 101, got %d", result.Article.ID)
	}
	if result.Article.Title != "Getting Started Guide" {
		t.Errorf("expected title 'Getting Started Guide', got %q", result.Article.Title)
	}
	if !result.Article.Promoted {
		t.Error("expected promoted to be true")
	}
}

func TestArticleService_Get_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"RecordNotFound"}`))
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	_, err := svc.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*types.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.ExitCode != types.ExitNotFound {
		t.Errorf("expected exit code %d, got %d", types.ExitNotFound, appErr.ExitCode)
	}
}

func TestArticleService_Search(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/articles_search.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/help_center/articles/search" {
			t.Errorf("expected /api/v2/help_center/articles/search, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "password reset" {
			t.Errorf("expected query='password reset', got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	page, err := svc.Search(context.Background(), "password reset", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(page.Results))
	}
	if page.Results[0].Title != "Password Reset Instructions" {
		t.Errorf("expected title 'Password Reset Instructions', got %q", page.Results[0].Title)
	}
	if page.Count != 1 {
		t.Errorf("expected count 1, got %d", page.Count)
	}
}
