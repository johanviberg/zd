package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.Len(t, nodes, 1, "expected 1 node")
	assert.Len(t, nodes[0].Comments, 1, "expected 1 comment")
	assert.Len(t, nodes[0].Changes, 0, "expected 0 changes")
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
	require.Len(t, nodes, 1, "expected 1 node")
	assert.Len(t, nodes[0].Comments, 0, "expected 0 comments")
	assert.Len(t, nodes[0].Changes, 1, "expected 1 change")
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
	require.Len(t, nodes, 1, "expected 1 node")
	assert.Len(t, nodes[0].Comments, 1, "expected 1 comment")
	assert.Len(t, nodes[0].Changes, 2, "expected 2 changes")
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
	require.Len(t, nodes, 0, "expected 0 nodes for irrelevant events")
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
	require.Len(t, filtered, 2, "expected 2 comment nodes")
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
	require.NotEmpty(t, result, "expected non-empty timeline render")
	assert.GreaterOrEqual(t, len(result), 20, "timeline render too short: %q", result)
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
		assert.Len(t, result, tt.lines, "wrapText(%q, %d)", tt.text, tt.width)
	}
}
