package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type cmdPaletteModel struct {
	active   bool
	input    textinput.Model
	all      []cmdItem
	filtered []cmdItem
	cursor   int
	width    int
	height   int
}

func newCmdPaletteModel() cmdPaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Prompt = "> "
	ti.CharLimit = 64
	return cmdPaletteModel{input: ti}
}

func (m *cmdPaletteModel) open(state viewState, focus panelFocus, showDetail bool, hasMore bool, hasItems bool) tea.Cmd {
	m.active = true
	m.input.Reset()
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
	if state == listView || state == splitView {
		items = append(items, cmdItem{"Toggle detail panel", "v", "Display", "toggle-detail"})
		items = append(items, cmdItem{"Toggle chart", "b", "Display", "toggle-chart"})
		items = append(items, cmdItem{"Toggle tags", "t", "Display", "toggle-tags"})
	}
	if state == splitView && showDetail {
		items = append(items, cmdItem{"Toggle focus", "tab", "Display", "toggle-focus"})
	}

	// Data
	items = append(items, cmdItem{"Refresh", "R", "Data", "refresh"})
	items = append(items, cmdItem{"Toggle auto-refresh", "r", "Data", "auto-refresh"})
	if hasMore {
		items = append(items, cmdItem{"Load more", "n", "Data", "load-more"})
	}

	// System
	items = append(items, cmdItem{"Quit", "q", "System", "quit"})

	m.all = items
	m.filtered = items

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

	case tea.KeyMsg:
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
	query := strings.ToLower(m.input.Value())
	if query == "" {
		m.filtered = m.all
		m.cursor = 0
		return
	}

	type scored struct {
		item  cmdItem
		score int
	}
	var results []scored
	for _, item := range m.all {
		target := strings.ToLower(item.name + " " + item.category)
		s := fuzzyScore(target, query)
		if s > 0 {
			results = append(results, scored{item, s})
		}
	}

	// Sort by score descending (simple insertion sort for ~15 items)
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	m.filtered = make([]cmdItem, len(results))
	for i, r := range results {
		m.filtered[i] = r.item
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// fuzzyScore returns a positive score if query is a subsequence of target, 0 otherwise.
// Higher scores for consecutive matches and matches at word boundaries.
func fuzzyScore(target, query string) int {
	tRunes := []rune(target)
	qRunes := []rune(query)
	if len(qRunes) == 0 {
		return 1
	}
	if len(qRunes) > len(tRunes) {
		return 0
	}

	score := 0
	qi := 0
	prevMatch := false
	for ti := 0; ti < len(tRunes) && qi < len(qRunes); ti++ {
		if unicode.ToLower(tRunes[ti]) == unicode.ToLower(qRunes[qi]) {
			score++
			// Bonus for consecutive matches
			if prevMatch {
				score += 2
			}
			// Bonus for word boundary match
			if ti == 0 || tRunes[ti-1] == ' ' {
				score += 3
			}
			qi++
			prevMatch = true
		} else {
			prevMatch = false
		}
	}
	if qi < len(qRunes) {
		return 0 // not all query chars matched
	}
	return score
}

func (m cmdPaletteModel) View() string {
	if !m.active {
		return ""
	}

	// Large modal: ~85% of viewport width, up to 84 cols
	w := m.width * 17 / 20
	if w > 84 {
		w = 84
	}
	if w < 40 {
		w = 40
	}
	innerW := w - 6 // padding (3 each side)

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
	maxVisible := m.height*5/6 - 8
	if maxVisible < 6 {
		maxVisible = 6
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

		nameText := item.name
		shortcutText := item.shortcut
		gap := innerW - lipgloss.Width(nameText) - lipgloss.Width(shortcutText)
		if gap < 1 {
			gap = 1
		}

		if i == m.cursor {
			row := cmdPaletteSelectedStyle.Width(innerW).Render(
				nameText + strings.Repeat(" ", gap) + shortcutText,
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
