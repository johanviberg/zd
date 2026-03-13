package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

type ticketLoadedMsg struct {
	ticket types.Ticket
	users  []types.User
}

type auditsLoadedMsg struct {
	audits []types.Audit
	users  []types.User
}

type goBackMsg struct{}

type detailModel struct {
	tickets          zendesk.TicketService
	ticket           *types.Ticket
	users            map[int64]types.User
	audits           []types.Audit
	timeline         []TimelineNode
	commentsOnly     bool
	viewport         viewport.Model
	loading          bool
	err              error
	spinner          spinner.Model
	width            int
	height           int
	ready            bool
	expectedID       int64
	imageAttachments []imageEntry
	imagePicker      imagePickerModel
}

func newDetailModel(tickets zendesk.TicketService) detailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ac("#1D4ED8", "#93C5FD"))
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

func (m detailModel) loadAudits(id int64) tea.Cmd {
	return func() tea.Msg {
		page, err := m.tickets.ListAudits(context.Background(), id, &types.ListAuditsOptions{
			Include:   "users",
			SortOrder: "asc",
		})
		if err != nil {
			return errMsg{err}
		}
		return auditsLoadedMsg{audits: page.Audits, users: page.Users}
	}
}

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ready {
			m.viewport.SetWidth(msg.Width - 4)
			m.viewport.SetHeight(msg.Height - 6)
			if m.ticket != nil {
				m.viewport.SetContent(m.renderContent())
			}
		}

	case ticketLoadedMsg:
		if m.expectedID != 0 && msg.ticket.ID != m.expectedID {
			return m, nil
		}
		m.loading = false
		m.ticket = &msg.ticket
		m.users = make(map[int64]types.User)
		for _, u := range msg.users {
			m.users[u.ID] = u
		}
		m.viewport = viewport.New(viewport.WithWidth(m.width-4), viewport.WithHeight(m.height-6))
		m.viewport.SetContent(m.renderContent())
		m.ready = true
		return m, nil

	case auditsLoadedMsg:
		m.audits = msg.audits
		if m.users == nil {
			m.users = make(map[int64]types.User)
		}
		for _, u := range msg.users {
			m.users[u.ID] = u
		}
		m.timeline = buildTimeline(m.audits)
		m.buildImageEntries()
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

	case imagePickerCloseMsg:
		m.imagePicker = m.imagePicker.close()

	case tea.KeyPressMsg:
		if m.imagePicker.active {
			var cmd tea.Cmd
			m.imagePicker, cmd = m.imagePicker.Update(msg)
			return m, cmd
		}
		switch {
		case key.Matches(msg, keys.Back):
			return m, func() tea.Msg { return goBackMsg{} }
		case key.Matches(msg, keys.FilterTimeline):
			m.commentsOnly = !m.commentsOnly
			if m.ready {
				m.viewport.SetContent(m.renderContent())
			}
			return m, nil
		case key.Matches(msg, keys.Images):
			if len(m.imageAttachments) > 0 {
				m.imagePicker = m.imagePicker.open(m.imageAttachments)
				return m, nil
			}
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

	if m.imagePicker.active {
		return m.imagePicker.View()
	}

	header := subtitleStyle.Render("← esc") + "   " +
		titleStyle.Render(fmt.Sprintf("Ticket #%d", m.ticket.ID))

	return header + "\n\n" + m.viewport.View()
}

func (m detailModel) ViewPanel() string {
	if m.loading {
		return m.spinner.View() + " Loading ticket..."
	}
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	if !m.ready || m.ticket == nil {
		return subtitleStyle.Render("Select a ticket to view details")
	}

	if m.imagePicker.active {
		return m.imagePicker.View()
	}

	header := titleStyle.Render(fmt.Sprintf("Ticket #%d", m.ticket.ID))
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
		wrappedDesc := renderMarkdown("", t.Description, contentWidth-4)
		descBox := borderStyle.Width(contentWidth).Render(
			headerStyle.Render(" Description") + "\n" +
				wrappedDesc,
		)
		b.WriteString(descBox + "\n\n")
	}

	// Timeline section
	nodes := m.timeline
	label := fmt.Sprintf(" Timeline (%d)", len(nodes))
	if m.commentsOnly {
		nodes = filterCommentNodes(m.timeline)
		label = fmt.Sprintf(" Timeline · comments (%d)", len(nodes))
	}
	if len(nodes) > 0 {
		timelineBox := borderStyle.Width(contentWidth).Render(
			headerStyle.Render(label) + "\n" +
				renderTimeline(nodes, m.users, contentWidth-4),
		)
		b.WriteString(timelineBox)
	}

	return b.String()
}

func (m *detailModel) buildImageEntries() {
	m.imageAttachments = nil
	idx := 0
	for _, audit := range m.audits {
		author := timelineUserName(audit.AuthorID, m.users)
		for _, ev := range audit.Events {
			if ev.Type != "Comment" {
				continue
			}
			for _, a := range ev.Attachments {
				idx++
				if a.IsImage() {
					m.imageAttachments = append(m.imageAttachments, imageEntry{
						index:      idx,
						attachment: a,
						authorName: author,
					})
				}
			}
		}
	}
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
