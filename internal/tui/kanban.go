package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/johanviberg/zd/internal/types"
)

// kanbanStatuses are the 5 statuses shown as columns (excludes "closed").
var kanbanStatuses = statusOrder[:5]

const (
	kanbanCardHeight   = 6 // 3 content lines + 2 border (rounded) + 1 gap
	kanbanHeaderHeight = 3 // header text + count + divider
)

type kanbanModel struct {
	columns     [5][]types.Ticket
	col         int // active column index into visibleCols
	row         int // cursor row within active column
	scrolls     [5]int
	width       int
	height      int
	visibleCols []int // indices into columns array for currently visible columns
	selectedID  int64 // track selected ticket across rebuilds
}

func newKanbanModel() kanbanModel {
	return kanbanModel{
		visibleCols: []int{0, 1, 2, 3, 4},
	}
}

func (m *kanbanModel) rebuildColumns(items []types.Ticket) {
	// Save currently selected ticket ID
	if t := m.selectedTicket(); t != nil {
		m.selectedID = t.ID
	}

	for i := range m.columns {
		m.columns[i] = nil
	}

	statusIdx := map[string]int{
		"new":     0,
		"open":    1,
		"pending": 2,
		"hold":    3,
		"solved":  4,
	}

	for _, t := range items {
		if idx, ok := statusIdx[t.Status]; ok {
			m.columns[idx] = append(m.columns[idx], t)
		}
	}

	m.recomputeVisible()

	// Try to restore selection
	if m.selectedID > 0 {
		for vi, ci := range m.visibleCols {
			for ri, t := range m.columns[ci] {
				if t.ID == m.selectedID {
					m.col = vi
					m.row = ri
					m.clampScroll(ci)
					return
				}
			}
		}
	}

	// Couldn't restore — clamp cursor
	m.clampCursor()
}

func (m *kanbanModel) recomputeVisible() {
	switch {
	case m.width >= 80:
		// Show all 5 columns (>= 100 comfortable, 80-99 skip empty)
		if m.width >= 100 {
			m.visibleCols = []int{0, 1, 2, 3, 4}
		} else {
			// Skip empty columns to reclaim width
			m.visibleCols = m.nonEmptyCols()
			if len(m.visibleCols) == 0 {
				m.visibleCols = []int{0, 1, 2, 3, 4}
			}
		}
	case m.width >= 60:
		// Show 3 columns centered on active column
		m.visibleCols = m.slidingWindow(3)
	case m.width >= 40:
		// Show 1 column — the active one
		activeCol := m.activeColumnIndex()
		m.visibleCols = []int{activeCol}
	default:
		// Too narrow — will trigger fallback to list view in app.go
		m.visibleCols = []int{0, 1, 2, 3, 4}
	}
}

func (m *kanbanModel) nonEmptyCols() []int {
	var cols []int
	for i := 0; i < 5; i++ {
		if len(m.columns[i]) > 0 {
			cols = append(cols, i)
		}
	}
	return cols
}

func (m *kanbanModel) activeColumnIndex() int {
	if m.col >= 0 && m.col < len(m.visibleCols) {
		return m.visibleCols[m.col]
	}
	return 0
}

func (m *kanbanModel) slidingWindow(size int) []int {
	active := m.activeColumnIndex()
	start := active - size/2
	if start < 0 {
		start = 0
	}
	if start+size > 5 {
		start = 5 - size
	}
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > 5 {
		end = 5
	}
	cols := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		cols = append(cols, i)
	}
	return cols
}

func (m *kanbanModel) clampCursor() {
	if len(m.visibleCols) == 0 {
		m.col = 0
		m.row = 0
		return
	}
	if m.col >= len(m.visibleCols) {
		m.col = len(m.visibleCols) - 1
	}
	if m.col < 0 {
		m.col = 0
	}
	ci := m.visibleCols[m.col]
	if m.row >= len(m.columns[ci]) {
		m.row = len(m.columns[ci]) - 1
	}
	if m.row < 0 {
		m.row = 0
	}
	m.clampScroll(ci)
}

func (m *kanbanModel) clampScroll(ci int) {
	vc := m.visibleCards()
	if vc <= 0 {
		m.scrolls[ci] = 0
		return
	}
	if m.row < m.scrolls[ci] {
		m.scrolls[ci] = m.row
	}
	if m.row >= m.scrolls[ci]+vc {
		m.scrolls[ci] = m.row - vc + 1
	}
	maxScroll := len(m.columns[ci]) - vc
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrolls[ci] > maxScroll {
		m.scrolls[ci] = maxScroll
	}
}

func (m *kanbanModel) visibleCards() int {
	vc := (m.height - kanbanHeaderHeight - 7) / kanbanCardHeight
	if vc < 1 {
		vc = 1
	}
	return vc
}

func (m *kanbanModel) selectedTicket() *types.Ticket {
	if len(m.visibleCols) == 0 || m.col < 0 || m.col >= len(m.visibleCols) {
		return nil
	}
	ci := m.visibleCols[m.col]
	if m.row < 0 || m.row >= len(m.columns[ci]) {
		return nil
	}
	return &m.columns[ci][m.row]
}

func (m kanbanModel) Update(msg tea.Msg) (kanbanModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Left):
			return m.moveLeft()
		case key.Matches(msg, keys.Right):
			return m.moveRight()
		case key.Matches(msg, keys.Up):
			return m.moveUp()
		case key.Matches(msg, keys.Down):
			return m.moveDown()
		case key.Matches(msg, keys.Enter):
			if t := m.selectedTicket(); t != nil {
				id := t.ID
				return m, func() tea.Msg { return showDetailMsg{id: id} }
			}
		}
	}
	return m, nil
}

func (m kanbanModel) moveLeft() (kanbanModel, tea.Cmd) {
	if len(m.visibleCols) <= 1 {
		return m, nil
	}
	// Move left, skipping empty columns
	for newCol := m.col - 1; newCol >= 0; newCol-- {
		ci := m.visibleCols[newCol]
		if len(m.columns[ci]) > 0 {
			m.col = newCol
			if m.row >= len(m.columns[ci]) {
				m.row = len(m.columns[ci]) - 1
			}
			m.clampScroll(ci)
			return m, m.emitCursorChanged()
		}
	}
	return m, nil
}

func (m kanbanModel) moveRight() (kanbanModel, tea.Cmd) {
	if len(m.visibleCols) <= 1 {
		return m, nil
	}
	for newCol := m.col + 1; newCol < len(m.visibleCols); newCol++ {
		ci := m.visibleCols[newCol]
		if len(m.columns[ci]) > 0 {
			m.col = newCol
			if m.row >= len(m.columns[ci]) {
				m.row = len(m.columns[ci]) - 1
			}
			m.clampScroll(ci)
			return m, m.emitCursorChanged()
		}
	}
	return m, nil
}

func (m kanbanModel) moveUp() (kanbanModel, tea.Cmd) {
	if m.row > 0 {
		m.row--
		ci := m.visibleCols[m.col]
		m.clampScroll(ci)
		return m, m.emitCursorChanged()
	}
	return m, nil
}

func (m kanbanModel) moveDown() (kanbanModel, tea.Cmd) {
	if len(m.visibleCols) == 0 || m.col >= len(m.visibleCols) {
		return m, nil
	}
	ci := m.visibleCols[m.col]
	if m.row < len(m.columns[ci])-1 {
		m.row++
		m.clampScroll(ci)
		return m, m.emitCursorChanged()
	}
	return m, nil
}

func (m kanbanModel) emitCursorChanged() tea.Cmd {
	if t := m.selectedTicket(); t != nil {
		id := t.ID
		return func() tea.Msg { return cursorChangedMsg{id: id} }
	}
	return nil
}

func (m kanbanModel) View() string {
	if len(m.visibleCols) == 0 {
		return subtitleStyle.Render("No columns to display.")
	}

	numCols := len(m.visibleCols)
	// Available width minus inter-column gaps
	availW := m.width - 4 // padding from app.go
	if availW < 10 {
		availW = 10
	}
	gaps := numCols - 1
	colWidth := (availW - gaps) / numCols
	if colWidth < 14 {
		colWidth = 14
	}

	vc := m.visibleCards()
	var colViews []string

	for vi, ci := range m.visibleCols {
		col := m.renderColumn(ci, vi == m.col, colWidth, vc)
		colViews = append(colViews, col)
	}

	// Join with single-space gaps
	return lipgloss.JoinHorizontal(lipgloss.Top, intersperse(colViews, " ")...)
}

// intersperse inserts sep between each element.
func intersperse(items []string, sep string) []string {
	if len(items) <= 1 {
		return items
	}
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			result = append(result, sep)
		}
		result = append(result, item)
	}
	return result
}

func (m kanbanModel) renderColumn(ci int, isActive bool, colWidth int, visCards int) string {
	status := kanbanStatuses[ci]
	count := len(m.columns[ci])

	// Column header
	icon := statusIcons[status]
	color := statusColors[status]
	headerText := lipgloss.NewStyle().Foreground(color).Render(icon+" "+status) +
		" " + dimStyle.Render(fmt.Sprintf("(%d)", count))
	header := kanbanColumnHeaderStyle.Width(colWidth).Render(headerText)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")

	if count == 0 {
		empty := dimStyle.Render("  (empty)")
		b.WriteString(lipgloss.NewStyle().Width(colWidth).Height(visCards * kanbanCardHeight).Render(empty))
		return b.String()
	}

	// Scroll indicators
	scroll := m.scrolls[ci]
	if scroll > 0 {
		b.WriteString(dimStyle.Render("  ▲ more"))
		b.WriteString("\n")
	}

	end := scroll + visCards
	if end > count {
		end = count
	}

	for i := scroll; i < end; i++ {
		t := m.columns[ci][i]
		selected := isActive && i == m.row
		card := m.renderCard(t, selected, colWidth, status)
		b.WriteString(card)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	if end < count {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  ▼ more"))
	}

	return b.String()
}

func (m kanbanModel) renderCard(t types.Ticket, selected bool, colWidth int, status string) string {
	innerW := colWidth - 4 // card border (2) is outside lipgloss Width; padding (2) is inside
	if innerW < 6 {
		innerW = 6
	}

	// Line 1: ticket ID
	idStr := fmt.Sprintf("#%d", t.ID)

	// Line 2: subject (truncated)
	subject := strings.ReplaceAll(strings.ReplaceAll(t.Subject, "\n", " "), "\r", "")
	runes := []rune(subject)
	if len(runes) > innerW {
		subject = string(runes[:innerW-1]) + "…"
	}

	// Line 3: priority + age
	prioIcon := priorityIcons[t.Priority]
	if prioIcon == "" {
		prioIcon = "-"
	}
	ago := relativeTime(t.UpdatedAt)
	line3 := prioIcon + " " + t.Priority
	if remaining := innerW - len([]rune(line3)) - len([]rune(ago)) - 1; remaining > 0 {
		line3 += strings.Repeat(" ", remaining) + ago
	} else {
		line3 += " " + ago
	}
	// Truncate line3 if needed
	if l3runes := []rune(line3); len(l3runes) > innerW {
		line3 = string(l3runes[:innerW])
	}

	content := idStr + "\n" + subject + "\n" + line3

	// lipgloss v1 Width excludes borders; subtract 2 for left+right border
	// so the rendered card width matches colWidth exactly.
	cardW := colWidth - 2
	if selected {
		color, ok := statusColors[status]
		if !ok {
			color = statusColors["open"]
		}
		return kanbanCardSelectedStyle.
			BorderForeground(color).
			Width(cardW).
			Render(content)
	}
	return kanbanCardStyle.Width(cardW).Render(content)
}
