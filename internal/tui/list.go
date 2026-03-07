package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

type ticketsLoadedMsg struct {
	page *types.TicketPage
}

type searchResultsMsg struct {
	page *types.SearchPage
}

type showDetailMsg struct {
	id int64
}

type listModel struct {
	tickets     zendesk.TicketService
	search      zendesk.SearchService
	items       []types.Ticket
	users       map[int64]types.User
	cursor      int
	width       int
	height      int
	loading     bool
	err         error
	spinner     spinner.Model
	hasMore     bool
	afterCursor string
	totalCount  int
	searchQuery string
	searching   bool
}

func newListModel(tickets zendesk.TicketService, search zendesk.SearchService) listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#93C5FD"})
	return listModel{
		tickets: tickets,
		search:  search,
		loading: true,
		spinner: s,
	}
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadTickets())
}

func (m listModel) loadTickets() tea.Cmd {
	return func() tea.Msg {
		opts := &types.ListTicketsOptions{
			Limit:     50,
			Sort:      "updated_at",
			SortOrder: "desc",
			Include:   "users",
		}
		page, err := m.tickets.List(context.Background(), opts)
		if err != nil {
			return errMsg{err}
		}
		return ticketsLoadedMsg{page}
	}
}

func (m listModel) doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		opts := &types.SearchOptions{
			Limit:     50,
			SortBy:    "updated_at",
			SortOrder: "desc",
			Include:   "users",
		}
		page, err := m.search.Search(context.Background(), query, opts)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{page}
	}
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ticketsLoadedMsg:
		m.loading = false
		m.items = msg.page.Tickets
		m.hasMore = msg.page.Meta.HasMore
		m.afterCursor = msg.page.Meta.AfterCursor
		m.totalCount = len(msg.page.Tickets)
		if msg.page.Count > 0 {
			m.totalCount = msg.page.Count
		}
		m.users = make(map[int64]types.User)
		for _, u := range msg.page.Users {
			m.users[u.ID] = u
		}
		m.cursor = 0

	case searchResultsMsg:
		m.loading = false
		m.searching = false
		m.items = make([]types.Ticket, len(msg.page.Results))
		for i, r := range msg.page.Results {
			m.items[i] = r.Ticket
		}
		m.totalCount = msg.page.Count
		m.hasMore = msg.page.Meta.HasMore
		m.users = make(map[int64]types.User)
		for _, u := range msg.page.Users {
			m.users[u.ID] = u
		}
		m.cursor = 0

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
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if len(m.items) > 0 && m.cursor < len(m.items) {
				return m, func() tea.Msg {
					return showDetailMsg{id: m.items[m.cursor].ID}
				}
			}
		}
	}
	return m, nil
}

func (m listModel) View() string {
	if m.loading {
		return m.spinner.View() + " Loading tickets..."
	}
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}
	if len(m.items) == 0 {
		return subtitleStyle.Render("No tickets found.")
	}

	var b strings.Builder

	// Header
	countLabel := fmt.Sprintf("%d tickets", m.totalCount)
	if m.searchQuery != "" {
		countLabel = fmt.Sprintf("%d results for %q", m.totalCount, m.searchQuery)
	}
	header := titleStyle.Render("Tickets") + "  " + subtitleStyle.Render(countLabel)
	b.WriteString(header + "\n\n")

	// Calculate visible rows
	visibleRows := m.height - 6 // header + help bar
	if visibleRows < 1 {
		visibleRows = 10
	}

	// Scrolling window
	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := start; i < end; i++ {
		t := m.items[i]
		line := m.renderTicketRow(t, i == m.cursor)
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m listModel) renderTicketRow(t types.Ticket, selected bool) string {
	pointer := "  "
	if selected {
		pointer = "▸ "
	}

	id := fmt.Sprintf("#%d", t.ID)
	status := styledStatus(t.Status)
	priority := styledPriority(t.Priority)
	subject := t.Subject
	ago := relativeTime(t.UpdatedAt)

	// Truncate subject to fit
	maxSubject := m.width - 55
	if maxSubject < 20 {
		maxSubject = 20
	}
	runes := []rune(subject)
	if len(runes) > maxSubject {
		subject = string(runes[:maxSubject-1]) + "…"
	}

	idCol := lipgloss.NewStyle().Width(7).Render(id)
	statusCol := lipgloss.NewStyle().Width(12).Render(status)
	prioCol := lipgloss.NewStyle().Width(11).Render(priority)
	subjectCol := lipgloss.NewStyle().Width(maxSubject).Render(subject)
	agoCol := dimStyle.Render(ago)

	row := pointer + idCol + " " + statusCol + " " + prioCol + " " + subjectCol + "  " + agoCol

	if selected {
		return selectedStyle.Render(row)
	}
	return row
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	d := time.Since(t)
	if d < 0 {
		return "just now"
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 0 {
			months = 1
		}
		return fmt.Sprintf("%dmo ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		return fmt.Sprintf("%dy ago", years)
	}
}
