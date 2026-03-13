package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/johanviberg/zd/internal/types"
)

const (
	chartMaxBarHeight = 6
	chartHeight       = chartMaxBarHeight + 6 // bars + count + label + title + padding
)

var statusOrder = []string{"new", "open", "pending", "hold", "solved", "closed"}

func renderStatusChart(items []types.Ticket, width int, height int) string {
	if width < 30 || len(items) < 2 {
		return ""
	}

	// Count tickets per status
	counts := make(map[string]int)
	for _, t := range items {
		counts[t.Status]++
	}

	// Determine active statuses (non-zero count, in fixed order)
	var active []string
	for _, s := range statusOrder {
		if counts[s] > 0 {
			active = append(active, s)
		}
	}
	if len(active) == 0 {
		return ""
	}

	// Find max count for scaling
	maxCount := 0
	for _, s := range active {
		if counts[s] > maxCount {
			maxCount = counts[s]
		}
	}

	maxBarHeight := chartMaxBarHeight
	if h := height - 4; h > 0 && h < maxBarHeight {
		maxBarHeight = h
	}

	// Calculate bar heights
	barHeights := make([]int, len(active))
	for i, s := range active {
		h := int(float64(counts[s]) / float64(maxCount) * float64(maxBarHeight))
		if h < 1 {
			h = 1 // ensure visibility
		}
		barHeights[i] = h
	}

	// Column width: evenly spaced, minimum 6 chars
	colWidth := width / len(active)
	if colWidth < 6 {
		colWidth = 6
	}
	if colWidth > 12 {
		colWidth = 12
	}

	var b strings.Builder
	b.WriteString(chartTitleStyle.Render("Status Distribution"))
	b.WriteString("\n\n")

	// Render bars row by row, top to bottom
	for row := maxBarHeight; row >= 1; row-- {
		var line strings.Builder
		for i, s := range active {
			color := statusColors[s]
			barStyle := lipgloss.NewStyle().Foreground(color)
			cell := strings.Repeat(" ", colWidth)
			if barHeights[i] >= row {
				// Center the block characters in the column
				bar := "██"
				pad := (colWidth - 2) / 2
				cell = strings.Repeat(" ", pad) + barStyle.Render(bar) + strings.Repeat(" ", colWidth-2-pad)
			}
			if i > 0 {
				line.WriteString(" ")
			}
			line.WriteString(cell)
		}
		b.WriteString(line.String())
		b.WriteString("\n")
	}

	// Count row
	var countLine strings.Builder
	for i, s := range active {
		label := fmt.Sprintf("%d", counts[s])
		pad := (colWidth - len(label)) / 2
		if pad < 0 {
			pad = 0
		}
		cell := strings.Repeat(" ", pad) + chartLabelStyle.Render(label) + strings.Repeat(" ", colWidth-len(label)-pad)
		if i > 0 {
			countLine.WriteString(" ")
		}
		countLine.WriteString(cell)
	}
	b.WriteString(countLine.String())
	b.WriteString("\n")

	// Label row (icon + full status name)
	var labelLine strings.Builder
	for i, s := range active {
		icon := statusIcons[s]
		color := statusColors[s]
		iconStyle := lipgloss.NewStyle().Foreground(color)
		label := iconStyle.Render(icon) + " " + chartLabelStyle.Render(s)
		visLen := 1 + 1 + len(s) // icon + space + status name
		pad := (colWidth - visLen) / 2
		if pad < 0 {
			pad = 0
		}
		cell := strings.Repeat(" ", pad) + label + strings.Repeat(" ", colWidth-visLen-pad)
		if i > 0 {
			labelLine.WriteString(" ")
		}
		labelLine.WriteString(cell)
	}
	b.WriteString(labelLine.String())
	b.WriteString("\n\n")

	return b.String()
}
