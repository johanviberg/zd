package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johanviberg/zd/internal/types"
)

func makeTickets(statuses ...string) []types.Ticket {
	var tickets []types.Ticket
	for i, s := range statuses {
		tickets = append(tickets, types.Ticket{ID: int64(i + 1), Status: s})
	}
	return tickets
}

func TestRenderStatusChart_Empty(t *testing.T) {
	result := renderStatusChart(nil, 80, chartHeight)
	assert.Equal(t, "", result, "expected empty string for nil items")
}

func TestRenderStatusChart_SingleItem(t *testing.T) {
	tickets := makeTickets("open")
	result := renderStatusChart(tickets, 80, chartHeight)
	assert.Equal(t, "", result, "expected empty string for single item")
}

func TestRenderStatusChart_NarrowWidth(t *testing.T) {
	tickets := makeTickets("open", "pending")
	result := renderStatusChart(tickets, 25, chartHeight)
	assert.Equal(t, "", result, "expected empty string for narrow width")
}

func TestRenderStatusChart_TwoStatuses(t *testing.T) {
	tickets := makeTickets("open", "open", "pending")
	result := renderStatusChart(tickets, 80, chartHeight)

	require.NotEmpty(t, result, "expected non-empty chart")
	assert.Contains(t, result, "Status Distribution", "expected chart title")
	assert.Contains(t, result, "open", "expected 'open' label")
	assert.Contains(t, result, "pend", "expected 'pend' label")
	assert.Contains(t, result, "██", "expected bar characters")
}

func TestRenderStatusChart_AllStatuses(t *testing.T) {
	tickets := makeTickets("new", "open", "open", "pending", "hold", "solved", "solved", "solved", "closed", "closed")
	result := renderStatusChart(tickets, 80, chartHeight)

	require.NotEmpty(t, result, "expected non-empty chart")
	for _, label := range []string{"new", "open", "pend", "hold", "solv", "clos"} {
		assert.Contains(t, result, label, "expected label %q in chart", label)
	}
}

func TestRenderStatusChart_CountsCorrect(t *testing.T) {
	tickets := makeTickets("open", "open", "open", "pending")
	result := renderStatusChart(tickets, 80, chartHeight)

	// Should show count "3" for open and "1" for pending
	assert.Contains(t, result, "3", "expected count '3' for open tickets")
	assert.Contains(t, result, "1", "expected count '1' for pending tickets")
}

func TestRenderStatusChart_UnknownStatusIgnored(t *testing.T) {
	tickets := makeTickets("open", "unknown_status", "open")
	result := renderStatusChart(tickets, 80, chartHeight)

	// Unknown status has no entry in statusOrder, so it won't appear
	assert.NotContains(t, result, "unknown", "unknown status should not appear in chart")
	// But open should still be there — though only 2 items of known status,
	// the total items slice is 3 so the chart should render
	assert.Contains(t, result, "open", "expected 'open' label in chart")
}
