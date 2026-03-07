package demo

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/johanviberg/zd/internal/types"
)

type TicketService struct {
	store *Store
}

func NewTicketService(store *Store) *TicketService {
	return &TicketService{store: store}
}

func (s *TicketService) List(ctx context.Context, opts *types.ListTicketsOptions) (*types.TicketPage, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	// Collect and filter
	var tickets []types.Ticket
	for _, t := range s.store.Tickets {
		if opts != nil {
			if opts.Status != "" && t.Status != opts.Status {
				continue
			}
			if opts.Assignee != 0 && t.AssigneeID != opts.Assignee {
				continue
			}
			if opts.Group != 0 && t.GroupID != opts.Group {
				continue
			}
		}
		tickets = append(tickets, t)
	}

	// Sort
	sortField := "updated_at"
	sortOrder := "desc"
	if opts != nil {
		if opts.Sort != "" {
			sortField = opts.Sort
		}
		if opts.SortOrder != "" {
			sortOrder = opts.SortOrder
		}
	}
	sortTickets(tickets, sortField, sortOrder)

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
	hasMore := end < len(tickets)
	if end > len(tickets) {
		end = len(tickets)
	}
	page := tickets[offset:end]

	var afterCursor string
	if hasMore {
		afterCursor = encodeCursor(end)
	}

	result := &types.TicketPage{
		Tickets: page,
		Meta: types.PageMeta{
			HasMore:     hasMore,
			AfterCursor: afterCursor,
		},
		Count: len(tickets),
	}

	if opts != nil && strings.Contains(opts.Include, "users") {
		result.Users = s.store.CollectUsers(page)
	}

	return result, nil
}

func (s *TicketService) Get(ctx context.Context, id int64, opts *types.GetTicketOptions) (*types.TicketResult, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	t, ok := s.store.Tickets[id]
	if !ok {
		return nil, types.NewNotFoundError(fmt.Sprintf("ticket %d not found", id))
	}

	result := &types.TicketResult{Ticket: t}

	if opts != nil && strings.Contains(opts.Include, "users") {
		result.Users = s.store.CollectUsers([]types.Ticket{t})
	}

	return result, nil
}

func (s *TicketService) Create(ctx context.Context, req *types.CreateTicketRequest) (*types.Ticket, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	id := s.store.NextID()
	now := nowUTC()

	t := types.Ticket{
		ID:        id,
		URL:       TicketURL(id),
		Subject:   req.Subject,
		Status:    "new",
		Priority:  req.Priority,
		Type:      req.Type,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if req.Status != "" {
		t.Status = req.Status
	}
	if req.AssigneeID != 0 {
		t.AssigneeID = req.AssigneeID
	}
	if req.GroupID != 0 {
		t.GroupID = req.GroupID
	}

	s.store.Tickets[id] = t

	if req.Comment.Body != "" {
		pub := true
		s.store.Comments[id] = []types.Comment{{
			ID:        id*100 + 1,
			Body:      req.Comment.Body,
			Public:    &pub,
			AuthorID:  1001,
			CreatedAt: now,
		}}
	}

	return &t, nil
}

func (s *TicketService) Update(ctx context.Context, id int64, req *types.UpdateTicketRequest) (*types.Ticket, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	t, ok := s.store.Tickets[id]
	if !ok {
		return nil, types.NewNotFoundError(fmt.Sprintf("ticket %d not found", id))
	}

	now := nowUTC()

	if req.Subject != "" {
		t.Subject = req.Subject
	}
	if req.Status != "" {
		t.Status = req.Status
	}
	if req.Priority != "" {
		t.Priority = req.Priority
	}
	if req.Tags != nil {
		t.Tags = req.Tags
	}
	if req.AssigneeID != nil {
		t.AssigneeID = *req.AssigneeID
	}
	if req.GroupID != nil {
		t.GroupID = *req.GroupID
	}
	t.UpdatedAt = now

	s.store.Tickets[id] = t

	if req.Comment != nil && req.Comment.Body != "" {
		comments := s.store.Comments[id]
		newID := id*100 + int64(len(comments)+1)
		pub := true
		if req.Comment.Public != nil {
			pub = *req.Comment.Public
		}
		comments = append(comments, types.Comment{
			ID:        newID,
			Body:      req.Comment.Body,
			Public:    &pub,
			AuthorID:  1001,
			CreatedAt: now,
		})
		s.store.Comments[id] = comments
	}

	return &t, nil
}

func (s *TicketService) Delete(ctx context.Context, id int64) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	if _, ok := s.store.Tickets[id]; !ok {
		return types.NewNotFoundError(fmt.Sprintf("ticket %d not found", id))
	}
	delete(s.store.Tickets, id)
	delete(s.store.Comments, id)
	return nil
}

func (s *TicketService) ListComments(ctx context.Context, ticketID int64, opts *types.ListCommentsOptions) (*types.CommentPage, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	if _, ok := s.store.Tickets[ticketID]; !ok {
		return nil, types.NewNotFoundError(fmt.Sprintf("ticket %d not found", ticketID))
	}

	comments := s.store.Comments[ticketID]

	// Sort order
	if opts != nil && opts.SortOrder == "desc" {
		sorted := make([]types.Comment, len(comments))
		copy(sorted, comments)
		for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}
		comments = sorted
	}

	// Pagination
	limit := 25
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}
	offset := 0
	if opts != nil && opts.Cursor != "" {
		offset = decodeCursor(opts.Cursor)
	}

	end := offset + limit
	hasMore := end < len(comments)
	if end > len(comments) {
		end = len(comments)
	}
	page := comments[offset:end]

	var afterCursor string
	if hasMore {
		afterCursor = encodeCursor(end)
	}

	result := &types.CommentPage{
		Comments: page,
		Meta: types.PageMeta{
			HasMore:     hasMore,
			AfterCursor: afterCursor,
		},
	}

	if opts != nil && strings.Contains(opts.Include, "users") {
		result.Users = s.store.CollectCommentUsers(page)
	}

	return result, nil
}

func sortTickets(tickets []types.Ticket, field, order string) {
	sort.SliceStable(tickets, func(i, j int) bool {
		var less bool
		switch field {
		case "created_at":
			less = tickets[i].CreatedAt.Before(tickets[j].CreatedAt)
		case "status":
			less = statusRank(tickets[i].Status) < statusRank(tickets[j].Status)
		case "priority":
			less = priorityRank(tickets[i].Priority) < priorityRank(tickets[j].Priority)
		default: // updated_at
			less = tickets[i].UpdatedAt.Before(tickets[j].UpdatedAt)
		}
		if order == "desc" {
			return !less
		}
		return less
	})
}

func statusRank(s string) int {
	switch s {
	case "new":
		return 0
	case "open":
		return 1
	case "pending":
		return 2
	case "hold":
		return 3
	case "solved":
		return 4
	default:
		return 5
	}
}

func priorityRank(p string) int {
	switch p {
	case "urgent":
		return 0
	case "high":
		return 1
	case "normal":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(cursor string) int {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(b))
	if err != nil {
		return 0
	}
	return n
}

func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
