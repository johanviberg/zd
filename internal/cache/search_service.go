package cache

import (
	"context"
	"fmt"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

// CachedSearchService wraps a zendesk.SearchService with a TTL cache.
// It shares a Cache instance with CachedTicketService so ticket mutations
// also invalidate search results.
type CachedSearchService struct {
	delegate zendesk.SearchService
	cache    *Cache
}

// NewCachedSearchService creates a caching decorator around the given SearchService.
func NewCachedSearchService(delegate zendesk.SearchService, c *Cache) *CachedSearchService {
	return &CachedSearchService{delegate: delegate, cache: c}
}

func (s *CachedSearchService) Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchPage, error) {
	cursor, sortBy, sortOrder, include := "", "", "", ""
	limit := 0
	export := false
	if opts != nil {
		cursor = opts.Cursor
		sortBy = opts.SortBy
		sortOrder = opts.SortOrder
		include = opts.Include
		limit = opts.Limit
		export = opts.Export
	}
	key := fmt.Sprintf("search:%s:%d:%s:%s:%s:%s:%t", query, limit, cursor, sortBy, sortOrder, include, export)

	if v, ok := s.cache.Get(key); ok {
		return v.(*types.SearchPage), nil
	}

	result, err := s.delegate.Search(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, result)
	return result, nil
}
