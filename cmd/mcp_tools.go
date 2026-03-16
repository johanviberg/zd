package cmd

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

func parseCollaboratorsFromStrings(values []string) []types.CollaboratorEntry {
	var entries []types.CollaboratorEntry
	for _, v := range values {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			entries = append(entries, types.CollaboratorEntry{UserID: id})
		} else {
			entries = append(entries, types.CollaboratorEntry{Email: v})
		}
	}
	return entries
}

// --- Input types ---

type ListTicketsInput struct {
	Status     string `json:"status,omitempty" jsonschema:"Filter by status (new/open/pending/hold/solved/closed)"`
	AssigneeID int64  `json:"assignee_id,omitempty" jsonschema:"Filter by assignee user ID"`
	GroupID    int64  `json:"group_id,omitempty" jsonschema:"Filter by group ID"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Max tickets to return (default 20, max 100)"`
}

type ShowTicketInput struct {
	ID int64 `json:"id" jsonschema:"Zendesk ticket ID (required)"`
}

type SearchTicketsInput struct {
	Query string `json:"query" jsonschema:"Zendesk search query using search syntax. Examples: 'status:open priority:high', 'tags:billing assignee:jane', 'created>2024-01-01 status:open', 'status:open OR status:pending'"`
	Limit int    `json:"limit,omitempty" jsonschema:"Max results to return (default 20, max 100)"`
}

type CreateTicketInput struct {
	Subject  string   `json:"subject" jsonschema:"Ticket subject (required)"`
	Comment  string   `json:"comment" jsonschema:"Ticket body/description (required)"`
	Priority string   `json:"priority,omitempty" jsonschema:"Priority: urgent, high, normal, low"`
	Tags     []string `json:"tags,omitempty" jsonschema:"Tags to add to the ticket"`
}

type UpdateTicketInput struct {
	ID         int64    `json:"id" jsonschema:"Zendesk ticket ID (required)"`
	Comment    string   `json:"comment,omitempty" jsonschema:"Add a comment to the ticket"`
	Public     *bool    `json:"public,omitempty" jsonschema:"Whether comment is public (default true). Set to false for internal notes"`
	Status     string   `json:"status,omitempty" jsonschema:"Set status: new, open, pending, hold, solved, closed"`
	Priority   string   `json:"priority,omitempty" jsonschema:"Set priority: urgent, high, normal, low"`
	AddTags    []string `json:"add_tags,omitempty" jsonschema:"Tags to add"`
	RemoveTags []string `json:"remove_tags,omitempty" jsonschema:"Tags to remove"`
	CC         []string `json:"cc,omitempty" jsonschema:"Add CCs to the comment (emails or user IDs). Ignored for internal notes"`
}

type DeleteTicketInput struct {
	ID int64 `json:"id" jsonschema:"Zendesk ticket ID (required)"`
}

// --- Registration ---

func registerTicketTools(server *mcp.Server, svc zendesk.TicketService) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "zendesk_list_tickets",
		Description: "List Zendesk tickets sorted by update time (newest first). " +
			"Returns ticket ID, subject, status, priority, and timestamps. " +
			"Use for a quick overview of the ticket queue.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListTicketsInput) (*mcp.CallToolResult, any, error) {
		limit := args.Limit
		if limit == 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		page, err := svc.List(ctx, &types.ListTicketsOptions{
			Status:    args.Status,
			Assignee:  args.AssigneeID,
			Group:     args.GroupID,
			Limit:     limit,
			Sort:      "updated_at",
			SortOrder: "desc",
			Include:   "users",
		})
		if err != nil {
			return errorResult(err), nil, nil
		}

		userMap := buildUserMap(page.Users)
		items := make([]any, len(page.Tickets))
		for i, t := range page.Tickets {
			items[i] = enrichTicket(t, userMap)
		}

		return jsonResult(items), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "zendesk_show_ticket",
		Description: "Show full details for a single Zendesk ticket including description, comments, " +
			"requester and assignee info. Always use this before updating or deleting a ticket.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ShowTicketInput) (*mcp.CallToolResult, any, error) {
		if args.ID == 0 {
			return errorResult(types.NewArgError("ticket id is required")), nil, nil
		}

		result, err := svc.Get(ctx, args.ID, &types.GetTicketOptions{
			Include: "users,groups,organizations",
		})
		if err != nil {
			return errorResult(err), nil, nil
		}

		var data any = result.Ticket
		if len(result.Users) > 0 {
			userMap := buildUserMap(result.Users)
			data = enrichTicket(result.Ticket, userMap)
		}

		return jsonResult(data), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "zendesk_create_ticket",
		Description: "Create a new Zendesk ticket. Requires subject and comment (body). " +
			"Returns the created ticket with its ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args CreateTicketInput) (*mcp.CallToolResult, any, error) {
		if args.Subject == "" {
			return errorResult(types.NewArgError("subject is required")), nil, nil
		}
		if args.Comment == "" {
			return errorResult(types.NewArgError("comment is required")), nil, nil
		}

		ticket, err := svc.Create(ctx, &types.CreateTicketRequest{
			Subject:  args.Subject,
			Comment:  types.Comment{Body: args.Comment},
			Priority: args.Priority,
			Tags:     args.Tags,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}

		return jsonResult(ticket), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "zendesk_update_ticket",
		Description: "Update an existing Zendesk ticket. Can add comments (public or internal notes), " +
			"change status/priority, manage tags, and add CCs. Always show the ticket first to understand context.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args UpdateTicketInput) (*mcp.CallToolResult, any, error) {
		if args.ID == 0 {
			return errorResult(types.NewArgError("ticket id is required")), nil, nil
		}

		updateReq := &types.UpdateTicketRequest{
			Status:     args.Status,
			Priority:   args.Priority,
			AddTags:    args.AddTags,
			RemoveTags: args.RemoveTags,
		}

		if args.Comment != "" {
			public := true
			if args.Public != nil {
				public = *args.Public
			}
			updateReq.Comment = &types.Comment{
				Body:   args.Comment,
				Public: &public,
			}
		}

		if len(args.CC) > 0 {
			updateReq.AdditionalCollaborators = parseCollaboratorsFromStrings(args.CC)
		}

		ticket, err := svc.Update(ctx, args.ID, updateReq)
		if err != nil {
			return errorResult(err), nil, nil
		}

		return jsonResult(ticket), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "zendesk_delete_ticket",
		Description: "Permanently delete a Zendesk ticket. This action cannot be undone. " +
			"Always show the ticket first to confirm you have the right one.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DeleteTicketInput) (*mcp.CallToolResult, any, error) {
		if args.ID == 0 {
			return errorResult(types.NewArgError("ticket id is required")), nil, nil
		}

		if err := svc.Delete(ctx, args.ID); err != nil {
			return errorResult(err), nil, nil
		}

		return jsonResult(map[string]any{
			"deleted":   true,
			"ticket_id": args.ID,
		}), nil, nil
	})
}

func registerSearchTools(server *mcp.Server, svc zendesk.SearchService) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "zendesk_search_tickets",
		Description: "Search Zendesk tickets using search syntax. " +
			"Query examples: 'status:open priority:high', 'tags:billing', " +
			"'assignee:jane status:pending', 'created>2024-01-01 status:open', " +
			"'status:open OR status:pending'. " +
			"Combine terms with spaces (AND) or 'OR'. " +
			"Fields: status, priority, type, assignee, requester, group, subject, description, tags, created, updated, organization.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchTicketsInput) (*mcp.CallToolResult, any, error) {
		if args.Query == "" {
			return errorResult(types.NewArgError("query is required")), nil, nil
		}

		limit := args.Limit
		if limit == 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		page, err := svc.Search(ctx, args.Query, &types.SearchOptions{
			Limit:     limit,
			SortOrder: "desc",
			Include:   "users",
		})
		if err != nil {
			return errorResult(err), nil, nil
		}

		userMap := buildUserMap(page.Users)
		var items []any
		for _, r := range page.Results {
			if r.Ticket.ID == 0 {
				continue // skip non-ticket results
			}
			items = append(items, enrichTicket(r.Ticket, userMap))
		}

		result := map[string]any{
			"count":   page.Count,
			"tickets": items,
		}

		return jsonResult(result), nil, nil
	})
}

// --- Helpers ---

func jsonResult(data any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return errorResult(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}

func errorResult(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
	result.SetError(err)
	return result
}
