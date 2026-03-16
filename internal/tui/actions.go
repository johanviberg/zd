package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

type ticketUpdatedMsg struct {
	ticket *types.Ticket
}

type actionErrMsg struct{ err error }

type actionMode int

const (
	actionNone actionMode = iota
	actionComment
	actionStatus
	actionPriority
)

var validStatuses = []string{"new", "open", "pending", "hold", "solved"}
var validPriorities = []string{"urgent", "high", "normal", "low"}

type actionsModel struct {
	tickets    zendesk.TicketService
	ticketID   int64
	mode       actionMode
	textarea   textarea.Model
	isPublic   bool
	statusIdx  int
	prioIdx    int
	submitting bool
	err        error
	spinner    spinner.Model
	width      int
	height     int
	current    string // current status or priority
	ccPicker   ccPickerModel
	ccFocused  bool
}

func newActionsModel(tickets zendesk.TicketService, users zendesk.UserService) actionsModel {
	ta := textarea.New()
	ta.Placeholder = "Type your comment..."
	ta.ShowLineNumbers = false
	ta.SetHeight(6)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ac("#1D4ED8", "#93C5FD"))

	return actionsModel{
		tickets:  tickets,
		textarea: ta,
		isPublic: true,
		spinner:  s,
		ccPicker: newCCPickerModel(users),
	}
}

func (m actionsModel) openComment(ticketID int64) (actionsModel, tea.Cmd) {
	m.ticketID = ticketID
	m.mode = actionComment
	m.isPublic = true
	m.err = nil
	m.ccFocused = false
	m.ccPicker = m.ccPicker.reset()
	m.textarea.Reset()
	return m, m.textarea.Focus()
}

func (m actionsModel) openStatus(ticketID int64, currentStatus string) actionsModel {
	m.ticketID = ticketID
	m.mode = actionStatus
	m.current = currentStatus
	m.err = nil
	m.statusIdx = 0
	for i, s := range validStatuses {
		if s == currentStatus {
			m.statusIdx = i
			break
		}
	}
	return m
}

func (m actionsModel) openPriority(ticketID int64, currentPriority string) actionsModel {
	m.ticketID = ticketID
	m.mode = actionPriority
	m.current = currentPriority
	m.err = nil
	m.prioIdx = 0
	for i, p := range validPriorities {
		if p == currentPriority {
			m.prioIdx = i
			break
		}
	}
	return m
}

func (m actionsModel) close() actionsModel {
	m.mode = actionNone
	m.textarea.Blur()
	return m
}

func (m actionsModel) submitComment() tea.Cmd {
	body := m.textarea.Value()
	isPublic := m.isPublic
	ticketID := m.ticketID
	tickets := m.tickets
	collaborators := append([]types.CollaboratorEntry(nil), m.ccPicker.selected...)
	return func() tea.Msg {
		pub := isPublic
		req := &types.UpdateTicketRequest{
			Comment: &types.Comment{
				Body:   body,
				Public: &pub,
			},
		}
		if pub && len(collaborators) > 0 {
			req.AdditionalCollaborators = collaborators
		}
		ticket, err := tickets.Update(context.Background(), ticketID, req)
		if err != nil {
			return actionErrMsg{err}
		}
		return ticketUpdatedMsg{ticket: ticket}
	}
}

func (m actionsModel) submitStatus() tea.Cmd {
	status := validStatuses[m.statusIdx]
	ticketID := m.ticketID
	tickets := m.tickets
	return func() tea.Msg {
		ticket, err := tickets.Update(context.Background(), ticketID, &types.UpdateTicketRequest{
			Status: status,
		})
		if err != nil {
			return actionErrMsg{err}
		}
		return ticketUpdatedMsg{ticket: ticket}
	}
}

func (m actionsModel) submitPriority() tea.Cmd {
	priority := validPriorities[m.prioIdx]
	ticketID := m.ticketID
	tickets := m.tickets
	return func() tea.Msg {
		ticket, err := tickets.Update(context.Background(), ticketID, &types.UpdateTicketRequest{
			Priority: priority,
		})
		if err != nil {
			return actionErrMsg{err}
		}
		return ticketUpdatedMsg{ticket: ticket}
	}
}

func (m actionsModel) Update(msg tea.Msg) (actionsModel, tea.Cmd) {
	if m.mode == actionNone {
		return m, nil
	}

	switch msg := msg.(type) {
	case ccAutocompleteMsg, ccAutocompleteErrMsg:
		if m.mode == actionComment && m.ccFocused {
			var cmd tea.Cmd
			m.ccPicker, cmd = m.ccPicker.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		if m.submitting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case ticketUpdatedMsg:
		m.submitting = false
		m = m.close()
		return m, nil

	case actionErrMsg:
		m.submitting = false
		m.err = msg.err
		return m, nil

	case tea.KeyPressMsg:
		if m.submitting {
			return m, nil
		}

		switch m.mode {
		case actionComment:
			// Route to CC picker when focused
			if m.ccFocused {
				switch {
				case key.Matches(msg, keys.AddCC):
					m.ccPicker = m.ccPicker.deactivate()
					m.ccFocused = false
					return m, m.textarea.Focus()
				default:
					var cmd tea.Cmd
					m.ccPicker, cmd = m.ccPicker.Update(msg)
					// Check if picker deactivated itself (esc)
					if !m.ccPicker.active {
						m.ccFocused = false
						return m, tea.Batch(cmd, m.textarea.Focus())
					}
					return m, cmd
				}
			}

			switch {
			case key.Matches(msg, keys.Back):
				m = m.close()
				return m, nil
			case key.Matches(msg, keys.Submit):
				if m.textarea.Value() != "" {
					m.submitting = true
					return m, tea.Batch(m.spinner.Tick, m.submitComment())
				}
			case key.Matches(msg, keys.Tab):
				m.isPublic = !m.isPublic
				if !m.isPublic {
					m.ccPicker = m.ccPicker.deactivate()
					m.ccFocused = false
					m.ccPicker.selected = nil
				}
				return m, nil
			case key.Matches(msg, keys.AddCC):
				if m.isPublic {
					m.ccFocused = true
					m.textarea.Blur()
					var cmd tea.Cmd
					m.ccPicker, cmd = m.ccPicker.activate()
					return m, cmd
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}

		case actionStatus:
			switch {
			case key.Matches(msg, keys.Back):
				m = m.close()
				return m, nil
			case key.Matches(msg, keys.Up):
				if m.statusIdx > 0 {
					m.statusIdx--
				}
			case key.Matches(msg, keys.Down):
				if m.statusIdx < len(validStatuses)-1 {
					m.statusIdx++
				}
			case key.Matches(msg, keys.Enter):
				m.submitting = true
				return m, tea.Batch(m.spinner.Tick, m.submitStatus())
			}

		case actionPriority:
			switch {
			case key.Matches(msg, keys.Back):
				m = m.close()
				return m, nil
			case key.Matches(msg, keys.Up):
				if m.prioIdx > 0 {
					m.prioIdx--
				}
			case key.Matches(msg, keys.Down):
				if m.prioIdx < len(validPriorities)-1 {
					m.prioIdx++
				}
			case key.Matches(msg, keys.Enter):
				m.submitting = true
				return m, tea.Batch(m.spinner.Tick, m.submitPriority())
			}
		}
	}
	return m, nil
}

func (m actionsModel) View() string {
	if m.mode == actionNone {
		return ""
	}

	switch m.mode {
	case actionComment:
		return m.viewComment()
	case actionStatus:
		return m.viewPicker("Change Status", validStatuses, m.statusIdx)
	case actionPriority:
		return m.viewPicker("Change Priority", validPriorities, m.prioIdx)
	}
	return ""
}

func (m actionsModel) viewComment() string {
	title := titleStyle.Render("Add Comment")

	publicToggle := "[ ] Public reply   [x] Internal note"
	if m.isPublic {
		publicToggle = "[x] Public reply   [ ] Internal note"
	}

	var statusLine string
	if m.submitting {
		statusLine = m.spinner.View() + " Submitting..."
	} else if m.err != nil {
		statusLine = errorStyle.Render("Error: " + m.err.Error())
	}

	help := dimStyle.Render("ctrl+s submit   esc cancel   tab toggle public/internal   ctrl+a add CC")

	width := m.width - 8
	if width < 40 {
		width = 40
	}
	m.textarea.SetWidth(width)
	m.ccPicker.width = width

	ccLine := m.ccPicker.viewFull(m.isPublic)

	content := title + "\n\n" +
		m.textarea.View() + "\n\n" +
		publicToggle + "\n" +
		ccLine + "\n\n" +
		help
	if statusLine != "" {
		content += "\n" + statusLine
	}

	return borderStyle.Width(width + 4).Render(content)
}

func (m actionsModel) viewPicker(title string, options []string, selected int) string {
	var b fmt.Stringer = &pickerBuilder{title: title, options: options, selected: selected, current: m.current}

	var statusLine string
	if m.submitting {
		statusLine = "\n" + m.spinner.View() + " Updating..."
	} else if m.err != nil {
		statusLine = "\n" + errorStyle.Render("Error: "+m.err.Error())
	}

	help := dimStyle.Render("↑↓ select   enter confirm   esc cancel")

	return borderStyle.Padding(1, 2).Render(b.String() + "\n\n" + help + statusLine)
}

type pickerBuilder struct {
	title    string
	options  []string
	selected int
	current  string
}

func (p *pickerBuilder) String() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(p.title) + "\n\n")
	for i, opt := range p.options {
		pointer := "  "
		if i == p.selected {
			pointer = "> "
		}
		label := opt
		if opt == p.current {
			label += " (current)"
		}
		if i == p.selected {
			b.WriteString(selectedStyle.Render(pointer+label) + "\n")
		} else {
			b.WriteString(pointer + label + "\n")
		}
	}
	return b.String()
}
