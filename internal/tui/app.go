package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/johanviberg/zd/internal/browser"
	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

type errMsg struct{ err error }

type cursorSettledMsg struct {
	seq uint64
	id  int64
}

type viewState int

const (
	listView viewState = iota
	detailView
	splitView
	kanbanView
)

type panelFocus int

const (
	focusList panelFocus = iota
	focusDetail
)

type currentUserMsg struct{ user *types.User }

type App struct {
	tickets     zendesk.TicketService
	search      zendesk.SearchService
	users       zendesk.UserService
	subdomain   string
	currentUser *types.User
	state       viewState
	prevState   viewState // saved state when entering detail from kanban
	list        listModel
	detail      detailModel
	kanban      kanbanModel
	actions     actionsModel
	searchM     searchModel
	gotoM       gotoModel
	cmdPalette  cmdPaletteModel
	width       int
	height      int
	focus       panelFocus
	showDetail  bool
	version     string
	cursorSeq   uint64
}

func NewApp(tickets zendesk.TicketService, search zendesk.SearchService, users zendesk.UserService, subdomain, version string) App {
	return App{
		tickets:    tickets,
		search:     search,
		users:      users,
		subdomain:  subdomain,
		version:    version,
		state:      splitView,
		showDetail: true,
		focus:      focusList,
		list:       newListModel(tickets, search),
		detail:     newDetailModel(tickets),
		kanban:     newKanbanModel(),
		actions:    newActionsModel(tickets, users),
		searchM:    newSearchModel(),
		gotoM:      newGotoModel(),
		cmdPalette: newCmdPaletteModel(),
	}
}

func (m App) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), m.fetchCurrentUser(), tea.SetWindowTitle("zd — Loading..."))
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

func (m App) listPanelWidth() int {
	if m.state != splitView || !m.showDetail {
		return m.width
	}
	return (m.width - 1) / 2 // -1 for divider
}

func (m App) detailPanelWidth() int {
	return m.width - m.listPanelWidth() - 1 // -1 for divider
}

func (m App) autoLoadFirstTicket() tea.Cmd {
	if m.state != splitView || !m.showDetail {
		return nil
	}
	if len(m.list.items) == 0 {
		return nil
	}
	id := m.list.items[m.list.cursor].ID
	m.detail = newDetailModel(m.tickets)
	m.detail.expectedID = id
	w := m.detailPanelWidth()
	m.detail.width = w
	m.detail.height = m.height
	return tea.Batch(m.detail.spinner.Tick, m.detail.loadTicket(id))
}

func (m *App) loadDetailForCursor() tea.Cmd {
	if len(m.list.items) == 0 {
		return nil
	}
	id := m.list.items[m.list.cursor].ID
	// Don't reload if already showing this ticket
	if m.detail.ticket != nil && m.detail.ticket.ID == id {
		return nil
	}
	m.detail = newDetailModel(m.tickets)
	m.detail.expectedID = id
	w := m.detailPanelWidth()
	m.detail.width = w
	m.detail.height = m.height
	return tea.Batch(m.detail.spinner.Tick, m.detail.loadTicket(id))
}

func (m App) windowTitle() string {
	switch m.state {
	case detailView:
		if m.detail.ticket != nil {
			subject := m.detail.ticket.Subject
			if len([]rune(subject)) > 50 {
				subject = string([]rune(subject)[:50]) + "…"
			}
			return fmt.Sprintf("zd — #%d: %s", m.detail.ticket.ID, subject)
		}
		return "zd — Loading..."
	case listView, splitView, kanbanView:
		if m.list.loading {
			return "zd — Loading..."
		}
		if len(m.list.items) == 0 {
			return "zd — No tickets"
		}
		if m.list.searchQuery != "" {
			q := m.list.searchQuery
			if len([]rune(q)) > 40 {
				q = string([]rune(q)[:40]) + "…"
			}
			return fmt.Sprintf("zd — Search: %q (%d results)", q, len(m.list.items))
		}
		newCount := len(m.list.newTicketIDs)
		if newCount > 0 {
			return fmt.Sprintf("zd — %d tickets (%d new)", len(m.list.items), newCount)
		}
		return fmt.Sprintf("zd — %d tickets", len(m.list.items))
	}
	return "zd"
}

func (m App) updateWindowTitle() tea.Cmd {
	return tea.SetWindowTitle(m.windowTitle())
}

func ringBell() tea.Cmd {
	return func() tea.Msg {
		os.Stderr.Write([]byte("\a"))
		return nil
	}
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Auto-collapse to list-only on narrow terminals
		if m.width < 80 && m.state == splitView {
			m.state = listView
			m.showDetail = false
		}

		// Kanban too narrow — switch back to list
		if m.width < 40 && m.state == kanbanView {
			m.state = listView
		}

		// Update kanban dimensions
		m.kanban.width = m.width
		m.kanban.height = m.height
		m.kanban.recomputeVisible()
		m.kanban.clampCursor()

		var cmds []tea.Cmd
		var cmd tea.Cmd

		// Send panel-appropriate sizes
		listMsg := tea.WindowSizeMsg{Width: m.listPanelWidth(), Height: msg.Height}
		m.list, cmd = m.list.Update(listMsg)
		cmds = append(cmds, cmd)

		if m.state == splitView && m.showDetail {
			detailMsg := tea.WindowSizeMsg{Width: m.detailPanelWidth(), Height: msg.Height}
			m.detail, cmd = m.detail.Update(detailMsg)
			cmds = append(cmds, cmd)
		} else {
			m.detail, cmd = m.detail.Update(msg)
			cmds = append(cmds, cmd)
		}

		m.actions, cmd = m.actions.Update(msg)
		cmds = append(cmds, cmd)
		m.searchM, cmd = m.searchM.Update(msg)
		cmds = append(cmds, cmd)
		m.gotoM, cmd = m.gotoM.Update(msg)
		cmds = append(cmds, cmd)
		m.cmdPalette, cmd = m.cmdPalette.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Global quit — but not when in input mode
		if m.actions.mode == actionNone && !m.searchM.active && !m.gotoM.active && !m.cmdPalette.active {
			if key.Matches(msg, keys.Quit) {
				return m, tea.Quit
			}

			// ctrl+p: command palette
			if key.Matches(msg, keys.CommandPalette) {
				hasItems := len(m.list.items) > 0
				cmd := m.cmdPalette.open(m.state, m.focus, m.showDetail, m.list.hasMore, hasItems)
				return m, cmd
			}

			// Tab: toggle focus in split view
			if key.Matches(msg, keys.Tab) && m.state == splitView && m.showDetail {
				if m.focus == focusList {
					m.focus = focusDetail
				} else {
					m.focus = focusList
				}
				return m, nil
			}

			// w: toggle kanban view
			if key.Matches(msg, keys.ToggleKanban) && (m.state == listView || m.state == splitView || m.state == kanbanView) {
				return m.toggleKanbanView()
			}

			// v: toggle detail panel
			if key.Matches(msg, keys.ToggleDetail) && (m.state == splitView || m.state == listView) {
				cmd := m.toggleDetailPanel()
				return m, cmd
			}

			// m: toggle my tickets filter
			if key.Matches(msg, keys.MyTickets) && (m.state == listView || m.state == splitView || m.state == kanbanView) {
				return m.toggleMyTickets()
			}

			// Esc handling for split view
			if msg.String() == "esc" && m.state == splitView {
				if m.focus == focusDetail {
					m.focus = focusList
					return m, nil
				}
				if m.list.searchQuery != "" {
					m.list.searchQuery = ""
					m.list.loading = true
					return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
				}
				return m, nil
			}

			// Clear search results on esc in list view
			if msg.String() == "esc" && m.state == listView && m.list.searchQuery != "" {
				m.list.searchQuery = ""
				m.list.loading = true
				return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
			}

			// Clear search results on esc in kanban view
			if msg.String() == "esc" && m.state == kanbanView && m.list.searchQuery != "" {
				m.list.searchQuery = ""
				m.list.loading = true
				return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
			}
		}
	}

	// Route to active action overlay first
	if m.actions.mode != actionNone {
		switch msg.(type) {
		case tea.KeyMsg, spinner.TickMsg, ticketUpdatedMsg, actionErrMsg, ccAutocompleteMsg, ccAutocompleteErrMsg:
			var cmd tea.Cmd
			m.actions, cmd = m.actions.Update(msg)
			if _, ok := msg.(ticketUpdatedMsg); ok {
				m.list.loading = true
				return m, tea.Batch(cmd, m.list.spinner.Tick, m.list.loadTickets())
			}
			return m, cmd
		}
	}

	// Route to command palette overlay
	if m.cmdPalette.active {
		if _, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.cmdPalette, cmd = m.cmdPalette.Update(msg)
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

	// Route to goto overlay
	if m.gotoM.active {
		if _, ok := msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			m.gotoM, cmd = m.gotoM.Update(msg)
			return m, cmd
		}
	}

	// Handle cross-cutting messages
	switch msg := msg.(type) {
	case imageOpenMsg:
		browser.Open(msg.url)
		return m, nil

	case currentUserMsg:
		m.currentUser = msg.user
		return m, nil

	case ticketLoadedMsg:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		if m.state == detailView {
			return m, tea.Batch(cmd, m.updateWindowTitle())
		}
		return m, cmd

	case countdownTickMsg:
		if !m.list.autoRefresh {
			return m, nil
		}
		m.list.refreshCountdown--
		if m.list.refreshCountdown <= 0 {
			if (m.state == listView || m.state == splitView || m.state == kanbanView) && m.list.searchQuery == "" && !m.list.loading {
				return m, m.list.loadTicketsForRefresh()
			}
			m.list.refreshCountdown = refreshIntervalSeconds
		}
		return m, scheduleCountdownTick()

	case refreshLoadedMsg:
		m.list.loading = false
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds := []tea.Cmd{cmd, m.updateWindowTitle()}
		// Reload detail if in split view
		if m.state == splitView && m.showDetail {
			if loadCmd := m.loadDetailForCursor(); loadCmd != nil {
				cmds = append(cmds, loadCmd)
			}
		}
		// Rebuild kanban columns
		if m.state == kanbanView {
			m.kanban.rebuildColumns(m.list.items)
		}
		// Ring bell when new tickets found
		if m.list.lastRefreshNewCount > 0 {
			cmds = append(cmds, ringBell())
		}
		return m, tea.Batch(cmds...)

	case ticketsLoadedMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds := []tea.Cmd{cmd, m.updateWindowTitle()}
		if m.state == splitView && m.showDetail {
			if len(m.list.items) > 0 {
				// Auto-load first ticket
				id := m.list.items[m.list.cursor].ID
				m.detail = newDetailModel(m.tickets)
				m.detail.expectedID = id
				m.detail.width = m.detailPanelWidth()
				m.detail.height = m.height
				cmds = append(cmds, m.detail.spinner.Tick, m.detail.loadTicket(id))
			} else {
				// Clear detail panel when no tickets
				m.detail = newDetailModel(m.tickets)
				m.detail.loading = false
				m.detail.width = m.detailPanelWidth()
				m.detail.height = m.height
			}
		}
		if m.state == kanbanView {
			m.kanban.rebuildColumns(m.list.items)
		}
		return m, tea.Batch(cmds...)

	case searchResultsMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds := []tea.Cmd{cmd, m.updateWindowTitle()}
		if m.state == splitView && m.showDetail {
			if len(m.list.items) > 0 {
				// Auto-load first result
				id := m.list.items[m.list.cursor].ID
				m.detail = newDetailModel(m.tickets)
				m.detail.expectedID = id
				m.detail.width = m.detailPanelWidth()
				m.detail.height = m.height
				cmds = append(cmds, m.detail.spinner.Tick, m.detail.loadTicket(id))
			} else {
				// Clear detail panel when no results
				m.detail = newDetailModel(m.tickets)
				m.detail.loading = false
				m.detail.width = m.detailPanelWidth()
				m.detail.height = m.height
			}
		}
		if m.state == kanbanView {
			m.kanban.rebuildColumns(m.list.items)
		}
		return m, tea.Batch(cmds...)

	case moreTicketsLoadedMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if m.state == kanbanView {
			m.kanban.rebuildColumns(m.list.items)
		}
		return m, cmd

	case moreSearchResultsMsg:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if m.state == kanbanView {
			m.kanban.rebuildColumns(m.list.items)
		}
		return m, cmd

	case cursorChangedMsg:
		if m.state == splitView && m.showDetail {
			m.cursorSeq++
			seq := m.cursorSeq
			id := msg.id
			return m, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return cursorSettledMsg{seq: seq, id: id}
			})
		}
		return m, nil

	case cursorSettledMsg:
		if msg.seq != m.cursorSeq {
			return m, nil // stale — user moved again
		}
		if m.state == splitView && m.showDetail {
			if m.detail.ticket != nil && m.detail.ticket.ID == msg.id {
				return m, nil
			}
			m.detail = newDetailModel(m.tickets)
			m.detail.expectedID = msg.id
			m.detail.width = m.detailPanelWidth()
			m.detail.height = m.height
			return m, tea.Batch(m.detail.spinner.Tick, m.detail.loadTicket(msg.id))
		}
		return m, nil

	case showDetailMsg:
		delete(m.list.newTicketIDs, msg.id)
		m.prevState = m.state
		if m.state == splitView {
			// If detail already has this ticket, just switch to full-screen
			if m.detail.ticket != nil && m.detail.ticket.ID == msg.id {
				m.state = detailView
				m.detail.width = m.width
				m.detail.height = m.height
				m.detail.viewport.Width = m.width - 4
				m.detail.viewport.Height = m.height - 6
				m.detail.viewport.SetContent(m.detail.renderContent())
				return m, m.updateWindowTitle()
			}
		}
		m.state = detailView
		m.detail = newDetailModel(m.tickets)
		m.detail.width = m.width
		m.detail.height = m.height
		return m, tea.Batch(m.detail.spinner.Tick, m.detail.loadTicket(msg.id), tea.SetWindowTitle("zd — Loading..."))

	case goBackMsg:
		if m.prevState == kanbanView {
			m.state = kanbanView
			return m, m.updateWindowTitle()
		}
		if m.showDetail {
			m.state = splitView
			m.focus = focusList
			// Resize detail to panel dimensions
			m.detail.width = m.detailPanelWidth()
			m.detail.height = m.height
			if m.detail.ready {
				m.detail.viewport.Width = m.detail.width - 4
				m.detail.viewport.Height = m.detail.height - 6
				m.detail.viewport.SetContent(m.detail.renderContent())
			}
			// Resize list to panel width
			listMsg := tea.WindowSizeMsg{Width: m.listPanelWidth(), Height: m.height}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(listMsg)
			return m, tea.Batch(cmd, m.updateWindowTitle())
		}
		m.state = listView
		return m, m.updateWindowTitle()

	case searchDoneMsg:
		m.list.searchQuery = msg.query
		m.list.loading = true
		return m, tea.Batch(m.list.spinner.Tick, m.list.doSearch(msg.query), tea.SetWindowTitle(fmt.Sprintf("zd — Searching: %q", msg.query)))

	case searchCancelMsg:
		if m.list.searchQuery != "" {
			m.list.searchQuery = ""
			m.list.loading = true
			return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
		}
		return m, m.updateWindowTitle()

	case gotoDoneMsg:
		return m, func() tea.Msg { return showDetailMsg{id: msg.id} }

	case gotoCancelMsg:
		return m, nil

	case cmdPaletteActionMsg:
		return m.handlePaletteAction(msg.action)
	}

	// Route to active view
	switch m.state {
	case splitView:
		if msg, ok := msg.(tea.KeyMsg); ok {
			// Toggle chart
			if key.Matches(msg, keys.ToggleChart) {
				m.list.showChart = !m.list.showChart
				return m, nil
			}
			// Toggle tags column
			if key.Matches(msg, keys.ToggleTags) {
				m.list.showTags = !m.list.showTags
				return m, nil
			}
			// Action keys work regardless of focus
			if len(m.list.items) > 0 {
				t := m.list.items[m.list.cursor]
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
				case key.Matches(msg, keys.GoTo):
					var cmd tea.Cmd
					m.gotoM, cmd = m.gotoM.open()
					return m, cmd
				case key.Matches(msg, keys.Comment):
					var cmd tea.Cmd
					m.actions, cmd = m.actions.openComment(t.ID)
					return m, cmd
				case key.Matches(msg, keys.Status):
					m.actions = m.actions.openStatus(t.ID, t.Status)
					return m, nil
				case key.Matches(msg, keys.Priority):
					m.actions = m.actions.openPriority(t.ID, t.Priority)
					return m, nil
				case key.Matches(msg, keys.Open):
					browser.Open(fmt.Sprintf("https://%s.zendesk.com/agent/tickets/%d", m.subdomain, t.ID))
					return m, nil
				case key.Matches(msg, keys.Enter):
					return m, func() tea.Msg {
						return showDetailMsg{id: t.ID}
					}
				}
			} else {
				// No items but still handle search/refresh/goto
				switch {
				case key.Matches(msg, keys.Search):
					var cmd tea.Cmd
					m.searchM, cmd = m.searchM.open()
					return m, cmd
				case key.Matches(msg, keys.GoTo):
					var cmd tea.Cmd
					m.gotoM, cmd = m.gotoM.open()
					return m, cmd
				case key.Matches(msg, keys.Refresh):
					m.list.autoRefresh = !m.list.autoRefresh
					if m.list.autoRefresh {
						m.list.refreshCountdown = refreshIntervalSeconds
						return m, scheduleCountdownTick()
					}
					return m, nil
				case key.Matches(msg, keys.ManualRefresh):
					if !m.list.loading {
						m.list.loading = true
						return m, tea.Batch(m.list.spinner.Tick, m.list.loadTicketsForRefresh())
					}
				}
			}

			// Route navigation keys to focused panel
			if m.focus == focusDetail {
				var cmd tea.Cmd
				m.detail, cmd = m.detail.Update(msg)
				return m, cmd
			}
			// focusList: route to list
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Non-key messages: route to both
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case listView:
		// Check for action keys before routing to list
		if msg, ok := msg.(tea.KeyMsg); ok {
			// Toggle chart
			if key.Matches(msg, keys.ToggleChart) {
				m.list.showChart = !m.list.showChart
				return m, nil
			}
			// Toggle tags column
			if key.Matches(msg, keys.ToggleTags) {
				m.list.showTags = !m.list.showTags
				return m, nil
			}
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
			case key.Matches(msg, keys.GoTo):
				var cmd tea.Cmd
				m.gotoM, cmd = m.gotoM.open()
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
		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, keys.GoTo) {
				var cmd tea.Cmd
				m.gotoM, cmd = m.gotoM.open()
				return m, cmd
			}
			if m.detail.ticket != nil {
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
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case kanbanView:
		if msg, ok := msg.(tea.KeyMsg); ok {
			// Action keys using selected ticket
			if t := m.kanban.selectedTicket(); t != nil {
				switch {
				case key.Matches(msg, keys.Comment):
					var cmd tea.Cmd
					m.actions, cmd = m.actions.openComment(t.ID)
					return m, cmd
				case key.Matches(msg, keys.Status):
					m.actions = m.actions.openStatus(t.ID, t.Status)
					return m, nil
				case key.Matches(msg, keys.Priority):
					m.actions = m.actions.openPriority(t.ID, t.Priority)
					return m, nil
				case key.Matches(msg, keys.Open):
					browser.Open(fmt.Sprintf("https://%s.zendesk.com/agent/tickets/%d", m.subdomain, t.ID))
					return m, nil
				}
			}

			switch {
			case key.Matches(msg, keys.Search):
				var cmd tea.Cmd
				m.searchM, cmd = m.searchM.open()
				return m, cmd
			case key.Matches(msg, keys.GoTo):
				var cmd tea.Cmd
				m.gotoM, cmd = m.gotoM.open()
				return m, cmd
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
			case key.Matches(msg, keys.NextPage):
				if m.list.hasMore && !m.list.loadingMore {
					m.list.loadingMore = true
					if m.list.searchQuery != "" {
						return m, m.list.loadMoreSearch()
					}
					return m, m.list.loadMoreTickets()
				}
			}

			// Route navigation to kanban model
			var cmd tea.Cmd
			m.kanban, cmd = m.kanban.Update(msg)
			return m, cmd
		}

		// Non-key messages: route to list (for spinner ticks etc.)
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *App) toggleDetailPanel() tea.Cmd {
	m.showDetail = !m.showDetail
	if m.showDetail {
		m.state = splitView
		listMsg := tea.WindowSizeMsg{Width: m.listPanelWidth(), Height: m.height}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(listMsg)
		cmds := []tea.Cmd{cmd}
		if loadCmd := m.loadDetailForCursor(); loadCmd != nil {
			cmds = append(cmds, loadCmd)
		}
		return tea.Batch(cmds...)
	}
	m.state = listView
	m.focus = focusList
	listMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(listMsg)
	return cmd
}

func (m *App) toggleKanbanView() (tea.Model, tea.Cmd) {
	if m.state == kanbanView {
		// Return to previous list/split state
		if m.showDetail {
			m.state = splitView
			m.focus = focusList
			listMsg := tea.WindowSizeMsg{Width: m.listPanelWidth(), Height: m.height}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(listMsg)
			cmds := []tea.Cmd{cmd, m.updateWindowTitle()}
			if loadCmd := m.loadDetailForCursor(); loadCmd != nil {
				cmds = append(cmds, loadCmd)
			}
			return m, tea.Batch(cmds...)
		}
		m.state = listView
		return m, m.updateWindowTitle()
	}
	// Enter kanban view
	m.kanban.width = m.width
	m.kanban.height = m.height
	m.kanban.rebuildColumns(m.list.items)
	m.state = kanbanView
	return m, m.updateWindowTitle()
}

func (m *App) toggleMyTickets() (tea.Model, tea.Cmd) {
	if m.currentUser == nil || m.currentUser.Email == "" {
		return m, nil
	}
	myQuery := "assignee:" + m.currentUser.Email
	if m.list.searchQuery == myQuery {
		// Toggle off: clear filter and reload all tickets
		m.list.searchQuery = ""
		m.list.loading = true
		return m, tea.Batch(m.list.spinner.Tick, m.list.loadTickets())
	}
	// Toggle on: search for my tickets
	m.list.searchQuery = myQuery
	m.list.loading = true
	return m, tea.Batch(m.list.spinner.Tick, m.list.doSearch(myQuery))
}

func (m *App) handlePaletteAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	case "quit":
		return m, tea.Quit
	case "enter":
		if m.state == kanbanView {
			if t := m.kanban.selectedTicket(); t != nil {
				id := t.ID
				return m, func() tea.Msg { return showDetailMsg{id: id} }
			}
		} else if len(m.list.items) > 0 {
			id := m.list.items[m.list.cursor].ID
			return m, func() tea.Msg { return showDetailMsg{id: id} }
		}
	case "goto":
		var cmd tea.Cmd
		m.gotoM, cmd = m.gotoM.open()
		return m, cmd
	case "search":
		var cmd tea.Cmd
		m.searchM, cmd = m.searchM.open()
		return m, cmd
	case "open":
		var id int64
		if m.state == detailView && m.detail.ticket != nil {
			id = m.detail.ticket.ID
		} else if m.state == kanbanView {
			if t := m.kanban.selectedTicket(); t != nil {
				id = t.ID
			}
		} else if len(m.list.items) > 0 {
			id = m.list.items[m.list.cursor].ID
		}
		if id > 0 {
			browser.Open(fmt.Sprintf("https://%s.zendesk.com/agent/tickets/%d", m.subdomain, id))
		}
		return m, nil
	case "comment":
		var id int64
		if m.state == detailView && m.detail.ticket != nil {
			id = m.detail.ticket.ID
		} else if m.state == kanbanView {
			if t := m.kanban.selectedTicket(); t != nil {
				id = t.ID
			}
		} else if len(m.list.items) > 0 {
			id = m.list.items[m.list.cursor].ID
		}
		if id > 0 {
			var cmd tea.Cmd
			m.actions, cmd = m.actions.openComment(id)
			return m, cmd
		}
	case "status":
		var id int64
		var status string
		if m.state == detailView && m.detail.ticket != nil {
			id = m.detail.ticket.ID
			status = m.detail.ticket.Status
		} else if m.state == kanbanView {
			if t := m.kanban.selectedTicket(); t != nil {
				id = t.ID
				status = t.Status
			}
		} else if len(m.list.items) > 0 {
			t := m.list.items[m.list.cursor]
			id = t.ID
			status = t.Status
		}
		if id > 0 {
			m.actions = m.actions.openStatus(id, status)
			return m, nil
		}
	case "priority":
		var id int64
		var priority string
		if m.state == detailView && m.detail.ticket != nil {
			id = m.detail.ticket.ID
			priority = m.detail.ticket.Priority
		} else if m.state == kanbanView {
			if t := m.kanban.selectedTicket(); t != nil {
				id = t.ID
				priority = t.Priority
			}
		} else if len(m.list.items) > 0 {
			t := m.list.items[m.list.cursor]
			id = t.ID
			priority = t.Priority
		}
		if id > 0 {
			m.actions = m.actions.openPriority(id, priority)
			return m, nil
		}
	case "my-tickets":
		return m.toggleMyTickets()
	case "toggle-kanban":
		return m.toggleKanbanView()
	case "toggle-detail":
		cmd := m.toggleDetailPanel()
		return m, cmd
	case "toggle-chart":
		m.list.showChart = !m.list.showChart
		return m, nil
	case "toggle-tags":
		m.list.showTags = !m.list.showTags
		return m, nil
	case "toggle-focus":
		if m.state == splitView && m.showDetail {
			if m.focus == focusList {
				m.focus = focusDetail
			} else {
				m.focus = focusList
			}
		}
		return m, nil
	case "refresh":
		if !m.list.loading {
			m.list.loading = true
			cmds := []tea.Cmd{m.list.spinner.Tick, m.list.loadTicketsForRefresh()}
			if m.list.autoRefresh {
				m.list.refreshCountdown = refreshIntervalSeconds
			}
			return m, tea.Batch(cmds...)
		}
	case "auto-refresh":
		m.list.autoRefresh = !m.list.autoRefresh
		if m.list.autoRefresh {
			m.list.refreshCountdown = refreshIntervalSeconds
			return m, scheduleCountdownTick()
		}
		m.list.newTicketIDs = make(map[int64]bool)
		return m, nil
	case "load-more":
		if m.list.hasMore && !m.list.loading {
			m.list.loadingMore = true
			if m.list.searchQuery != "" {
				return m, m.list.loadMoreSearch()
			}
			return m, m.list.loadMoreTickets()
		}
	}
	return m, nil
}

func (m App) View() string {
	// Overlay: action modal
	if m.actions.mode != actionNone {
		overlay := m.actions.View()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	// Overlay: command palette
	if m.cmdPalette.active {
		overlay := m.cmdPalette.View()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	var content string

	// Goto overlay (shown above list when active)
	if m.gotoM.active {
		content = m.gotoM.View() + "\n\n"
		switch m.state {
		case splitView:
			content += m.renderSplitView()
		case kanbanView:
			content += m.kanban.View()
		case detailView:
			content += m.detail.View()
		default:
			content += m.list.View()
		}
	} else if m.searchM.active {
		content = m.searchM.View() + "\n\n"
		switch m.state {
		case splitView:
			content += m.renderSplitView()
		case kanbanView:
			content += m.kanban.View()
		default:
			content += m.list.View()
		}
	} else {
		switch m.state {
		case listView:
			content = m.list.View()
		case detailView:
			content = m.detail.View()
		case splitView:
			content = m.renderSplitView()
		case kanbanView:
			content = m.kanban.View()
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
		Padding(2, 2).
		Render(content)

	return styledContent + "\n" + help
}

func (m App) renderSplitView() string {
	listWidth := m.listPanelWidth()
	detailWidth := m.detailPanelWidth()

	listContent := m.list.View()
	detailContent := m.detail.ViewPanel()

	// Apply focus indicator
	listPanel := lipgloss.NewStyle().Width(listWidth).Render(listContent)
	detailPanel := lipgloss.NewStyle().Width(detailWidth).Render(detailContent)

	if m.focus == focusList {
		listPanel = focusBorderStyle.Width(listWidth).Render(listContent)
	} else {
		detailPanel = focusBorderStyle.Width(detailWidth).Render(detailContent)
	}

	divider := m.renderDivider()

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, divider, detailPanel)
}

func (m App) renderDivider() string {
	height := m.height - 4
	if height < 1 {
		height = 1
	}
	divider := strings.Repeat("│\n", height-1) + "│"
	return dividerStyle.Render(divider)
}

func (m App) helpBar() string {
	var left string
	switch m.state {
	case listView:
		left = "↑↓ navigate  enter view  / search  m my-tickets  ctrl+p commands  q quit"
	case detailView:
		if len(m.detail.imageAttachments) > 0 {
			left = "↑↓ scroll  i images  esc back  ctrl+p commands  q quit"
		} else {
			left = "↑↓ scroll  esc back  ctrl+p commands  q quit"
		}
	case splitView:
		if m.focus == focusList {
			left = "↑↓ navigate  enter view  tab focus  m my-tickets  ctrl+p commands  q quit"
		} else if len(m.detail.imageAttachments) > 0 {
			left = "↑↓ scroll  i images  tab focus  esc back  ctrl+p commands  q quit"
		} else {
			left = "↑↓ scroll  tab focus  esc back  ctrl+p commands  q quit"
		}
	case kanbanView:
		left = "←→ columns  ↑↓ navigate  enter view  w list  m my-tickets  / search  ctrl+p commands  q quit"
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
	if m.version != "" {
		userInfo += "  " + m.version
	}

	// Right-align user info with padding
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(userInfo) - 2
	if gap < 2 {
		return left
	}
	return left + strings.Repeat(" ", gap) + userInfo
}
