package tui

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

func ac(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{
		Light: lipgloss.Color(light),
		Dark:  lipgloss.Color(dark),
	}
}

var (
	statusColors = map[string]compat.AdaptiveColor{
		"new":     ac("#6B21A8", "#C084FC"),
		"open":    ac("#B91C1C", "#F87171"),
		"pending": ac("#B45309", "#FBBF24"),
		"hold":    ac("#4B5563", "#9CA3AF"),
		"solved":  ac("#15803D", "#4ADE80"),
		"closed":  ac("#374151", "#6B7280"),
	}

	priorityColors = map[string]compat.AdaptiveColor{
		"urgent": ac("#991B1B", "#FCA5A5"),
		"high":   ac("#C2410C", "#FB923C"),
		"normal": ac("#1D4ED8", "#93C5FD"),
		"low":    ac("#4B5563", "#9CA3AF"),
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
			Foreground(ac("#1a1a2e", "#FAFAFA"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(ac("#4B5563", "#9CA3AF"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ac("#1a1a2e", "#FAFAFA"))

	dimStyle = lipgloss.NewStyle().
			Foreground(ac("#6B7280", "#6B7280"))

	helpBarStyle = lipgloss.NewStyle().
			Foreground(ac("#6B7280", "#9CA3AF")).
			Background(ac("#F3F4F6", "#1F2937"))

	errorStyle = lipgloss.NewStyle().
			Foreground(ac("#DC2626", "#F87171")).
			Bold(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ac("#D1D5DB", "#374151"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ac("#1F2937", "#F9FAFB")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ac("#D1D5DB", "#374151"))

	labelStyle = lipgloss.NewStyle().
			Foreground(ac("#6B7280", "#9CA3AF")).
			Width(14)

	valueStyle = lipgloss.NewStyle().
			Foreground(ac("#111827", "#F3F4F6"))

	commentAuthorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ac("#1F2937", "#E5E7EB"))

	commentTimeStyle = lipgloss.NewStyle().
				Foreground(ac("#9CA3AF", "#6B7280"))

	internalNoteStyle = lipgloss.NewStyle().
				Foreground(ac("#D97706", "#FBBF24")).
				Italic(true)

	tagStyle = lipgloss.NewStyle().
			Foreground(ac("#4338CA", "#A5B4FC")).
			Background(ac("#EEF2FF", "#1E1B4B")).
			Padding(0, 1)

	newTicketStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ac("#0E7490", "#22D3EE"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(ac("#D1D5DB", "#374151"))

	focusBorderStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ac("#1D4ED8", "#93C5FD"))

	chartTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ac("#374151", "#D1D5DB"))

	chartLabelStyle = lipgloss.NewStyle().
			Foreground(ac("#6B7280", "#9CA3AF"))

	cmdPaletteCategoryStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ac("#7C3AED", "#A78BFA"))

	cmdPaletteSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ac("#1a1a2e", "#1a1a2e")).
				Background(ac("#F9A87A", "#F9A87A"))

	cmdPaletteShortcutStyle = lipgloss.NewStyle().
				Foreground(ac("#9CA3AF", "#6B7280"))

	cmdPaletteItemStyle = lipgloss.NewStyle().
				Foreground(ac("#1F2937", "#E5E7EB"))

	kanbanCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ac("#D1D5DB", "#374151")).
			Padding(0, 1)

	kanbanCardSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1)

	kanbanColumnHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ac("#D1D5DB", "#374151")).
				Padding(0, 1)

	attachmentStyle = lipgloss.NewStyle().
			Foreground(ac("#0369A1", "#7DD3FC"))

	attachmentImageStyle = lipgloss.NewStyle().
				Foreground(ac("#7C3AED", "#A78BFA"))

	ccChipStyle = lipgloss.NewStyle().
			Foreground(ac("#1E40AF", "#93C5FD")).
			Background(ac("#DBEAFE", "#1E3A5F")).
			Padding(0, 1)

	ccDisabledStyle = lipgloss.NewStyle().
			Foreground(ac("#9CA3AF", "#6B7280")).
			Italic(true)

	ccResultHighlightStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ac("#1a1a2e", "#FAFAFA"))

	timelineConnectorStyle = lipgloss.NewStyle().
				Foreground(ac("#D1D5DB", "#4B5563"))

	timelineChangeStyle = lipgloss.NewStyle().
				Foreground(ac("#6B7280", "#9CA3AF"))

	timelineArrowStyle = lipgloss.NewStyle().
				Foreground(ac("#9CA3AF", "#6B7280"))

	logoAccentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ac("#7C3AED", "#A78BFA"))

	logoTextStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ac("#1D4ED8", "#93C5FD"))

	accentStyle = lipgloss.NewStyle().
			Foreground(ac("#1D4ED8", "#93C5FD")).
			Bold(true)
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
