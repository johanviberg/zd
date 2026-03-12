package cache

import (
	"context"
	"fmt"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

// CachedTicketService wraps a zendesk.TicketService with a TTL cache.
// Read operations check the cache first; mutations pass through and invalidate.
type CachedTicketService struct {
	delegate zendesk.TicketService
	cache    *Cache
}

// NewCachedTicketService creates a caching decorator around the given TicketService.
// The provided Cache instance should be shared with CachedSearchService so that
// ticket mutations also invalidate search results.
func NewCachedTicketService(delegate zendesk.TicketService, c *Cache) *CachedTicketService {
	return &CachedTicketService{delegate: delegate, cache: c}
}

func (s *CachedTicketService) Get(ctx context.Context, id int64, opts *types.GetTicketOptions) (*types.TicketResult, error) {
	include := ""
	if opts != nil {
		include = opts.Include
	}
	key := fmt.Sprintf("ticket:get:%d:%s", id, include)

	if v, ok := s.cache.Get(key); ok {
		return v.(*types.TicketResult), nil
	}

	result, err := s.delegate.Get(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, result)
	return result, nil
}

func (s *CachedTicketService) List(ctx context.Context, opts *types.ListTicketsOptions) (*types.TicketPage, error) {
	key := fmt.Sprintf("ticket:list:%d:%s:%s:%s:%d:%d:%s",
		opts.Limit, opts.Cursor, opts.Sort, opts.SortOrder,
		opts.Assignee, opts.Group, opts.Include)

	if v, ok := s.cache.Get(key); ok {
		return v.(*types.TicketPage), nil
	}

	result, err := s.delegate.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, result)
	return result, nil
}

func (s *CachedTicketService) ListAudits(ctx context.Context, ticketID int64, opts *types.ListAuditsOptions) (*types.AuditPage, error) {
	sortOrder, cursor, include := "", "", ""
	limit := 0
	if opts != nil {
		sortOrder = opts.SortOrder
		cursor = opts.Cursor
		include = opts.Include
		limit = opts.Limit
	}
	key := fmt.Sprintf("ticket:audits:%d:%s:%s:%s:%d", ticketID, sortOrder, cursor, include, limit)

	if v, ok := s.cache.Get(key); ok {
		return v.(*types.AuditPage), nil
	}

	result, err := s.delegate.ListAudits(ctx, ticketID, opts)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, result)
	return result, nil
}

func (s *CachedTicketService) ListComments(ctx context.Context, ticketID int64, opts *types.ListCommentsOptions) (*types.CommentPage, error) {
	sortOrder, cursor, include := "", "", ""
	limit := 0
	if opts != nil {
		sortOrder = opts.SortOrder
		cursor = opts.Cursor
		include = opts.Include
		limit = opts.Limit
	}
	key := fmt.Sprintf("ticket:comments:%d:%s:%s:%s:%d", ticketID, sortOrder, cursor, include, limit)

	if v, ok := s.cache.Get(key); ok {
		return v.(*types.CommentPage), nil
	}

	result, err := s.delegate.ListComments(ctx, ticketID, opts)
	if err != nil {
		return nil, err
	}
	s.cache.Set(key, result)
	return result, nil
}

func (s *CachedTicketService) Create(ctx context.Context, req *types.CreateTicketRequest) (*types.Ticket, error) {
	result, err := s.delegate.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	s.cache.Invalidate("ticket:list:", "search:")
	return result, nil
}

func (s *CachedTicketService) Update(ctx context.Context, id int64, req *types.UpdateTicketRequest) (*types.Ticket, error) {
	result, err := s.delegate.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	s.cache.Invalidate(
		fmt.Sprintf("ticket:get:%d:", id),
		fmt.Sprintf("ticket:audits:%d:", id),
		fmt.Sprintf("ticket:comments:%d:", id),
		"ticket:list:",
		"search:",
	)
	return result, nil
}

func (s *CachedTicketService) Delete(ctx context.Context, id int64) error {
	err := s.delegate.Delete(ctx, id)
	if err != nil {
		return err
	}
	s.cache.Invalidate(
		fmt.Sprintf("ticket:get:%d:", id),
		fmt.Sprintf("ticket:audits:%d:", id),
		fmt.Sprintf("ticket:comments:%d:", id),
		"ticket:list:",
		"search:",
	)
	return nil
}
