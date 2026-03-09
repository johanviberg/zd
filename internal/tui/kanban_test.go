package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johanviberg/zd/internal/types"
)

func makeTicket(id int64, status, priority, subject string) types.Ticket {
	return types.Ticket{
		ID:        id,
		Status:    status,
		Priority:  priority,
		Subject:   subject,
		UpdatedAt: time.Now(),
	}
}

func TestRebuildColumns(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(1, "new", "high", "New ticket"),
		makeTicket(2, "open", "urgent", "Open ticket 1"),
		makeTicket(3, "open", "normal", "Open ticket 2"),
		makeTicket(4, "pending", "low", "Pending ticket"),
		makeTicket(5, "solved", "normal", "Solved ticket"),
	}

	m.rebuildColumns(items)

	if len(m.columns[0]) != 1 {
		t.Errorf("expected 1 new ticket, got %d", len(m.columns[0]))
	}
	if len(m.columns[1]) != 2 {
		t.Errorf("expected 2 open tickets, got %d", len(m.columns[1]))
	}
	if len(m.columns[2]) != 1 {
		t.Errorf("expected 1 pending ticket, got %d", len(m.columns[2]))
	}
	if len(m.columns[3]) != 0 {
		t.Errorf("expected 0 hold tickets, got %d", len(m.columns[3]))
	}
	if len(m.columns[4]) != 1 {
		t.Errorf("expected 1 solved ticket, got %d", len(m.columns[4]))
	}
}

func TestRebuildColumns_Empty(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	m.rebuildColumns(nil)

	for i := 0; i < 5; i++ {
		if len(m.columns[i]) != 0 {
			t.Errorf("column %d: expected 0 tickets, got %d", i, len(m.columns[i]))
		}
	}
}

func TestRebuildColumns_UnknownStatus(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(1, "open", "normal", "Open"),
		makeTicket(2, "closed", "normal", "Closed"),
		makeTicket(3, "unknown", "normal", "Unknown"),
	}

	m.rebuildColumns(items)

	total := 0
	for i := 0; i < 5; i++ {
		total += len(m.columns[i])
	}
	if total != 1 {
		t.Errorf("expected 1 total ticket (only open), got %d", total)
	}
	if len(m.columns[1]) != 1 {
		t.Errorf("expected 1 open ticket, got %d", len(m.columns[1]))
	}
}

func TestNavigation_LeftRight(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(1, "new", "high", "New"),
		makeTicket(2, "open", "normal", "Open"),
		makeTicket(3, "solved", "normal", "Solved"),
	}

	m.rebuildColumns(items)
	// Cursor should start at first visible column with tickets (new = col 0)
	if m.col != 0 {
		t.Errorf("expected initial col 0, got %d", m.col)
	}

	// Move right — should skip empty columns
	m, _ = m.moveRight()
	activeCI := m.visibleCols[m.col]
	if activeCI != 1 {
		t.Errorf("after right: expected column index 1 (open), got %d", activeCI)
	}

	// Move right again — should skip pending (empty) and hold (empty)
	m, _ = m.moveRight()
	activeCI = m.visibleCols[m.col]
	if activeCI != 4 {
		t.Errorf("after second right: expected column index 4 (solved), got %d", activeCI)
	}

	// Move right again — should stay (at rightmost non-empty)
	prevCol := m.col
	m, _ = m.moveRight()
	if m.col != prevCol {
		t.Errorf("expected to stay at col %d, moved to %d", prevCol, m.col)
	}

	// Move left — should go back
	m, _ = m.moveLeft()
	activeCI = m.visibleCols[m.col]
	if activeCI != 1 {
		t.Errorf("after left: expected column index 1, got %d", activeCI)
	}
}

func TestNavigation_UpDown(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(1, "open", "urgent", "First"),
		makeTicket(2, "open", "high", "Second"),
		makeTicket(3, "open", "normal", "Third"),
	}

	m.rebuildColumns(items)

	// Move to open column
	m, _ = m.moveRight()
	if m.row != 0 {
		t.Errorf("expected row 0, got %d", m.row)
	}

	m, _ = m.moveDown()
	if m.row != 1 {
		t.Errorf("expected row 1, got %d", m.row)
	}

	m, _ = m.moveDown()
	if m.row != 2 {
		t.Errorf("expected row 2, got %d", m.row)
	}

	// At bottom — should not move further
	m, _ = m.moveDown()
	if m.row != 2 {
		t.Errorf("expected row 2 (clamped), got %d", m.row)
	}

	m, _ = m.moveUp()
	if m.row != 1 {
		t.Errorf("expected row 1, got %d", m.row)
	}

	// At top — should not go negative
	m, _ = m.moveUp()
	m, _ = m.moveUp()
	if m.row != 0 {
		t.Errorf("expected row 0, got %d", m.row)
	}
}

func TestNavigation_EmptyColumn(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	// Only open and solved have tickets — pending, hold are empty
	items := []types.Ticket{
		makeTicket(1, "open", "normal", "Open"),
		makeTicket(2, "solved", "normal", "Solved"),
	}

	m.rebuildColumns(items)

	// Start on first non-empty visible column
	// Move right from wherever we are — should land on open (index 1)
	m.col = 0
	m.row = 0
	m, _ = m.moveRight()
	activeCI := m.visibleCols[m.col]
	if activeCI != 1 {
		t.Errorf("expected to land on open (1), got %d", activeCI)
	}

	// Move right should skip pending(2), hold(3) and land on solved(4)
	m, _ = m.moveRight()
	activeCI = m.visibleCols[m.col]
	if activeCI != 4 {
		t.Errorf("expected to skip to solved (4), got %d", activeCI)
	}
}

func TestSelectedTicket(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(10, "new", "high", "Ticket 10"),
		makeTicket(20, "open", "normal", "Ticket 20"),
	}

	m.rebuildColumns(items)

	// First selected ticket should be from first column
	ticket := m.selectedTicket()
	if ticket == nil {
		t.Fatal("expected a selected ticket, got nil")
	}
	if ticket.ID != 10 {
		t.Errorf("expected ticket 10, got %d", ticket.ID)
	}

	// Move to open column
	m, _ = m.moveRight()
	ticket = m.selectedTicket()
	if ticket == nil {
		t.Fatal("expected a selected ticket after move, got nil")
	}
	if ticket.ID != 20 {
		t.Errorf("expected ticket 20, got %d", ticket.ID)
	}
}

func TestSelectedTicket_Empty(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	m.rebuildColumns(nil)

	ticket := m.selectedTicket()
	if ticket != nil {
		t.Errorf("expected nil ticket for empty kanban, got %v", ticket)
	}
}

func TestCursorStability(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(1, "new", "high", "New"),
		makeTicket(2, "open", "normal", "Open 1"),
		makeTicket(3, "open", "urgent", "Open 2"),
		makeTicket(4, "pending", "low", "Pending"),
	}

	m.rebuildColumns(items)

	// Move to open column, row 1 (ticket 3)
	m, _ = m.moveRight()
	m, _ = m.moveDown()
	ticket := m.selectedTicket()
	if ticket == nil || ticket.ID != 3 {
		t.Fatalf("expected ticket 3, got %v", ticket)
	}

	// Rebuild with same items + an extra — should preserve selection
	items = append(items, makeTicket(5, "open", "high", "Open 3"))
	m.rebuildColumns(items)

	ticket = m.selectedTicket()
	if ticket == nil {
		t.Fatal("expected ticket to be preserved after rebuild, got nil")
	}
	if ticket.ID != 3 {
		t.Errorf("expected ticket 3 preserved, got %d", ticket.ID)
	}
}

func TestKanbanUpdate_Enter(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	items := []types.Ticket{
		makeTicket(42, "open", "normal", "Test ticket"),
	}
	m.rebuildColumns(items)

	// Move to open column where the ticket is
	m, _ = m.moveRight()

	// Press enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected a command from Enter, got nil")
	}

	// Execute the command to get the message
	result := cmd()
	detail, ok := result.(showDetailMsg)
	if !ok {
		t.Fatalf("expected showDetailMsg, got %T", result)
	}
	if detail.id != 42 {
		t.Errorf("expected detail for ticket 42, got %d", detail.id)
	}
}

func TestScrollFollowsCursor(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 30 // with header(3) + chrome(7) + cardHeight(6): fits (30-3-7)/6 = 3 cards

	// Put 20 tickets in the "open" column
	var items []types.Ticket
	for i := 1; i <= 20; i++ {
		items = append(items, makeTicket(int64(i), "open", "normal", "Ticket"))
	}
	m.rebuildColumns(items)

	// Move to open column
	m, _ = m.moveRight()

	vc := m.visibleCards()
	if vc < 2 {
		t.Fatalf("expected at least 2 visible cards, got %d", vc)
	}

	// Navigate down to row 10
	for i := 0; i < 10; i++ {
		m, _ = m.moveDown()
	}
	if m.row != 10 {
		t.Fatalf("expected row 10, got %d", m.row)
	}

	// Scroll offset must have moved so the selected card is visible
	ci := m.visibleCols[m.col]
	scroll := m.scrolls[ci]
	if m.row < scroll || m.row >= scroll+vc {
		t.Errorf("selected row %d not visible: scroll=%d, visibleCards=%d", m.row, scroll, vc)
	}
}

func TestVisibleColumns_NarrowWidth(t *testing.T) {
	m := newKanbanModel()
	m.height = 40

	items := []types.Ticket{
		makeTicket(1, "new", "high", "New"),
		makeTicket(2, "open", "normal", "Open"),
		makeTicket(3, "pending", "low", "Pending"),
		makeTicket(4, "hold", "normal", "Hold"),
		makeTicket(5, "solved", "normal", "Solved"),
	}

	// Wide: all 5 columns
	m.width = 120
	m.rebuildColumns(items)
	if len(m.visibleCols) != 5 {
		t.Errorf("at width 120: expected 5 visible cols, got %d", len(m.visibleCols))
	}

	// Medium: 3 columns (sliding window)
	m.width = 65
	m.recomputeVisible()
	if len(m.visibleCols) != 3 {
		t.Errorf("at width 65: expected 3 visible cols, got %d", len(m.visibleCols))
	}

	// Narrow: 1 column
	m.width = 45
	m.recomputeVisible()
	if len(m.visibleCols) != 1 {
		t.Errorf("at width 45: expected 1 visible col, got %d", len(m.visibleCols))
	}
}
