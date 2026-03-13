package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchService_Search(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/search_results.json")
	require.NoError(t, err, "reading fixture")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/api/v2/search"), "expected /api/v2/search, got %s", r.URL.Path)
		assert.Contains(t, r.URL.Query().Get("query"), "status:open")
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	client := testClient(t, handler)
	svc := NewSearchService(client)

	page, err := svc.Search(context.Background(), "status:open", &types.SearchOptions{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, page.Results, 2)
	assert.Equal(t, 2, page.Count)
}

func TestSearchService_SearchExport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/api/v2/search/export"), "expected /api/v2/search/export, got %s", r.URL.Path)
		assert.Equal(t, "ticket", r.URL.Query().Get("filter[type]"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[],"meta":{"has_more":false},"count":0}`))
	})

	client := testClient(t, handler)
	svc := NewSearchService(client)

	_, err := svc.Search(context.Background(), "status:open", &types.SearchOptions{Export: true})
	require.NoError(t, err)
}
