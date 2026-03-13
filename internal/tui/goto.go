package tui

import (
	"strconv"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type gotoDoneMsg struct {
	id int64
}

type gotoCancelMsg struct{}

type gotoModel struct {
	input  textinput.Model
	active bool
	width  int
}

func newGotoModel() gotoModel {
	ti := textinput.New()
	ti.Placeholder = "Ticket number"
	ti.Prompt = "#> "
	ti.CharLimit = 20
	return gotoModel{input: ti}
}

func (m gotoModel) open() (gotoModel, tea.Cmd) {
	m.active = true
	m.input.Reset()
	return m, m.input.Focus()
}

func (m gotoModel) close() gotoModel {
	m.active = false
	m.input.Blur()
	return m
}

func (m gotoModel) Update(msg tea.Msg) (gotoModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.SetWidth(msg.Width - 10)

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m = m.close()
			return m, func() tea.Msg { return gotoCancelMsg{} }
		case key.Matches(msg, keys.Enter):
			val := m.input.Value()
			m = m.close()
			id, err := strconv.ParseInt(val, 10, 64)
			if err != nil || id <= 0 {
				return m, func() tea.Msg { return gotoCancelMsg{} }
			}
			return m, func() tea.Msg { return gotoDoneMsg{id: id} }
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m gotoModel) View() string {
	if !m.active {
		return ""
	}

	width := m.width - 8
	if width < 30 {
		width = 30
	}

	content := headerStyle.Render(" Go to Ticket") + "\n" + m.input.View()
	return borderStyle.Width(width).Render(content)
}
