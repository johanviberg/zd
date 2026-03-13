package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

const maxCCs = 48

var (
	arrowUp   = key.NewBinding(key.WithKeys("up"))
	arrowDown = key.NewBinding(key.WithKeys("down"))
)

type ccAutocompleteMsg struct {
	users []types.User
	query string
}
type ccAutocompleteErrMsg struct{ err error }

type ccPickerModel struct {
	users    zendesk.UserService
	input    textinput.Model
	results  []types.User
	selected []types.CollaboratorEntry
	cursor   int
	loading  bool
	active   bool
	width    int
}

func newCCPickerModel(users zendesk.UserService) ccPickerModel {
	ti := textinput.New()
	ti.Placeholder = "Search users or enter email..."
	ti.CharLimit = 120
	return ccPickerModel{
		users: users,
		input: ti,
	}
}

func (m ccPickerModel) activate() (ccPickerModel, tea.Cmd) {
	m.active = true
	m.cursor = 0
	m.results = nil
	m.input.Reset()
	return m, m.input.Focus()
}

func (m ccPickerModel) deactivate() ccPickerModel {
	m.active = false
	m.input.Blur()
	m.results = nil
	m.cursor = 0
	return m
}

func (m ccPickerModel) reset() ccPickerModel {
	m.selected = nil
	m.results = nil
	m.cursor = 0
	m.input.Reset()
	m.active = false
	m.input.Blur()
	return m
}

func (m ccPickerModel) doAutocomplete(query string) tea.Cmd {
	users := m.users
	return func() tea.Msg {
		if users == nil {
			return ccAutocompleteMsg{query: query}
		}
		result, err := users.AutocompleteUsers(context.Background(), query)
		if err != nil {
			return ccAutocompleteErrMsg{err: err}
		}
		return ccAutocompleteMsg{users: result, query: query}
	}
}

func (m ccPickerModel) addEntry(entry types.CollaboratorEntry) ccPickerModel {
	if len(m.selected) >= maxCCs {
		return m
	}
	// Deduplicate
	for _, s := range m.selected {
		if entry.UserID > 0 && s.UserID == entry.UserID {
			return m
		}
		if entry.Email != "" && s.Email == entry.Email {
			return m
		}
	}
	m.selected = append(m.selected, entry)
	return m
}

func (m ccPickerModel) removeLastEntry() ccPickerModel {
	if len(m.selected) > 0 {
		m.selected = m.selected[:len(m.selected)-1]
	}
	return m
}

func looksLikeEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

func (m ccPickerModel) Update(msg tea.Msg) (ccPickerModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case ccAutocompleteMsg:
		// Only apply results if the query still matches current input
		if msg.query == m.input.Value() {
			m.results = msg.users
			m.cursor = 0
			m.loading = false
		}
		return m, nil

	case ccAutocompleteErrMsg:
		m.loading = false
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m = m.deactivate()
			return m, nil

		case key.Matches(msg, arrowUp):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, arrowDown):
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, keys.Enter):
			if len(m.results) > 0 && m.cursor < len(m.results) {
				u := m.results[m.cursor]
				m = m.addEntry(types.CollaboratorEntry{UserID: u.ID, Email: u.Email, Name: u.Name})
				m.input.Reset()
				m.results = nil
				m.cursor = 0
				return m, nil
			}
			// Raw email entry
			val := strings.TrimSpace(m.input.Value())
			if looksLikeEmail(val) {
				m = m.addEntry(types.CollaboratorEntry{Email: val})
				m.input.Reset()
				m.results = nil
				m.cursor = 0
			}
			return m, nil

		case msg.Code == tea.KeyBackspace:
			if m.input.Value() == "" {
				m = m.removeLastEntry()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			// Trigger autocomplete on input change
			val := m.input.Value()
			if len(val) >= 2 {
				m.loading = true
				return m, tea.Batch(cmd, m.doAutocomplete(val))
			}
			m.results = nil
			return m, cmd

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			// Trigger autocomplete on input change
			val := m.input.Value()
			if len(val) >= 2 {
				m.loading = true
				return m, tea.Batch(cmd, m.doAutocomplete(val))
			}
			m.results = nil
			return m, cmd
		}
	}
	return m, nil
}

func (m ccPickerModel) View() string {
	return m.viewFull(true)
}

func (m ccPickerModel) viewFull(isPublic bool) string {
	if !isPublic {
		return ccDisabledStyle.Render("CC: (not available for internal notes)")
	}

	var b strings.Builder
	b.WriteString("CC: ")

	// Render selected chips
	for _, entry := range m.selected {
		label := entry.Email
		if entry.Name != "" {
			label = entry.Name
		}
		b.WriteString(ccChipStyle.Render(label) + " ")
	}

	if !m.active {
		return b.String()
	}

	// Input line
	b.WriteString("\n  > " + m.input.View())

	// Results
	for i, u := range m.results {
		pointer := "    "
		if i == m.cursor {
			pointer = "  > "
		}
		line := fmt.Sprintf("%s (%s)", u.Name, u.Email)
		if i == m.cursor {
			b.WriteString("\n" + pointer + ccResultHighlightStyle.Render(line))
		} else {
			b.WriteString("\n" + pointer + line)
		}
	}

	if len(m.selected) >= maxCCs {
		b.WriteString("\n" + dimStyle.Render("  Maximum CCs reached"))
	}

	return b.String()
}
