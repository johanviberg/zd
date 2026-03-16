package demo

import (
	"context"
	"strings"

	"github.com/johanviberg/zd/internal/types"
)

type SearchService struct {
	store *Store
}

func NewSearchService(store *Store) *SearchService {
	return &SearchService{store: store}
}

func (s *SearchService) Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchPage, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	terms := parseQuery(query)

	var results []types.SearchResult
	for _, t := range s.store.Tickets {
		if matchesTerms(t, terms) {
			results = append(results, types.SearchResult{
				Ticket:     t,
				ResultType: "ticket",
			})
		}
	}

	// Sort by updated_at desc by default
	sortField := "updated_at"
	sortOrder := "desc"
	if opts != nil {
		if opts.SortBy != "" {
			sortField = opts.SortBy
		}
		if opts.SortOrder != "" {
			sortOrder = opts.SortOrder
		}
	}
	sortResults(results, sortField, sortOrder)

	// Paginate
	limit := 25
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}
	offset := 0
	if opts != nil && opts.Cursor != "" {
		offset = decodeCursor(opts.Cursor)
	}

	end := offset + limit
	hasMore := end < len(results)
	if end > len(results) {
		end = len(results)
	}
	page := results[offset:end]

	var afterCursor string
	if hasMore {
		afterCursor = encodeCursor(end)
	}

	result := &types.SearchPage{
		Results: page,
		Meta: types.PageMeta{
			HasMore:     hasMore,
			AfterCursor: afterCursor,
		},
		Count: len(results),
	}

	if opts != nil && strings.Contains(opts.Include, "users") {
		tickets := make([]types.Ticket, len(page))
		for i, r := range page {
			tickets[i] = r.Ticket
		}
		result.Users = s.store.CollectUsers(tickets)
	}

	return result, nil
}

type searchTerm struct {
	field string // empty for bare words
	value string
}

func parseQuery(query string) []searchTerm {
	parts := strings.Fields(query)
	var terms []searchTerm
	for _, p := range parts {
		if idx := strings.Index(p, ":"); idx > 0 && idx < len(p)-1 {
			terms = append(terms, searchTerm{
				field: strings.ToLower(p[:idx]),
				value: strings.ToLower(p[idx+1:]),
			})
		} else {
			terms = append(terms, searchTerm{
				value: strings.ToLower(p),
			})
		}
	}
	return terms
}

func matchesTerms(t types.Ticket, terms []searchTerm) bool {
	for _, term := range terms {
		if !matchesTerm(t, term) {
			return false
		}
	}
	return true
}

func matchesTerm(t types.Ticket, term searchTerm) bool {
	if term.field != "" {
		switch term.field {
		case "status":
			return strings.EqualFold(t.Status, term.value)
		case "priority":
			return strings.EqualFold(t.Priority, term.value)
		case "type":
			return strings.EqualFold(t.Type, term.value)
		case "tags":
			for _, tag := range t.Tags {
				if strings.EqualFold(tag, term.value) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}

	// Bare word: substring match on subject, description, tags
	lower := term.value
	if strings.Contains(strings.ToLower(t.Subject), lower) {
		return true
	}
	if strings.Contains(strings.ToLower(t.Description), lower) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), lower) {
			return true
		}
	}
	return false
}

func sortResults(results []types.SearchResult, field, order string) {
	// Sort results in place using the embedded ticket fields
	tickets := make([]types.Ticket, len(results))
	for i := range results {
		tickets[i] = results[i].Ticket
	}

	// Build index mapping and sort it alongside tickets
	idx := make([]int, len(results))
	for i := range idx {
		idx[i] = i
	}

	// Use a copy to sort and rebuild
	sortTickets(tickets, field, order)

	// Build ID→result map for reconstruction
	idToResult := make(map[int64]types.SearchResult, len(results))
	for _, r := range results {
		idToResult[r.Ticket.ID] = r
	}
	for i, t := range tickets {
		results[i] = idToResult[t.ID]
	}
}
