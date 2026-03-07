package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type searchDoneMsg struct {
	query string
}

type searchCancelMsg struct{}

type searchModel struct {
	input  textinput.Model
	active bool
	width  int
}

func newSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = "status:open priority:high tags:billing"
	ti.Prompt = "> "
	ti.CharLimit = 256
	return searchModel{input: ti}
}

func (m searchModel) open() (searchModel, tea.Cmd) {
	m.active = true
	m.input.Reset()
	return m, m.input.Focus()
}

func (m searchModel) close() searchModel {
	m.active = false
	m.input.Blur()
	return m
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.Width = msg.Width - 10

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m = m.close()
			return m, func() tea.Msg { return searchCancelMsg{} }
		case key.Matches(msg, keys.Enter):
			query := m.input.Value()
			m = m.close()
			if query != "" {
				return m, func() tea.Msg { return searchDoneMsg{query: query} }
			}
			return m, func() tea.Msg { return searchCancelMsg{} }
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m searchModel) View() string {
	if !m.active {
		return ""
	}

	width := m.width - 8
	if width < 30 {
		width = 30
	}

	content := headerStyle.Render(" Search") + "\n" + m.input.View()
	return borderStyle.Width(width).Render(content)
}
