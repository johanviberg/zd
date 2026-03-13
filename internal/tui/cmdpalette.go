package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

type cmdPaletteActionMsg struct {
	action string
}

type cmdItem struct {
	name     string
	shortcut string
	category string
	action   string
}

// cmdItems implements fuzzy.Source for fuzzy matching.
type cmdItems []cmdItem

func (c cmdItems) Len() int            { return len(c) }
func (c cmdItems) String(i int) string { return c[i].name + " " + c[i].shortcut + " " + c[i].category }

type cmdPaletteModel struct {
	active   bool
	input    textinput.Model
	all      []cmdItem
	filtered []cmdItem
	matches  fuzzy.Matches
	cursor   int
	width    int
	height   int
}

func newCmdPaletteModel() cmdPaletteModel {
	ti := textinput.New()
	ti.Placeholder = "filter commands..."
	ti.Prompt = "> "
	ti.CharLimit = 64
	return cmdPaletteModel{input: ti}
}

func (m *cmdPaletteModel) open(state viewState, focus panelFocus, showDetail bool, hasMore bool, hasItems bool) tea.Cmd {
	m.active = true
	m.input.Reset()
	m.input.SetValue("")
	m.cursor = 0

	var items []cmdItem

	// Navigation
	if (state == listView || state == splitView) && hasItems {
		items = append(items, cmdItem{"View ticket", "enter", "Navigation", "enter"})
	}
	items = append(items, cmdItem{"Go to ticket", "g", "Navigation", "goto"})
	items = append(items, cmdItem{"Search", "/", "Navigation", "search"})
	if hasItems || state == detailView {
		items = append(items, cmdItem{"Open in browser", "o", "Navigation", "open"})
	}

	// Ticket Actions
	if hasItems || state == detailView {
		items = append(items, cmdItem{"Add comment", "c", "Ticket Actions", "comment"})
		items = append(items, cmdItem{"Change status", "s", "Ticket Actions", "status"})
		items = append(items, cmdItem{"Change priority", "p", "Ticket Actions", "priority"})
	}

	// Display
	if state == listView || state == splitView || state == kanbanView {
		items = append(items, cmdItem{"Toggle Kanban view", "w", "Display", "toggle-kanban"})
	}
	if state == listView || state == splitView {
		items = append(items, cmdItem{"Toggle detail panel", "v", "Display", "toggle-detail"})
		items = append(items, cmdItem{"Toggle chart", "b", "Display", "toggle-chart"})
		items = append(items, cmdItem{"Toggle tags", "t", "Display", "toggle-tags"})
	}
	if state == splitView && showDetail {
		items = append(items, cmdItem{"Toggle focus", "tab", "Display", "toggle-focus"})
	}

	// Data
	items = append(items, cmdItem{"My tickets", "m", "Data", "my-tickets"})
	items = append(items, cmdItem{"Refresh", "R", "Data", "refresh"})
	items = append(items, cmdItem{"Toggle auto-refresh", "r", "Data", "auto-refresh"})
	if hasMore {
		items = append(items, cmdItem{"Load more", "n", "Data", "load-more"})
	}

	// System
	items = append(items, cmdItem{"Quit", "q", "System", "quit"})

	m.all = items
	m.filtered = items
	m.matches = nil

	return m.input.Focus()
}

func (m *cmdPaletteModel) close() {
	m.active = false
	m.input.Blur()
}

func (m cmdPaletteModel) Update(msg tea.Msg) (cmdPaletteModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m.close()
			return m, nil
		case key.Matches(msg, keys.Enter):
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				action := m.filtered[m.cursor].action
				m.close()
				return m, func() tea.Msg { return cmdPaletteActionMsg{action: action} }
			}
			m.close()
			return m, nil
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.refilter()
			return m, cmd
		}
	}
	return m, nil
}

func (m *cmdPaletteModel) refilter() {
	query := m.input.Value()
	if query == "" {
		m.filtered = m.all
		m.matches = nil
		m.cursor = 0
		return
	}
	m.matches = fuzzy.FindFrom(query, cmdItems(m.all))
	m.filtered = make([]cmdItem, len(m.matches))
	for i, match := range m.matches {
		m.filtered[i] = m.all[match.Index]
	}
	// Fallback: if fuzzy found nothing, match against shortcuts exactly
	if len(m.filtered) == 0 {
		for _, item := range m.all {
			if strings.EqualFold(item.shortcut, query) {
				m.filtered = append(m.filtered, item)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// highlightMatches returns text with matched character positions rendered in accentStyle.
func highlightMatches(text string, matchedIndexes []int) string {
	if len(matchedIndexes) == 0 {
		return text
	}
	matched := make(map[int]bool, len(matchedIndexes))
	for _, idx := range matchedIndexes {
		if idx < len(text) {
			matched[idx] = true
		}
	}
	var b strings.Builder
	for i, ch := range text {
		if matched[i] {
			b.WriteString(accentStyle.Render(string(ch)))
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func (m cmdPaletteModel) View() string {
	if !m.active {
		return ""
	}

	// Large modal: ~90% of viewport width, up to 96 cols
	w := m.width * 9 / 10
	if w > 96 {
		w = 96
	}
	if w < 40 {
		w = 40
	}
	innerW := w - 8 // padding (3 each side) + border (1 each side)

	// Title bar: "Commands" left, "esc" right
	titleText := titleStyle.Render("Commands")
	escText := cmdPaletteShortcutStyle.Render("esc")
	titleGap := innerW - lipgloss.Width(titleText) - lipgloss.Width(escText)
	if titleGap < 1 {
		titleGap = 1
	}
	titleBar := titleText + strings.Repeat(" ", titleGap) + escText

	// Input with blank line above
	inputView := "\n" + m.input.View()

	// Command list with category headers and spacing
	maxVisible := m.height*9/10 - 6
	if maxVisible < 10 {
		maxVisible = 10
	}

	var lines []string
	lastCategory := ""
	for i, item := range m.filtered {
		if item.category != lastCategory {
			// Blank line before each category (except first)
			if lastCategory != "" {
				lines = append(lines, "")
			}
			lastCategory = item.category
			lines = append(lines, cmdPaletteCategoryStyle.Render(item.category))
		}

		// Build name with match highlighting when a query is active.
		var nameText string
		if m.matches != nil && i < len(m.matches) {
			var matchedIndexes []int
			for _, idx := range m.matches[i].MatchedIndexes {
				if idx < len(item.name) {
					matchedIndexes = append(matchedIndexes, idx)
				}
			}
			nameText = highlightMatches(item.name, matchedIndexes)
		} else {
			nameText = item.name
		}

		shortcutText := item.shortcut
		// Use plain name width for gap calculation (nameText may contain ANSI codes)
		gap := innerW - lipgloss.Width(item.name) - lipgloss.Width(shortcutText)
		if gap < 1 {
			gap = 1
		}

		if i == m.cursor {
			row := cmdPaletteSelectedStyle.Width(innerW).Render(
				item.name + strings.Repeat(" ", gap) + shortcutText,
			)
			lines = append(lines, row)
		} else {
			row := cmdPaletteItemStyle.Render(nameText) +
				strings.Repeat(" ", gap) +
				cmdPaletteShortcutStyle.Render(shortcutText)
			lines = append(lines, row)
		}
	}

	// Scroll windowing
	visibleLines := lines
	if len(lines) > maxVisible {
		cursorLine := 0
		lineIdx := 0
		lastCat := ""
		for i, item := range m.filtered {
			if item.category != lastCat {
				if lastCat != "" {
					lineIdx++ // blank line
				}
				lastCat = item.category
				lineIdx++ // category header
			}
			if i == m.cursor {
				cursorLine = lineIdx
				break
			}
			lineIdx++
		}

		start := 0
		if cursorLine >= maxVisible {
			start = cursorLine - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(lines) {
			end = len(lines)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
		visibleLines = lines[start:end]
	}

	listContent := strings.Join(visibleLines, "\n")

	content := titleBar + inputView + "\n\n" + listContent

	return borderStyle.Width(w).Padding(1, 3).Render(content)
}
