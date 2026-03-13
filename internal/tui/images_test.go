package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johanviberg/zd/internal/types"
)

func testImageEntries() []imageEntry {
	return []imageEntry{
		{index: 1, attachment: types.Attachment{ID: 10, FileName: "screenshot.png", ContentURL: "https://example.com/1.png", ContentType: "image/png", Size: 45000}, authorName: "Alice"},
		{index: 3, attachment: types.Attachment{ID: 30, FileName: "error.jpg", ContentURL: "https://example.com/3.jpg", ContentType: "image/jpeg", Size: 128000}, authorName: "Bob"},
		{index: 5, attachment: types.Attachment{ID: 50, FileName: "receipt.png", ContentURL: "https://example.com/5.png", ContentType: "image/png", Size: 23000}, authorName: "Alice"},
	}
}

func TestImagePickerOpen(t *testing.T) {
	m := imagePickerModel{}
	items := testImageEntries()
	m = m.open(items)

	require.True(t, m.active, "expected picker to be active")
	assert.Equal(t, 0, m.cursor, "expected cursor 0")
	assert.Len(t, m.items, 3, "expected 3 items")
}

func TestImagePickerNavigation(t *testing.T) {
	m := imagePickerModel{}
	m = m.open(testImageEntries())

	// Move down
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 1, m.cursor, "expected cursor 1 after down")

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 2, m.cursor, "expected cursor 2 after second down")

	// Should not go past last item
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 2, m.cursor, "expected cursor to stay at 2")

	// Move up
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.Equal(t, 1, m.cursor, "expected cursor 1 after up")

	// Should not go below 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.Equal(t, 0, m.cursor, "expected cursor 0")
}

func TestImagePickerEnter(t *testing.T) {
	m := imagePickerModel{}
	m = m.open(testImageEntries())

	// Move to second item
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// Press enter
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.False(t, m.active, "expected picker to close after enter")
	require.NotNil(t, cmd, "expected a command from enter")

	msg := cmd()
	openMsg, ok := msg.(imageOpenMsg)
	require.True(t, ok, "expected imageOpenMsg, got %T", msg)
	assert.Equal(t, "https://example.com/3.jpg", openMsg.url, "expected URL https://example.com/3.jpg")
}

func TestImagePickerEsc(t *testing.T) {
	m := imagePickerModel{}
	m = m.open(testImageEntries())

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.False(t, m.active, "expected picker to close after esc")
	require.NotNil(t, cmd, "expected a command from esc")

	msg := cmd()
	_, ok := msg.(imagePickerCloseMsg)
	require.True(t, ok, "expected imagePickerCloseMsg, got %T", msg)
}

func TestImagePickerViewEmpty(t *testing.T) {
	m := imagePickerModel{}
	assert.Equal(t, "", m.View(), "expected empty view when not active")

	m.active = true
	assert.Equal(t, "", m.View(), "expected empty view when no items")
}
