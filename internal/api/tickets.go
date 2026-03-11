package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/johanviberg/zd/internal/types"
)

type TicketService struct {
	client *Client
}

func NewTicketService(client *Client) *TicketService {
	return &TicketService{client: client}
}

func (s *TicketService) List(ctx context.Context, opts *types.ListTicketsOptions) (*types.TicketPage, error) {
	path := "/api/v2/tickets"
	params := url.Values{}

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("page[size]", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("page[after]", opts.Cursor)
		}
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}
		if sort := params.Get("sort"); sort != "" {
			if opts.SortOrder == "desc" && sort[0] != '-' {
				params.Set("sort", "-"+sort)
			} else if opts.SortOrder == "asc" && sort[0] == '-' {
				params.Set("sort", sort[1:])
			}
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Assignee > 0 {
			params.Set("assignee_id", strconv.FormatInt(opts.Assignee, 10))
		}
		if opts.Group > 0 {
			params.Set("group_id", strconv.FormatInt(opts.Group, 10))
		}
		if opts.Include != "" {
			params.Set("include", opts.Include)
		}
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var page types.TicketPage
	if err := s.client.doJSON(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (s *TicketService) Get(ctx context.Context, id int64, opts *types.GetTicketOptions) (*types.TicketResult, error) {
	path := fmt.Sprintf("/api/v2/tickets/%d", id)

	if opts != nil && opts.Include != "" {
		path += "?include=" + url.QueryEscape(opts.Include)
	}

	var result types.TicketResult
	if err := s.client.doJSON(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *TicketService) Create(ctx context.Context, req *types.CreateTicketRequest) (*types.Ticket, error) {
	if req.RequesterEmail != "" || req.RequesterName != "" {
		req.Requester = &types.Requester{
			Email: req.RequesterEmail,
			Name:  req.RequesterName,
		}
	}

	body := struct {
		Ticket *types.CreateTicketRequest `json:"ticket"`
	}{Ticket: req}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var result struct {
		Ticket types.Ticket `json:"ticket"`
	}
	if err := s.client.doJSON(ctx, "POST", "/api/v2/tickets", bytes.NewReader(b), &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

func (s *TicketService) Update(ctx context.Context, id int64, req *types.UpdateTicketRequest) (*types.Ticket, error) {
	body := struct {
		Ticket *types.UpdateTicketRequest `json:"ticket"`
	}{Ticket: req}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	path := fmt.Sprintf("/api/v2/tickets/%d", id)

	var result struct {
		Ticket types.Ticket `json:"ticket"`
	}
	if err := s.client.doJSON(ctx, "PUT", path, bytes.NewReader(b), &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

func (s *TicketService) ListComments(ctx context.Context, ticketID int64, opts *types.ListCommentsOptions) (*types.CommentPage, error) {
	path := fmt.Sprintf("/api/v2/tickets/%d/comments", ticketID)
	params := url.Values{}

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("page[size]", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("page[after]", opts.Cursor)
		}
		if opts.SortOrder != "" {
			params.Set("sort_order", opts.SortOrder)
		}
		if opts.Include != "" {
			params.Set("include", opts.Include)
		}
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var page types.CommentPage
	if err := s.client.doJSON(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (s *TicketService) ListAudits(ctx context.Context, ticketID int64, opts *types.ListAuditsOptions) (*types.AuditPage, error) {
	path := fmt.Sprintf("/api/v2/tickets/%d/audits", ticketID)
	params := url.Values{}

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("page[size]", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("page[after]", opts.Cursor)
		}
		if opts.SortOrder != "" {
			params.Set("sort_order", opts.SortOrder)
		}
		if opts.Include != "" {
			params.Set("include", opts.Include)
		}
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var page types.AuditPage
	if err := s.client.doJSON(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (s *TicketService) Delete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/v2/tickets/%d", id)
	resp, err := s.client.do(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
