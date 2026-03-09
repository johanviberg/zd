package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up             key.Binding
	Down           key.Binding
	Left           key.Binding
	Right          key.Binding
	Enter          key.Binding
	Back           key.Binding
	Quit           key.Binding
	Search         key.Binding
	Comment        key.Binding
	Status         key.Binding
	Priority       key.Binding
	Submit         key.Binding
	Tab            key.Binding
	NextPage       key.Binding
	Refresh        key.Binding
	ManualRefresh  key.Binding
	Open           key.Binding
	ToggleDetail   key.Binding
	ToggleChart    key.Binding
	ToggleTags     key.Binding
	ToggleKanban   key.Binding
	GoTo           key.Binding
	CommandPalette key.Binding
	Images         key.Binding
	AddCC          key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Comment: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "comment"),
	),
	Status: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "status"),
	),
	Priority: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "priority"),
	),
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "load more"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "auto-refresh"),
	),
	ManualRefresh: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "refresh"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	ToggleDetail: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "toggle detail panel"),
	),
	ToggleChart: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "toggle chart"),
	),
	ToggleTags: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle tags"),
	),
	ToggleKanban: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "kanban view"),
	),
	GoTo: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "go to ticket"),
	),
	CommandPalette: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "commands"),
	),
	Images: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "images"),
	),
	AddCC: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "add CC"),
	),
}
