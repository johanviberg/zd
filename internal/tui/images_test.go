package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

	if !m.active {
		t.Fatal("expected picker to be active")
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	if len(m.items) != 3 {
		t.Errorf("expected 3 items, got %d", len(m.items))
	}
}

func TestImagePickerNavigation(t *testing.T) {
	m := imagePickerModel{}
	m = m.open(testImageEntries())

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after down, got %d", m.cursor)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor 2 after second down, got %d", m.cursor)
	}

	// Should not go past last item
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m.cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after up, got %d", m.cursor)
	}

	// Should not go below 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
}

func TestImagePickerEnter(t *testing.T) {
	m := imagePickerModel{}
	m = m.open(testImageEntries())

	// Move to second item
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press enter
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.active {
		t.Fatal("expected picker to close after enter")
	}

	if cmd == nil {
		t.Fatal("expected a command from enter")
	}

	msg := cmd()
	openMsg, ok := msg.(imageOpenMsg)
	if !ok {
		t.Fatalf("expected imageOpenMsg, got %T", msg)
	}
	if openMsg.url != "https://example.com/3.jpg" {
		t.Errorf("expected URL https://example.com/3.jpg, got %s", openMsg.url)
	}
}

func TestImagePickerEsc(t *testing.T) {
	m := imagePickerModel{}
	m = m.open(testImageEntries())

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.active {
		t.Fatal("expected picker to close after esc")
	}

	if cmd == nil {
		t.Fatal("expected a command from esc")
	}

	msg := cmd()
	if _, ok := msg.(imagePickerCloseMsg); !ok {
		t.Fatalf("expected imagePickerCloseMsg, got %T", msg)
	}
}

func TestImagePickerViewEmpty(t *testing.T) {
	m := imagePickerModel{}
	if m.View() != "" {
		t.Error("expected empty view when not active")
	}

	m.active = true
	if m.View() != "" {
		t.Error("expected empty view when no items")
	}
}
