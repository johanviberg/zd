package cmd

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johanviberg/zd/internal/types"
)

// --- Mock services ---

type mockTicketService struct {
	tickets  []types.Ticket
	users    []types.User
	comments []types.Comment
	created  *types.Ticket
	updated  *types.Ticket
	deleted  int64

	listOpts    *types.ListTicketsOptions
	getOpts     *types.GetTicketOptions
	createReq   *types.CreateTicketRequest
	updateReq   *types.UpdateTicketRequest
	commentOpts *types.ListCommentsOptions
	commentMeta types.PageMeta
	listMeta    types.PageMeta
}

func (m *mockTicketService) List(_ context.Context, opts *types.ListTicketsOptions) (*types.TicketPage, error) {
	m.listOpts = opts
	return &types.TicketPage{
		Tickets: m.tickets,
		Users:   m.users,
		Meta:    m.listMeta,
	}, nil
}

func (m *mockTicketService) Get(_ context.Context, id int64, opts *types.GetTicketOptions) (*types.TicketResult, error) {
	m.getOpts = opts
	if len(m.tickets) > 0 {
		return &types.TicketResult{Ticket: m.tickets[0], Users: m.users}, nil
	}
	return &types.TicketResult{}, nil
}

func (m *mockTicketService) Create(_ context.Context, req *types.CreateTicketRequest) (*types.Ticket, error) {
	m.createReq = req
	if m.created != nil {
		return m.created, nil
	}
	return &types.Ticket{ID: 999, Subject: req.Subject, Status: "new"}, nil
}

func (m *mockTicketService) Update(_ context.Context, id int64, req *types.UpdateTicketRequest) (*types.Ticket, error) {
	m.updateReq = req
	if m.updated != nil {
		return m.updated, nil
	}
	return &types.Ticket{ID: id, Subject: req.Subject, Status: req.Status}, nil
}

func (m *mockTicketService) Delete(_ context.Context, id int64) error {
	m.deleted = id
	return nil
}

func (m *mockTicketService) ListComments(_ context.Context, ticketID int64, opts *types.ListCommentsOptions) (*types.CommentPage, error) {
	m.commentOpts = opts
	return &types.CommentPage{
		Comments: m.comments,
		Users:    m.users,
		Meta:     m.commentMeta,
	}, nil
}

func (m *mockTicketService) ListAudits(_ context.Context, ticketID int64, opts *types.ListAuditsOptions) (*types.AuditPage, error) {
	return &types.AuditPage{}, nil
}

type mockSearchService struct {
	results []types.SearchResult
	users   []types.User
	count   int
	opts    *types.SearchOptions
	query   string
	meta    types.PageMeta
}

func (m *mockSearchService) Search(_ context.Context, query string, opts *types.SearchOptions) (*types.SearchPage, error) {
	m.query = query
	m.opts = opts
	return &types.SearchPage{
		Results: m.results,
		Users:   m.users,
		Count:   m.count,
		Meta:    m.meta,
	}, nil
}

type mockArticleService struct {
	articles   []types.Article
	article    *types.Article
	searchRes  []types.Article
	listMeta   types.PageMeta
	searchMeta types.PageMeta
	count      int
}

func (m *mockArticleService) List(_ context.Context, opts *types.ListArticlesOptions) (*types.ArticlePage, error) {
	return &types.ArticlePage{
		Articles: m.articles,
		Meta:     m.listMeta,
	}, nil
}

func (m *mockArticleService) Get(_ context.Context, id int64) (*types.ArticleResult, error) {
	if m.article != nil {
		return &types.ArticleResult{Article: *m.article}, nil
	}
	return &types.ArticleResult{Article: types.Article{ID: id}}, nil
}

func (m *mockArticleService) Search(_ context.Context, query string, opts *types.SearchArticlesOptions) (*types.ArticleSearchPage, error) {
	return &types.ArticleSearchPage{
		Results: m.searchRes,
		Count:   m.count,
		Meta:    m.searchMeta,
	}, nil
}

// --- Test helpers ---

func setupMCPServer(t *testing.T, ticketSvc *mockTicketService, searchSvc *mockSearchService, articleSvc *mockArticleService) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "zd-test",
		Version: "test",
	}, nil)

	if ticketSvc != nil {
		registerTicketTools(server, ticketSvc)
	}
	if searchSvc != nil {
		registerSearchTools(server, searchSvc)
	}
	if articleSvc != nil {
		registerArticleTools(server, articleSvc)
	}

	ct, st := mcp.NewInMemoryTransports()
	_, err := server.Connect(context.Background(), st, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "zd-test-client",
		Version: "test",
	}, nil)
	cs, err := client.Connect(context.Background(), ct, nil)
	require.NoError(t, err)

	t.Cleanup(func() { cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)
	return result
}

// callToolExpectErr calls a tool and expects a protocol-level error (e.g. schema validation).
func callToolExpectErr(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) error {
	t.Helper()
	_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.Error(t, err)
	return err
}

func resultJSON(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	require.NotEmpty(t, result.Content)
	tc, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected TextContent, got %T", result.Content[0])
	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(tc.Text), &data))
	return data
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, result.Content)
	tc, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected TextContent")
	return tc.Text
}

// --- Ticket tool tests ---

func TestMCPListTickets(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := &mockTicketService{
		tickets: []types.Ticket{
			{ID: 1, Subject: "First", Status: "open", UpdatedAt: now},
			{ID: 2, Subject: "Second", Status: "pending", UpdatedAt: now},
		},
		users: []types.User{
			{ID: 10, Name: "Alice", Email: "alice@example.com"},
		},
	}

	cs := setupMCPServer(t, svc, nil, nil)

	t.Run("basic list", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_list_tickets", map[string]any{})
		data := resultJSON(t, result)

		tickets, ok := data["tickets"].([]any)
		require.True(t, ok)
		assert.Len(t, tickets, 2)
		assert.False(t, result.IsError)
	})

	t.Run("with filters", func(t *testing.T) {
		callTool(t, cs, "zendesk_list_tickets", map[string]any{
			"status":      "open",
			"assignee_id": float64(42),
			"group_id":    float64(7),
		})
		assert.Equal(t, "open", svc.listOpts.Status)
		assert.Equal(t, int64(42), svc.listOpts.Assignee)
		assert.Equal(t, int64(7), svc.listOpts.Group)
	})

	t.Run("pagination and sort", func(t *testing.T) {
		callTool(t, cs, "zendesk_list_tickets", map[string]any{
			"cursor":     "abc123",
			"sort":       "created_at",
			"sort_order": "asc",
		})
		assert.Equal(t, "abc123", svc.listOpts.Cursor)
		assert.Equal(t, "created_at", svc.listOpts.Sort)
		assert.Equal(t, "asc", svc.listOpts.SortOrder)
	})

	t.Run("limit clamping", func(t *testing.T) {
		callTool(t, cs, "zendesk_list_tickets", map[string]any{"limit": float64(200)})
		assert.Equal(t, 100, svc.listOpts.Limit)

		callTool(t, cs, "zendesk_list_tickets", map[string]any{})
		assert.Equal(t, 20, svc.listOpts.Limit)
	})

	t.Run("has_more pagination", func(t *testing.T) {
		svc.listMeta = types.PageMeta{HasMore: true, AfterCursor: "next123"}
		result := callTool(t, cs, "zendesk_list_tickets", map[string]any{})
		data := resultJSON(t, result)
		assert.Equal(t, true, data["has_more"])
		assert.Equal(t, "next123", data["next_cursor"])
		svc.listMeta = types.PageMeta{} // reset
	})
}

func TestMCPShowTicket(t *testing.T) {
	svc := &mockTicketService{
		tickets: []types.Ticket{
			{ID: 42, Subject: "Test Ticket", Status: "open", RequesterID: 10},
		},
		users: []types.User{
			{ID: 10, Name: "Bob", Email: "bob@example.com"},
		},
	}
	cs := setupMCPServer(t, svc, nil, nil)

	t.Run("success", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_show_ticket", map[string]any{"id": float64(42)})
		data := resultJSON(t, result)
		assert.Equal(t, float64(42), data["id"])
		assert.Equal(t, "Bob", data["requester_name"])
		assert.False(t, result.IsError)
	})

	t.Run("missing id", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_show_ticket", map[string]any{})
		assert.Contains(t, err.Error(), "id")
	})
}

func TestMCPCreateTicket(t *testing.T) {
	svc := &mockTicketService{}
	cs := setupMCPServer(t, svc, nil, nil)

	t.Run("full create", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_create_ticket", map[string]any{
			"subject":         "New ticket",
			"comment":         "Body text",
			"priority":        "high",
			"type":            "incident",
			"status":          "open",
			"assignee_id":     float64(5),
			"group_id":        float64(3),
			"tags":            []any{"billing", "urgent"},
			"requester_email": "user@example.com",
			"requester_name":  "Test User",
			"custom_fields": []any{
				map[string]any{"id": float64(123), "value": "custom-val"},
			},
		})
		assert.False(t, result.IsError)

		req := svc.createReq
		assert.Equal(t, "New ticket", req.Subject)
		assert.Equal(t, "Body text", req.Comment.Body)
		assert.Equal(t, "high", req.Priority)
		assert.Equal(t, "incident", req.Type)
		assert.Equal(t, "open", req.Status)
		assert.Equal(t, int64(5), req.AssigneeID)
		assert.Equal(t, int64(3), req.GroupID)
		assert.Equal(t, []string{"billing", "urgent"}, req.Tags)
		assert.Equal(t, "user@example.com", req.RequesterEmail)
		assert.Equal(t, "Test User", req.RequesterName)
		require.Len(t, req.CustomFields, 1)
		assert.Equal(t, int64(123), req.CustomFields[0].ID)
	})

	t.Run("missing subject", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_create_ticket", map[string]any{
			"comment": "body",
		})
		assert.Contains(t, err.Error(), "subject")
	})

	t.Run("missing comment", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_create_ticket", map[string]any{
			"subject": "title",
		})
		assert.Contains(t, err.Error(), "comment")
	})
}

func TestMCPUpdateTicket(t *testing.T) {
	svc := &mockTicketService{}
	cs := setupMCPServer(t, svc, nil, nil)

	t.Run("comment and status", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_update_ticket", map[string]any{
			"id":      float64(1),
			"comment": "Adding a note",
			"status":  "pending",
		})
		assert.False(t, result.IsError)

		req := svc.updateReq
		assert.Equal(t, "pending", req.Status)
		require.NotNil(t, req.Comment)
		assert.Equal(t, "Adding a note", req.Comment.Body)
		assert.True(t, *req.Comment.Public)
	})

	t.Run("internal note", func(t *testing.T) {
		boolFalse := false
		callTool(t, cs, "zendesk_update_ticket", map[string]any{
			"id":      float64(1),
			"comment": "Internal note",
			"public":  boolFalse,
		})
		assert.False(t, *svc.updateReq.Comment.Public)
	})

	t.Run("subject and assignee", func(t *testing.T) {
		callTool(t, cs, "zendesk_update_ticket", map[string]any{
			"id":          float64(1),
			"subject":     "Updated subject",
			"assignee_id": float64(42),
			"group_id":    float64(7),
		})
		assert.Equal(t, "Updated subject", svc.updateReq.Subject)
		require.NotNil(t, svc.updateReq.AssigneeID)
		assert.Equal(t, int64(42), *svc.updateReq.AssigneeID)
		require.NotNil(t, svc.updateReq.GroupID)
		assert.Equal(t, int64(7), *svc.updateReq.GroupID)
	})

	t.Run("tags and custom fields", func(t *testing.T) {
		callTool(t, cs, "zendesk_update_ticket", map[string]any{
			"id":       float64(1),
			"tags":     []any{"a", "b"},
			"add_tags": []any{"c"},
			"custom_fields": []any{
				map[string]any{"id": float64(100), "value": "val"},
			},
			"safe_update": true,
		})
		assert.Equal(t, []string{"a", "b"}, svc.updateReq.Tags)
		assert.Equal(t, []string{"c"}, svc.updateReq.AddTags)
		assert.True(t, svc.updateReq.SafeUpdate)
		require.Len(t, svc.updateReq.CustomFields, 1)
	})

	t.Run("missing id", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_update_ticket", map[string]any{
			"status": "open",
		})
		assert.Contains(t, err.Error(), "id")
	})
}

func TestMCPDeleteTicket(t *testing.T) {
	svc := &mockTicketService{}
	cs := setupMCPServer(t, svc, nil, nil)

	t.Run("success", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_delete_ticket", map[string]any{"id": float64(42)})
		assert.False(t, result.IsError)
		data := resultJSON(t, result)
		assert.Equal(t, true, data["deleted"])
		assert.Equal(t, float64(42), data["ticket_id"])
		assert.Equal(t, int64(42), svc.deleted)
	})

	t.Run("missing id", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_delete_ticket", map[string]any{})
		assert.Contains(t, err.Error(), "id")
	})
}

// --- Comment tool tests ---

func TestMCPListComments(t *testing.T) {
	svc := &mockTicketService{
		comments: []types.Comment{
			{ID: 1, Body: "First comment", AuthorID: 10},
			{ID: 2, Body: "Second comment", AuthorID: 20},
		},
		users: []types.User{
			{ID: 10, Name: "Alice", Email: "alice@example.com"},
			{ID: 20, Name: "Bob", Email: "bob@example.com"},
		},
	}
	cs := setupMCPServer(t, svc, nil, nil)

	t.Run("with user enrichment", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_list_comments", map[string]any{
			"ticket_id": float64(42),
		})
		assert.False(t, result.IsError)
		data := resultJSON(t, result)
		comments, ok := data["comments"].([]any)
		require.True(t, ok)
		assert.Len(t, comments, 2)

		first := comments[0].(map[string]any)
		assert.Equal(t, "Alice", first["author_name"])
	})

	t.Run("passes options", func(t *testing.T) {
		callTool(t, cs, "zendesk_list_comments", map[string]any{
			"ticket_id":  float64(42),
			"limit":      float64(50),
			"cursor":     "cur123",
			"sort_order": "desc",
		})
		assert.Equal(t, 50, svc.commentOpts.Limit)
		assert.Equal(t, "cur123", svc.commentOpts.Cursor)
		assert.Equal(t, "desc", svc.commentOpts.SortOrder)
		assert.Equal(t, "users", svc.commentOpts.Include)
	})

	t.Run("missing ticket_id", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_list_comments", map[string]any{})
		assert.Contains(t, err.Error(), "ticket_id")
	})

	t.Run("has_more pagination", func(t *testing.T) {
		svc.commentMeta = types.PageMeta{HasMore: true, AfterCursor: "cmt_next"}
		result := callTool(t, cs, "zendesk_list_comments", map[string]any{
			"ticket_id": float64(1),
		})
		data := resultJSON(t, result)
		assert.Equal(t, true, data["has_more"])
		assert.Equal(t, "cmt_next", data["next_cursor"])
	})
}

// --- Search tool tests ---

func TestMCPSearchTickets(t *testing.T) {
	svc := &mockSearchService{
		results: []types.SearchResult{
			{Ticket: types.Ticket{ID: 1, Subject: "Match", Status: "open"}},
		},
		count: 1,
	}
	cs := setupMCPServer(t, nil, svc, nil)

	t.Run("zendesk syntax passthrough", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_search_tickets", map[string]any{
			"query": "status:open priority:high",
		})
		assert.False(t, result.IsError)
		// Zendesk syntax should pass through NLQ unchanged
		assert.Equal(t, "status:open priority:high", svc.query)
		data := resultJSON(t, result)
		assert.Equal(t, float64(1), data["count"])
	})

	t.Run("NLQ translation", func(t *testing.T) {
		callTool(t, cs, "zendesk_search_tickets", map[string]any{
			"query": "urgent tickets",
		})
		// NLQ should translate "urgent tickets" to Zendesk syntax
		assert.Contains(t, svc.query, "priority:urgent")
	})

	t.Run("pagination and sort", func(t *testing.T) {
		callTool(t, cs, "zendesk_search_tickets", map[string]any{
			"query":      "status:open",
			"cursor":     "search_cur",
			"sort_by":    "created_at",
			"sort_order": "asc",
		})
		assert.Equal(t, "search_cur", svc.opts.Cursor)
		assert.Equal(t, "created_at", svc.opts.SortBy)
		assert.Equal(t, "asc", svc.opts.SortOrder)
	})

	t.Run("missing query", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_search_tickets", map[string]any{})
		assert.Contains(t, err.Error(), "query")
	})

	t.Run("has_more pagination", func(t *testing.T) {
		svc.meta = types.PageMeta{HasMore: true, AfterCursor: "s_next"}
		result := callTool(t, cs, "zendesk_search_tickets", map[string]any{
			"query": "status:open",
		})
		data := resultJSON(t, result)
		assert.Equal(t, true, data["has_more"])
		assert.Equal(t, "s_next", data["next_cursor"])
	})
}

// --- Article tool tests ---

func TestMCPListArticles(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := &mockArticleService{
		articles: []types.Article{
			{ID: 1, Title: "Getting Started", CreatedAt: now, UpdatedAt: now},
			{ID: 2, Title: "FAQ", CreatedAt: now, UpdatedAt: now},
		},
	}
	cs := setupMCPServer(t, nil, nil, svc)

	t.Run("basic list", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_list_articles", map[string]any{})
		assert.False(t, result.IsError)
		data := resultJSON(t, result)
		articles, ok := data["articles"].([]any)
		require.True(t, ok)
		assert.Len(t, articles, 2)
	})

	t.Run("has_more pagination", func(t *testing.T) {
		svc.listMeta = types.PageMeta{HasMore: true, AfterCursor: "art_next"}
		result := callTool(t, cs, "zendesk_list_articles", map[string]any{})
		data := resultJSON(t, result)
		assert.Equal(t, true, data["has_more"])
		assert.Equal(t, "art_next", data["next_cursor"])
	})
}

func TestMCPShowArticle(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := &mockArticleService{
		article: &types.Article{ID: 42, Title: "Test Article", Body: "<p>Content</p>", CreatedAt: now, UpdatedAt: now},
	}
	cs := setupMCPServer(t, nil, nil, svc)

	t.Run("success", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_show_article", map[string]any{"id": float64(42)})
		assert.False(t, result.IsError)
		data := resultJSON(t, result)
		assert.Equal(t, float64(42), data["id"])
		assert.Equal(t, "Test Article", data["title"])
	})

	t.Run("missing id", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_show_article", map[string]any{})
		assert.Contains(t, err.Error(), "id")
	})
}

func TestMCPSearchArticles(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := &mockArticleService{
		searchRes: []types.Article{
			{ID: 1, Title: "Result", CreatedAt: now, UpdatedAt: now},
		},
		count: 1,
	}
	cs := setupMCPServer(t, nil, nil, svc)

	t.Run("basic search", func(t *testing.T) {
		result := callTool(t, cs, "zendesk_search_articles", map[string]any{
			"query": "password reset",
		})
		assert.False(t, result.IsError)
		data := resultJSON(t, result)
		assert.Equal(t, float64(1), data["count"])
		articles, ok := data["articles"].([]any)
		require.True(t, ok)
		assert.Len(t, articles, 1)
	})

	t.Run("missing query", func(t *testing.T) {
		err := callToolExpectErr(t, cs, "zendesk_search_articles", map[string]any{})
		assert.Contains(t, err.Error(), "query")
	})
}

// --- Helper tests ---

func TestConvertCustomFields(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, convertCustomFields(nil))
	})

	t.Run("empty input", func(t *testing.T) {
		assert.Nil(t, convertCustomFields([]CustomFieldInput{}))
	})

	t.Run("converts fields", func(t *testing.T) {
		inputs := []CustomFieldInput{
			{ID: 1, Value: "text"},
			{ID: 2, Value: true},
		}
		fields := convertCustomFields(inputs)
		require.Len(t, fields, 2)
		assert.Equal(t, int64(1), fields[0].ID)
		assert.Equal(t, "text", fields[0].Value)
		assert.Equal(t, int64(2), fields[1].ID)
		assert.Equal(t, true, fields[1].Value)
	})
}
