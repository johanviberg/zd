package api

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArticleService_List(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/articles_list.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v2/help_center/articles", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	page, err := svc.List(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, page.Articles, 3)
	assert.True(t, page.Meta.HasMore, "expected has_more to be true")
	assert.Equal(t, int64(101), page.Articles[0].ID)
	assert.Equal(t, "Getting Started Guide", page.Articles[0].Title)
}

func TestArticleService_List_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "10", r.URL.Query().Get("page[size]"))
		assert.Equal(t, "updated_at", r.URL.Query().Get("sort_by"))
		assert.Equal(t, "asc", r.URL.Query().Get("sort_order"))
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
	require.NoError(t, err)
}

func TestArticleService_Get(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/article.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/help_center/articles/101", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	result, err := svc.Get(context.Background(), 101)
	require.NoError(t, err)
	assert.Equal(t, int64(101), result.Article.ID)
	assert.Equal(t, "Getting Started Guide", result.Article.Title)
	assert.True(t, result.Article.Promoted, "expected promoted to be true")
}

func TestArticleService_Get_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"RecordNotFound"}`))
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	_, err := svc.Get(context.Background(), 999)
	require.Error(t, err)

	appErr, ok := err.(*types.AppError)
	require.True(t, ok, "expected AppError, got %T", err)
	assert.Equal(t, types.ExitNotFound, appErr.ExitCode)
}

func TestArticleService_Search(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/articles_search.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/help_center/articles/search", r.URL.Path)
		assert.Equal(t, "password reset", r.URL.Query().Get("query"))
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewArticleService(client)

	page, err := svc.Search(context.Background(), "password reset", nil)
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	assert.Equal(t, "Password Reset Instructions", page.Results[0].Title)
	assert.Equal(t, 1, page.Count)
}
