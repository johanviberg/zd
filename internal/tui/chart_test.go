package tui

import (
	"strings"
	"testing"

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
	if result != "" {
		t.Errorf("expected empty string for nil items, got %q", result)
	}
}

func TestRenderStatusChart_SingleItem(t *testing.T) {
	tickets := makeTickets("open")
	result := renderStatusChart(tickets, 80, chartHeight)
	if result != "" {
		t.Errorf("expected empty string for single item, got %q", result)
	}
}

func TestRenderStatusChart_NarrowWidth(t *testing.T) {
	tickets := makeTickets("open", "pending")
	result := renderStatusChart(tickets, 25, chartHeight)
	if result != "" {
		t.Errorf("expected empty string for narrow width, got %q", result)
	}
}

func TestRenderStatusChart_TwoStatuses(t *testing.T) {
	tickets := makeTickets("open", "open", "pending")
	result := renderStatusChart(tickets, 80, chartHeight)

	if result == "" {
		t.Fatal("expected non-empty chart")
	}
	if !strings.Contains(result, "Status Distribution") {
		t.Error("expected chart title")
	}
	if !strings.Contains(result, "open") {
		t.Error("expected 'open' label")
	}
	if !strings.Contains(result, "pend") {
		t.Error("expected 'pend' label")
	}
	if !strings.Contains(result, "██") {
		t.Error("expected bar characters")
	}
}

func TestRenderStatusChart_AllStatuses(t *testing.T) {
	tickets := makeTickets("new", "open", "open", "pending", "hold", "solved", "solved", "solved", "closed", "closed")
	result := renderStatusChart(tickets, 80, chartHeight)

	if result == "" {
		t.Fatal("expected non-empty chart")
	}
	for _, label := range []string{"new", "open", "pend", "hold", "solv", "clos"} {
		if !strings.Contains(result, label) {
			t.Errorf("expected label %q in chart", label)
		}
	}
}

func TestRenderStatusChart_CountsCorrect(t *testing.T) {
	tickets := makeTickets("open", "open", "open", "pending")
	result := renderStatusChart(tickets, 80, chartHeight)

	// Should show count "3" for open and "1" for pending
	if !strings.Contains(result, "3") {
		t.Error("expected count '3' for open tickets")
	}
	if !strings.Contains(result, "1") {
		t.Error("expected count '1' for pending tickets")
	}
}

func TestRenderStatusChart_UnknownStatusIgnored(t *testing.T) {
	tickets := makeTickets("open", "unknown_status", "open")
	result := renderStatusChart(tickets, 80, chartHeight)

	// Unknown status has no entry in statusOrder, so it won't appear
	if strings.Contains(result, "unknown") {
		t.Error("unknown status should not appear in chart")
	}
	// But open should still be there — though only 2 items of known status,
	// the total items slice is 3 so the chart should render
	if !strings.Contains(result, "open") {
		t.Error("expected 'open' label in chart")
	}
}
