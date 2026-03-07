package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/johanviberg/zd/internal/browser"
	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

type errMsg struct{ err error }

type viewState int

const (
	listView viewState = iota
	detailView
)

type currentUserMsg struct{ user *types.User }

type App struct {
	tickets     zendesk.TicketService
	search      zendesk.SearchService
	users       zendesk.UserService
	subdomain   string
	currentUser *types.User
	state       viewState
	list        listModel
	detail      detailModel
	actions     actionsModel
	searchM     searchModel
	width       int
	height      int
}

func NewApp(tickets zendesk.TicketService, search zendesk.SearchService, users zendesk.UserService, subdomain string) App {
	return App{
		tickets:   tickets,
		search:    search,
		users:     users,
		subdomain: subdomain,
		list:      newListModel(tickets, search),
		detail:    newDetailModel(tickets),
		actions:   newActionsModel(tickets),
		searchM:   newSearchModel(),
	}
}

func (m App) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), m.fetchCurrentUser())
}

func (m App) fetchCurrentUser() tea.Cmd {
	return func() tea.Msg {
		if m.users == nil {
			return currentUserMsg{}
		}
		user, err := m.users.GetMe(context.Background())
		if err != nil {
			return currentUserMsg{}
		}
		return currentUserMsg{user: user}
	}
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
		m.actions, cmd = m.actions.Update(msg)
		cmds = append(cmds, cmd)
		m.searchM, cmd = m.searchM.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Global quit — but not when in input mode
		if m.actions.mode == actionNone && !m.searchM.active {
			if key.Matches(msg, keys.Quit) {
				return m, tea.Quit
			}
			// Clear search results on esc in list view
			if msg.String() == "esc" && m.state == listView && m.list.searchQuery != "" {
				m.list.searchQuery = ""
				m.list.loading = true
				return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
			}
		}
	}

	// Route to active action overlay first
	if m.actions.mode != actionNone {
		switch msg.(type) {
		case tea.KeyMsg, spinner.TickMsg, ticketUpdatedMsg, actionErrMsg:
			var cmd tea.Cmd
			m.actions, cmd = m.actions.Update(msg)
			if _, ok := msg.(ticketUpdatedMsg); ok {
				m.list.loading = true
				return m, tea.Batch(cmd, m.list.spinner.Tick, m.list.loadTickets())
			}
			return m, cmd
		}
	}

	// Route to search overlay
	if m.searchM.active {
		if _, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.searchM, cmd = m.searchM.Update(msg)
			return m, cmd
		}
	}

	// Handle cross-cutting messages
	switch msg := msg.(type) {
	case currentUserMsg:
		m.currentUser = msg.user
		return m, nil

	case countdownTickMsg:
		if !m.list.autoRefresh {
			return m, nil
		}
		m.list.refreshCountdown--
		if m.list.refreshCountdown <= 0 {
			if m.state == listView && m.list.searchQuery == "" && !m.list.loading {
				return m, m.list.loadTicketsForRefresh()
			}
			// Can't refresh right now, reset and keep ticking
			m.list.refreshCountdown = refreshIntervalSeconds
		}
		return m, scheduleCountdownTick()

	case refreshLoadedMsg:
		m.list.loading = false
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case showDetailMsg:
		m.state = detailView
		delete(m.list.newTicketIDs, msg.id)
		m.detail = newDetailModel(m.tickets)
		m.detail.width = m.width
		m.detail.height = m.height
		return m, tea.Batch(m.detail.spinner.Tick, m.detail.loadTicket(msg.id))

	case goBackMsg:
		m.state = listView
		return m, nil

	case searchDoneMsg:
		m.list.searchQuery = msg.query
		m.list.loading = true
		return m, tea.Batch(m.list.spinner.Tick, m.list.doSearch(msg.query))

	case searchCancelMsg:
		if m.list.searchQuery != "" {
			m.list.searchQuery = ""
			m.list.loading = true
			return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
		}
		return m, nil
	}

	// Route to active view
	switch m.state {
	case listView:
		// Check for action keys before routing to list
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(msg, keys.Refresh):
				m.list.autoRefresh = !m.list.autoRefresh
				if m.list.autoRefresh {
					m.list.refreshCountdown = refreshIntervalSeconds
					return m, scheduleCountdownTick()
				}
				m.list.newTicketIDs = make(map[int64]bool)
				return m, nil
			case key.Matches(msg, keys.ManualRefresh):
				if !m.list.loading {
					m.list.loading = true
					cmds := []tea.Cmd{m.list.spinner.Tick, m.list.loadTicketsForRefresh()}
					if m.list.autoRefresh {
						m.list.refreshCountdown = refreshIntervalSeconds
					}
					return m, tea.Batch(cmds...)
				}
			case key.Matches(msg, keys.Search):
				var cmd tea.Cmd
				m.searchM, cmd = m.searchM.open()
				return m, cmd
			case key.Matches(msg, keys.Comment):
				if len(m.list.items) > 0 {
					t := m.list.items[m.list.cursor]
					var cmd tea.Cmd
					m.actions, cmd = m.actions.openComment(t.ID)
					return m, cmd
				}
			case key.Matches(msg, keys.Status):
				if len(m.list.items) > 0 {
					t := m.list.items[m.list.cursor]
					m.actions = m.actions.openStatus(t.ID, t.Status)
					return m, nil
				}
			case key.Matches(msg, keys.Priority):
				if len(m.list.items) > 0 {
					t := m.list.items[m.list.cursor]
					m.actions = m.actions.openPriority(t.ID, t.Priority)
					return m, nil
				}
			case key.Matches(msg, keys.Open):
				if len(m.list.items) > 0 {
					t := m.list.items[m.list.cursor]
					browser.Open(fmt.Sprintf("https://%s.zendesk.com/agent/tickets/%d", m.subdomain, t.ID))
					return m, nil
				}
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case detailView:
		// Check for action keys before routing to detail
		if msg, ok := msg.(tea.KeyMsg); ok && m.detail.ticket != nil {
			switch {
			case key.Matches(msg, keys.Comment):
				var cmd tea.Cmd
				m.actions, cmd = m.actions.openComment(m.detail.ticket.ID)
				return m, cmd
			case key.Matches(msg, keys.Status):
				m.actions = m.actions.openStatus(m.detail.ticket.ID, m.detail.ticket.Status)
				return m, nil
			case key.Matches(msg, keys.Priority):
				m.actions = m.actions.openPriority(m.detail.ticket.ID, m.detail.ticket.Priority)
				return m, nil
			case key.Matches(msg, keys.Open):
				browser.Open(fmt.Sprintf("https://%s.zendesk.com/agent/tickets/%d", m.subdomain, m.detail.ticket.ID))
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m App) View() string {
	// Overlay: action modal
	if m.actions.mode != actionNone {
		overlay := m.actions.View()
		// Center the overlay
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	var content string

	// Search bar (shown above list when active)
	if m.searchM.active {
		content = m.searchM.View() + "\n\n"
		if m.state == listView {
			content += m.list.View()
		}
	} else {
		switch m.state {
		case listView:
			content = m.list.View()
		case detailView:
			content = m.detail.View()
		}
	}

	// Help bar at the bottom
	helpText := m.helpBar()
	help := helpBarStyle.Width(m.width).Padding(0, 1).Render(helpText)

	// Layout: content takes remaining space, help bar at bottom
	contentHeight := m.height - lipgloss.Height(help) - 1
	styledContent := lipgloss.NewStyle().
		Height(contentHeight).
		MaxHeight(contentHeight).
		Padding(1, 2).
		Render(content)

	return styledContent + "\n" + help
}

func (m App) helpBar() string {
	var left string
	switch m.state {
	case listView:
		if m.list.searchQuery != "" {
			left = "↑↓/jk navigate  enter view  o open  / search  esc clear search  r auto-refresh  R refresh  c comment  s status  p priority  q quit"
		} else {
			left = "↑↓/jk navigate  enter view  o open  / search  r auto-refresh  R refresh  c comment  s status  p priority  q quit"
		}
	case detailView:
		left = "esc back  ↑↓ scroll  o open  c comment  s status  p priority  q quit"
	}

	if m.currentUser == nil || m.width == 0 {
		return left
	}

	userInfo := m.currentUser.Email
	if userInfo == "" {
		userInfo = m.currentUser.Name
	}
	if userInfo == "" {
		return left
	}

	// Right-align user info with padding
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(userInfo) - 2 // -2 for padding
	if gap < 2 {
		return left
	}
	return left + strings.Repeat(" ", gap) + userInfo
}
