package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

type ticketLoadedMsg struct {
	ticket types.Ticket
	users  []types.User
}

type commentsLoadedMsg struct {
	comments []types.Comment
	users    []types.User
}

type goBackMsg struct{}

type detailModel struct {
	tickets  zendesk.TicketService
	ticket   *types.Ticket
	users    map[int64]types.User
	comments []types.Comment
	viewport viewport.Model
	loading  bool
	err      error
	spinner  spinner.Model
	width    int
	height   int
	ready    bool
}

func newDetailModel(tickets zendesk.TicketService) detailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#93C5FD"})
	return detailModel{
		tickets: tickets,
		loading: true,
		spinner: s,
	}
}

func (m detailModel) loadTicket(id int64) tea.Cmd {
	return func() tea.Msg {
		result, err := m.tickets.Get(context.Background(), id, &types.GetTicketOptions{
			Include: "users",
		})
		if err != nil {
			return errMsg{err}
		}
		return ticketLoadedMsg{ticket: result.Ticket, users: result.Users}
	}
}

func (m detailModel) loadComments(id int64) tea.Cmd {
	return func() tea.Msg {
		page, err := m.tickets.ListComments(context.Background(), id, &types.ListCommentsOptions{
			Include: "users",
		})
		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments: page.Comments, users: page.Users}
	}
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ready {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 6
			if m.ticket != nil {
				m.viewport.SetContent(m.renderContent())
			}
		}

	case ticketLoadedMsg:
		m.loading = false
		m.ticket = &msg.ticket
		m.users = make(map[int64]types.User)
		for _, u := range msg.users {
			m.users[u.ID] = u
		}
		m.viewport = viewport.New(m.width-4, m.height-6)
		m.viewport.SetContent(m.renderContent())
		m.ready = true
		return m, m.loadComments(msg.ticket.ID)

	case commentsLoadedMsg:
		m.comments = msg.comments
		for _, u := range msg.users {
			m.users[u.ID] = u
		}
		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case errMsg:
		m.loading = false
		m.err = msg.err

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			return m, func() tea.Msg { return goBackMsg{} }
		}
		if m.ready {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m detailModel) View() string {
	if m.loading {
		return m.spinner.View() + " Loading ticket..."
	}
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	if !m.ready || m.ticket == nil {
		return ""
	}

	header := subtitleStyle.Render("← esc") + "   " +
		titleStyle.Render(fmt.Sprintf("Ticket #%d", m.ticket.ID))

	return header + "\n\n" + m.viewport.View()
}

func (m detailModel) renderContent() string {
	if m.ticket == nil {
		return ""
	}
	t := m.ticket
	var b strings.Builder

	// Details section
	contentWidth := m.width - 8
	if contentWidth < 40 {
		contentWidth = 40
	}

	detailBox := borderStyle.Width(contentWidth).Render(
		headerStyle.Render(" Details") + "\n" +
			m.renderField("Subject", t.Subject) +
			m.renderField("Status", styledStatus(t.Status)) +
			m.renderField("Priority", styledPriority(t.Priority)) +
			m.renderField("Requester", m.userName(t.RequesterID)) +
			m.renderField("Assignee", m.userName(t.AssigneeID)) +
			m.renderField("Created", t.CreatedAt.Format("2006-01-02 15:04")) +
			m.renderField("Updated", relativeTime(t.UpdatedAt)) +
			m.renderTags(t.Tags),
	)
	b.WriteString(detailBox + "\n\n")

	// Description section
	if t.Description != "" {
		descBox := borderStyle.Width(contentWidth).Render(
			headerStyle.Render(" Description") + "\n" +
				t.Description,
		)
		b.WriteString(descBox + "\n\n")
	}

	// Comments section
	if len(m.comments) > 0 {
		var commentLines strings.Builder
		commentLines.WriteString(headerStyle.Render(fmt.Sprintf(" Comments (%d)", len(m.comments))) + "\n")

		for i, c := range m.comments {
			author := m.userName(c.AuthorID)
			timeAgo := relativeTime(c.CreatedAt)
			isPublic := c.Public == nil || *c.Public

			authorLine := commentAuthorStyle.Render(author)
			if !isPublic {
				authorLine += " " + internalNoteStyle.Render("(internal)")
			}
			authorLine += " " + commentTimeStyle.Render("· "+timeAgo)

			commentLines.WriteString(authorLine + "\n")
			commentLines.WriteString(c.Body + "\n")
			if i < len(m.comments)-1 {
				commentLines.WriteString("\n")
			}
		}

		commentBox := borderStyle.Width(contentWidth).Render(commentLines.String())
		b.WriteString(commentBox)
	}

	return b.String()
}

func (m detailModel) renderField(label, value string) string {
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value) + "\n"
}

func (m detailModel) renderTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var styled []string
	for _, t := range tags {
		styled = append(styled, tagStyle.Render(t))
	}
	return labelStyle.Render("Tags:") + " " + strings.Join(styled, " ") + "\n"
}

func (m detailModel) userName(id int64) string {
	if id == 0 {
		return dimStyle.Render("unassigned")
	}
	if u, ok := m.users[id]; ok {
		if u.Email != "" {
			return u.Name + " (" + u.Email + ")"
		}
		return u.Name
	}
	return fmt.Sprintf("User #%d", id)
}
