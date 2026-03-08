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

	"github.com/johanviberg/zd/internal/nlq"
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

type countdownTickMsg struct{}

type cursorChangedMsg struct {
	id int64
}

type refreshLoadedMsg struct {
	page *types.TicketPage
}

type moreTicketsLoadedMsg struct {
	page *types.TicketPage
}

type moreSearchResultsMsg struct {
	page *types.SearchPage
}

type listModel struct {
	tickets             zendesk.TicketService
	search              zendesk.SearchService
	items               []types.Ticket
	users               map[int64]types.User
	cursor              int
	width               int
	height              int
	loading             bool
	err                 error
	spinner             spinner.Model
	hasMore             bool
	afterCursor         string
	totalCount          int
	searchQuery         string
	searching           bool
	autoRefresh         bool
	refreshCountdown    int
	knownTicketIDs      map[int64]bool
	newTicketIDs        map[int64]bool
	loadingMore         bool
	translatedQuery     string
	showChart           bool
	lastRefreshNewCount int
}

func newListModel(tickets zendesk.TicketService, search zendesk.SearchService) listModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#93C5FD"})
	return listModel{
		tickets:   tickets,
		search:    search,
		loading:   true,
		spinner:   s,
		showChart: true,
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

func (m *listModel) doSearch(query string) tea.Cmd {
	translated := nlq.Translate(query)
	m.translatedQuery = translated
	return func() tea.Msg {
		opts := &types.SearchOptions{
			Limit:   50,
			Export:  true,
			Include: "users",
		}
		page, err := m.search.Search(context.Background(), translated, opts)
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{page}
	}
}

const refreshIntervalSeconds = 300 // 5 minutes

func scheduleCountdownTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return countdownTickMsg{}
	})
}

func (m listModel) loadTicketsForRefresh() tea.Cmd {
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
		return refreshLoadedMsg{page}
	}
}

func (m listModel) loadMoreTickets() tea.Cmd {
	return func() tea.Msg {
		opts := &types.ListTicketsOptions{
			Limit:     50,
			Sort:      "updated_at",
			SortOrder: "desc",
			Include:   "users",
			Cursor:    m.afterCursor,
		}
		page, err := m.tickets.List(context.Background(), opts)
		if err != nil {
			return errMsg{err}
		}
		return moreTicketsLoadedMsg{page}
	}
}

func (m listModel) loadMoreSearch() tea.Cmd {
	return func() tea.Msg {
		opts := &types.SearchOptions{
			Limit:   50,
			Export:  true,
			Include: "users",
			Cursor:  m.afterCursor,
		}
		page, err := m.search.Search(context.Background(), m.translatedQuery, opts)
		if err != nil {
			return errMsg{err}
		}
		return moreSearchResultsMsg{page}
	}
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ticketsLoadedMsg:
		m.loading = false
		m.loadingMore = false
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
		m.knownTicketIDs = make(map[int64]bool)
		for _, t := range msg.page.Tickets {
			m.knownTicketIDs[t.ID] = true
		}
		m.newTicketIDs = make(map[int64]bool)
		m.cursor = 0

	case searchResultsMsg:
		m.loading = false
		m.loadingMore = false
		m.searching = false
		m.items = make([]types.Ticket, len(msg.page.Results))
		for i, r := range msg.page.Results {
			m.items[i] = r.Ticket
		}
		m.totalCount = msg.page.Count
		m.hasMore = msg.page.Meta.HasMore
		m.afterCursor = msg.page.Meta.AfterCursor
		m.users = make(map[int64]types.User)
		for _, u := range msg.page.Users {
			m.users[u.ID] = u
		}
		m.cursor = 0

	case moreTicketsLoadedMsg:
		m.loadingMore = false
		m.items = append(m.items, msg.page.Tickets...)
		m.hasMore = msg.page.Meta.HasMore
		m.afterCursor = msg.page.Meta.AfterCursor
		if m.users == nil {
			m.users = make(map[int64]types.User)
		}
		for _, u := range msg.page.Users {
			m.users[u.ID] = u
		}
		if m.knownTicketIDs == nil {
			m.knownTicketIDs = make(map[int64]bool)
		}
		for _, t := range msg.page.Tickets {
			m.knownTicketIDs[t.ID] = true
		}
		if msg.page.Count > 0 {
			m.totalCount = msg.page.Count
		}

	case moreSearchResultsMsg:
		m.loadingMore = false
		for _, r := range msg.page.Results {
			m.items = append(m.items, r.Ticket)
		}
		m.hasMore = msg.page.Meta.HasMore
		m.afterCursor = msg.page.Meta.AfterCursor
		if m.users == nil {
			m.users = make(map[int64]types.User)
		}
		for _, u := range msg.page.Users {
			m.users[u.ID] = u
		}
		if msg.page.Count > 0 {
			m.totalCount = msg.page.Count
		}

	case refreshLoadedMsg:
		m.loadingMore = false
		newKnown := make(map[int64]bool)
		newCount := 0
		for _, t := range msg.page.Tickets {
			newKnown[t.ID] = true
			if !m.knownTicketIDs[t.ID] {
				m.newTicketIDs[t.ID] = true
				newCount++
			}
		}
		m.lastRefreshNewCount = newCount
		m.knownTicketIDs = newKnown
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
		// Clamp cursor to valid range
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.refreshCountdown = refreshIntervalSeconds
		return m, scheduleCountdownTick()

	case errMsg:
		if m.loadingMore {
			m.loadingMore = false
			return m, nil
		}
		m.loading = false
		m.err = msg.err

	case spinner.TickMsg:
		if m.loading || m.loadingMore {
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
				return m, m.emitCursorChanged()
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
				cmds := []tea.Cmd{m.emitCursorChanged()}
				// Auto-load more when reaching last item
				if m.cursor == len(m.items)-1 && m.hasMore && !m.loadingMore {
					m.loadingMore = true
					cmds = append(cmds, m.spinner.Tick, m.triggerLoadMore())
				}
				return m, tea.Batch(cmds...)
			}
		case key.Matches(msg, keys.NextPage):
			if m.hasMore && !m.loadingMore {
				m.loadingMore = true
				return m, tea.Batch(m.spinner.Tick, m.triggerLoadMore())
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

func (m listModel) triggerLoadMore() tea.Cmd {
	if m.searchQuery != "" {
		return m.loadMoreSearch()
	}
	return m.loadMoreTickets()
}

func (m listModel) emitCursorChanged() tea.Cmd {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		id := m.items[m.cursor].ID
		return func() tea.Msg { return cursorChangedMsg{id: id} }
	}
	return nil
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
	var countLabel string
	if m.searchQuery != "" {
		if m.totalCount > len(m.items) {
			countLabel = fmt.Sprintf("Showing %d of %d results for %q", len(m.items), m.totalCount, m.searchQuery)
		} else {
			countLabel = fmt.Sprintf("%d results for %q", len(m.items), m.searchQuery)
		}
	} else if m.totalCount > len(m.items) {
		countLabel = fmt.Sprintf("Showing %d of %d tickets", len(m.items), m.totalCount)
	} else {
		countLabel = fmt.Sprintf("%d tickets", len(m.items))
	}
	header := titleStyle.Render("Tickets") + "  " + subtitleStyle.Render(countLabel)
	if m.autoRefresh {
		mins := m.refreshCountdown / 60
		secs := m.refreshCountdown % 60
		header += "  " + newTicketStyle.Render(fmt.Sprintf("↻ auto-refresh (%d:%02d)", mins, secs))
	}
	b.WriteString(header + "\n\n")

	// Calculate visible rows
	visibleRows := m.height - 6 // header + help bar
	if m.hasMore || m.loadingMore {
		visibleRows-- // reserve line for bottom indicator
	}
	if m.showChart && len(m.items) > 1 {
		visibleRows -= chartHeight
	}
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

	// Bottom indicator
	if m.loadingMore {
		b.WriteString(m.spinner.View() + " Loading more tickets...")
	} else if m.hasMore {
		b.WriteString(dimStyle.Render("  ↓ press n or scroll down to load more"))
	}

	// Status distribution chart
	if m.showChart && len(m.items) > 1 {
		b.WriteString("\n")
		b.WriteString(renderStatusChart(m.items, m.width, chartHeight))
	}

	return b.String()
}

func (m listModel) renderTicketRow(t types.Ticket, selected bool) string {
	isNew := m.newTicketIDs[t.ID]

	pointer := "  "
	if selected {
		pointer = "▸ "
	} else if isNew {
		pointer = "★ "
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
	if isNew {
		return newTicketStyle.Render(row)
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
