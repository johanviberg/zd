package tui

import (
	"testing"
	"time"

	"github.com/johanviberg/zd/internal/types"
)

func boolP(b bool) *bool { return &b }

func TestBuildTimeline_CommentOnlyAudit(t *testing.T) {
	audits := []types.Audit{
		{
			ID: 1, TicketID: 1, AuthorID: 10,
			CreatedAt: time.Now(),
			Events: []types.AuditEvent{
				{ID: 1, Type: "Comment", Body: "Hello", Public: boolP(true), AuthorID: 10},
			},
		},
	}

	nodes := buildTimeline(audits)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if len(nodes[0].Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(nodes[0].Comments))
	}
	if len(nodes[0].Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(nodes[0].Changes))
	}
}

func TestBuildTimeline_ChangeOnlyAudit(t *testing.T) {
	audits := []types.Audit{
		{
			ID: 1, TicketID: 1, AuthorID: 10,
			CreatedAt: time.Now(),
			Events: []types.AuditEvent{
				{ID: 1, Type: "Change", FieldName: "status", Value: "open", PreviousValue: "new"},
			},
		},
	}

	nodes := buildTimeline(audits)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if len(nodes[0].Comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(nodes[0].Comments))
	}
	if len(nodes[0].Changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(nodes[0].Changes))
	}
}

func TestBuildTimeline_MixedAudit(t *testing.T) {
	audits := []types.Audit{
		{
			ID: 1, TicketID: 1, AuthorID: 10,
			CreatedAt: time.Now(),
			Events: []types.AuditEvent{
				{ID: 1, Type: "Comment", Body: "Fixed it", Public: boolP(true), AuthorID: 10},
				{ID: 2, Type: "Change", FieldName: "status", Value: "solved", PreviousValue: "open"},
				{ID: 3, Type: "Change", FieldName: "priority", Value: "high", PreviousValue: "normal"},
			},
		},
	}

	nodes := buildTimeline(audits)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if len(nodes[0].Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(nodes[0].Comments))
	}
	if len(nodes[0].Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(nodes[0].Changes))
	}
}

func TestBuildTimeline_FiltersIrrelevantEvents(t *testing.T) {
	audits := []types.Audit{
		{
			ID: 1, TicketID: 1, AuthorID: 10,
			CreatedAt: time.Now(),
			Events: []types.AuditEvent{
				{ID: 1, Type: "Change", FieldName: "custom_field_123", Value: "foo"},
				{ID: 2, Type: "Comment", Body: ""}, // empty body
			},
		},
	}

	nodes := buildTimeline(audits)
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes for irrelevant events, got %d", len(nodes))
	}
}

func TestFilterCommentNodes(t *testing.T) {
	nodes := []TimelineNode{
		{Comments: []types.AuditEvent{{Type: "Comment", Body: "Hi"}}},
		{Changes: []types.AuditEvent{{Type: "Change", FieldName: "status"}}},
		{
			Comments: []types.AuditEvent{{Type: "Comment", Body: "Done"}},
			Changes:  []types.AuditEvent{{Type: "Change", FieldName: "status"}},
		},
	}

	filtered := filterCommentNodes(nodes)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 comment nodes, got %d", len(filtered))
	}
}

func TestRenderTimeline_NonEmpty(t *testing.T) {
	now := time.Now()
	audits := []types.Audit{
		{
			ID: 1, TicketID: 1, AuthorID: 10,
			CreatedAt: now.Add(-2 * time.Hour),
			Events: []types.AuditEvent{
				{ID: 1, Type: "Comment", Body: "First message", Public: boolP(true), AuthorID: 10},
			},
		},
		{
			ID: 2, TicketID: 1, AuthorID: 20,
			CreatedAt: now.Add(-1 * time.Hour),
			Events: []types.AuditEvent{
				{ID: 2, Type: "Change", FieldName: "status", Value: "open", PreviousValue: "new"},
			},
		},
	}

	nodes := buildTimeline(audits)
	users := map[int64]types.User{
		10: {ID: 10, Name: "Alice"},
		20: {ID: 20, Name: "Bob"},
	}

	result := renderTimeline(nodes, users, 60)
	if result == "" {
		t.Fatal("expected non-empty timeline render")
	}
	if len(result) < 20 {
		t.Errorf("timeline render too short: %q", result)
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text  string
		width int
		lines int
	}{
		{"short", 80, 1},
		{"hello world this is a test", 12, 3},
		{"line1\nline2", 80, 2},
		{"", 80, 1},
		{"https://example.com/very/long/url/that/exceeds/the/width/limit", 20, 4},
		{"before https://example.com/very/long/url/that/exceeds after", 20, 4},
	}

	for _, tt := range tests {
		result := wrapText(tt.text, tt.width)
		if len(result) != tt.lines {
			t.Errorf("wrapText(%q, %d) = %d lines, want %d", tt.text, tt.width, len(result), tt.lines)
		}
	}
}
