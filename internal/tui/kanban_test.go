package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	assert.Len(t, m.columns[0], 1, "expected 1 new ticket")
	assert.Len(t, m.columns[1], 2, "expected 2 open tickets")
	assert.Len(t, m.columns[2], 1, "expected 1 pending ticket")
	assert.Len(t, m.columns[3], 0, "expected 0 hold tickets")
	assert.Len(t, m.columns[4], 1, "expected 1 solved ticket")
}

func TestRebuildColumns_Empty(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	m.rebuildColumns(nil)

	for i := 0; i < 5; i++ {
		assert.Len(t, m.columns[i], 0, "column %d: expected 0 tickets", i)
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
	assert.Equal(t, 1, total, "expected 1 total ticket (only open)")
	assert.Len(t, m.columns[1], 1, "expected 1 open ticket")
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
	assert.Equal(t, 0, m.col, "expected initial col 0")

	// Move right — should skip empty columns
	m, _ = m.moveRight()
	activeCI := m.visibleCols[m.col]
	assert.Equal(t, 1, activeCI, "after right: expected column index 1 (open)")

	// Move right again — should skip pending (empty) and hold (empty)
	m, _ = m.moveRight()
	activeCI = m.visibleCols[m.col]
	assert.Equal(t, 4, activeCI, "after second right: expected column index 4 (solved)")

	// Move right again — should stay (at rightmost non-empty)
	prevCol := m.col
	m, _ = m.moveRight()
	assert.Equal(t, prevCol, m.col, "expected to stay at col %d", prevCol)

	// Move left — should go back
	m, _ = m.moveLeft()
	activeCI = m.visibleCols[m.col]
	assert.Equal(t, 1, activeCI, "after left: expected column index 1")
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
	assert.Equal(t, 0, m.row, "expected row 0")

	m, _ = m.moveDown()
	assert.Equal(t, 1, m.row, "expected row 1")

	m, _ = m.moveDown()
	assert.Equal(t, 2, m.row, "expected row 2")

	// At bottom — should not move further
	m, _ = m.moveDown()
	assert.Equal(t, 2, m.row, "expected row 2 (clamped)")

	m, _ = m.moveUp()
	assert.Equal(t, 1, m.row, "expected row 1")

	// At top — should not go negative
	m, _ = m.moveUp()
	m, _ = m.moveUp()
	assert.Equal(t, 0, m.row, "expected row 0")
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
	assert.Equal(t, 1, activeCI, "expected to land on open (1)")

	// Move right should skip pending(2), hold(3) and land on solved(4)
	m, _ = m.moveRight()
	activeCI = m.visibleCols[m.col]
	assert.Equal(t, 4, activeCI, "expected to skip to solved (4)")
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
	require.NotNil(t, ticket, "expected a selected ticket, got nil")
	assert.Equal(t, int64(10), ticket.ID, "expected ticket 10")

	// Move to open column
	m, _ = m.moveRight()
	ticket = m.selectedTicket()
	require.NotNil(t, ticket, "expected a selected ticket after move, got nil")
	assert.Equal(t, int64(20), ticket.ID, "expected ticket 20")
}

func TestSelectedTicket_Empty(t *testing.T) {
	m := newKanbanModel()
	m.width = 120
	m.height = 40

	m.rebuildColumns(nil)

	ticket := m.selectedTicket()
	assert.Nil(t, ticket, "expected nil ticket for empty kanban")
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
	require.NotNil(t, ticket, "expected ticket 3")
	require.Equal(t, int64(3), ticket.ID, "expected ticket 3, got %v", ticket)

	// Rebuild with same items + an extra — should preserve selection
	items = append(items, makeTicket(5, "open", "high", "Open 3"))
	m.rebuildColumns(items)

	ticket = m.selectedTicket()
	require.NotNil(t, ticket, "expected ticket to be preserved after rebuild, got nil")
	assert.Equal(t, int64(3), ticket.ID, "expected ticket 3 preserved")
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
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	m, cmd := m.Update(msg)
	require.NotNil(t, cmd, "expected a command from Enter, got nil")

	// Execute the command to get the message
	result := cmd()
	detail, ok := result.(showDetailMsg)
	require.True(t, ok, "expected showDetailMsg, got %T", result)
	assert.Equal(t, int64(42), detail.id, "expected detail for ticket 42")
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
	require.GreaterOrEqual(t, vc, 2, "expected at least 2 visible cards")

	// Navigate down to row 10
	for i := 0; i < 10; i++ {
		m, _ = m.moveDown()
	}
	require.Equal(t, 10, m.row, "expected row 10")

	// Scroll offset must have moved so the selected card is visible
	ci := m.visibleCols[m.col]
	scroll := m.scrolls[ci]
	assert.True(t, m.row >= scroll && m.row < scroll+vc,
		"selected row %d not visible: scroll=%d, visibleCards=%d", m.row, scroll, vc)
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
	assert.Len(t, m.visibleCols, 5, "at width 120: expected 5 visible cols")

	// Medium: 3 columns (sliding window)
	m.width = 65
	m.recomputeVisible()
	assert.Len(t, m.visibleCols, 3, "at width 65: expected 3 visible cols")

	// Narrow: 1 column
	m.width = 45
	m.recomputeVisible()
	assert.Len(t, m.visibleCols, 1, "at width 45: expected 1 visible col")
}
