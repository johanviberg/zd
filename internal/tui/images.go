package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johanviberg/zd/internal/types"
)

// imageEntry represents a single image attachment with context.
type imageEntry struct {
	index      int // 1-based display number
	attachment types.Attachment
	authorName string
}

type imageOpenMsg struct {
	url string
}

type imagePickerCloseMsg struct{}

// imagePickerModel is an overlay that lists image attachments and lets the user open one.
type imagePickerModel struct {
	active bool
	items  []imageEntry
	cursor int
	width  int
	height int
}

func (m imagePickerModel) open(items []imageEntry) imagePickerModel {
	m.active = true
	m.items = items
	m.cursor = 0
	return m
}

func (m imagePickerModel) close() imagePickerModel {
	m.active = false
	return m
}

func (m imagePickerModel) Update(msg tea.Msg) (imagePickerModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			m = m.close()
			return m, func() tea.Msg { return imagePickerCloseMsg{} }
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if len(m.items) > 0 {
				url := m.items[m.cursor].attachment.ContentURL
				m = m.close()
				return m, func() tea.Msg { return imageOpenMsg{url: url} }
			}
		}
	}
	return m, nil
}

func (m imagePickerModel) View() string {
	if !m.active || len(m.items) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Image Attachments (%d)", len(m.items))) + "\n\n")

	for i, item := range m.items {
		pointer := "  "
		if i == m.cursor {
			pointer = "> "
		}

		label := fmt.Sprintf("[%d] %s  %s",
			item.index,
			item.attachment.FileName,
			dimStyle.Render(item.attachment.HumanSize()),
		)
		if item.authorName != "" {
			label += "  " + commentTimeStyle.Render(item.authorName)
		}

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(pointer+label) + "\n")
		} else {
			b.WriteString(pointer + label + "\n")
		}
	}

	b.WriteString("\n" + dimStyle.Render("↑↓ select   enter open   esc cancel"))

	return borderStyle.Padding(1, 2).Render(b.String())
}
