package tui

import "github.com/charmbracelet/lipgloss"

var (
	statusColors = map[string]lipgloss.AdaptiveColor{
		"new":     {Light: "#6B21A8", Dark: "#C084FC"},
		"open":    {Light: "#B91C1C", Dark: "#F87171"},
		"pending": {Light: "#B45309", Dark: "#FBBF24"},
		"hold":    {Light: "#4B5563", Dark: "#9CA3AF"},
		"solved":  {Light: "#15803D", Dark: "#4ADE80"},
		"closed":  {Light: "#374151", Dark: "#6B7280"},
	}

	priorityColors = map[string]lipgloss.AdaptiveColor{
		"urgent": {Light: "#991B1B", Dark: "#FCA5A5"},
		"high":   {Light: "#C2410C", Dark: "#FB923C"},
		"normal": {Light: "#1D4ED8", Dark: "#93C5FD"},
		"low":    {Light: "#4B5563", Dark: "#9CA3AF"},
	}

	statusIcons = map[string]string{
		"new":     "○",
		"open":    "●",
		"pending": "◉",
		"hold":    "◎",
		"solved":  "✓",
		"closed":  "✗",
	}

	priorityIcons = map[string]string{
		"urgent": "⬆",
		"high":   "▲",
		"normal": "■",
		"low":    "▼",
	}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#FAFAFA"})

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"})

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#FAFAFA"})

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6B7280"})

	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).
			Background(lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}).
			Bold(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"})

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F9FAFB"}).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"})

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}).
			Width(14)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F3F4F6"})

	commentAuthorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"})

	commentTimeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"})

	internalNoteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}).
				Italic(true)

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#4338CA", Dark: "#A5B4FC"}).
			Background(lipgloss.AdaptiveColor{Light: "#EEF2FF", Dark: "#1E1B4B"}).
			Padding(0, 1)

	newTicketStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"})

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"})

	focusBorderStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#93C5FD"})

	chartTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"})

	chartLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})

	cmdPaletteCategoryStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"})

	cmdPaletteSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#1a1a2e"}).
				Background(lipgloss.AdaptiveColor{Light: "#F9A87A", Dark: "#F9A87A"})

	cmdPaletteShortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"})

	cmdPaletteItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"})

	kanbanCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}).
			Padding(0, 1)

	kanbanCardSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1)

	kanbanColumnHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}).
				Padding(0, 1)
)

func styledStatus(status string) string {
	color, ok := statusColors[status]
	if !ok {
		return dimStyle.Render("? " + status)
	}
	icon, ok := statusIcons[status]
	if !ok {
		icon = "?"
	}
	return lipgloss.NewStyle().Foreground(color).Render(icon + " " + status)
}

func styledPriority(priority string) string {
	if priority == "" {
		return dimStyle.Render("- none")
	}
	color, ok := priorityColors[priority]
	if !ok {
		color = priorityColors["normal"]
	}
	icon, ok := priorityIcons[priority]
	if !ok {
		icon = "■"
	}
	return lipgloss.NewStyle().Foreground(color).Render(icon + " " + priority)
}
