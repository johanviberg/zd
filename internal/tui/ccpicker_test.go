package tui

import (
	"fmt"
	"testing"

	"github.com/johanviberg/zd/internal/types"
)

func TestCCPickerAddEntry(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "alice@example.com"})
	if len(m.selected) != 1 {
		t.Fatalf("expected 1 selected, got %d", len(m.selected))
	}
	if m.selected[0].Email != "alice@example.com" {
		t.Errorf("expected alice@example.com, got %s", m.selected[0].Email)
	}
}

func TestCCPickerAddUserByID(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{UserID: 123, Name: "Alice", Email: "alice@example.com"})
	if len(m.selected) != 1 {
		t.Fatalf("expected 1 selected, got %d", len(m.selected))
	}
	if m.selected[0].UserID != 123 {
		t.Errorf("expected user ID 123, got %d", m.selected[0].UserID)
	}
}

func TestCCPickerDuplicateEmail(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "alice@example.com"})
	m = m.addEntry(types.CollaboratorEntry{Email: "alice@example.com"})
	if len(m.selected) != 1 {
		t.Fatalf("expected 1 after duplicate, got %d", len(m.selected))
	}
}

func TestCCPickerDuplicateUserID(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{UserID: 123, Email: "a@example.com"})
	m = m.addEntry(types.CollaboratorEntry{UserID: 123, Email: "b@example.com"})
	if len(m.selected) != 1 {
		t.Fatalf("expected 1 after duplicate user ID, got %d", len(m.selected))
	}
}

func TestCCPickerMaxLimit(t *testing.T) {
	m := newCCPickerModel(nil)
	for i := 0; i < maxCCs+5; i++ {
		m = m.addEntry(types.CollaboratorEntry{Email: fmt.Sprintf("user%d@example.com", i)})
	}
	if len(m.selected) != maxCCs {
		t.Fatalf("expected %d max, got %d", maxCCs, len(m.selected))
	}
}

func TestCCPickerRemoveLast(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "a@example.com"})
	m = m.addEntry(types.CollaboratorEntry{Email: "b@example.com"})
	m = m.removeLastEntry()
	if len(m.selected) != 1 {
		t.Fatalf("expected 1 after remove, got %d", len(m.selected))
	}
	if m.selected[0].Email != "a@example.com" {
		t.Errorf("expected a@example.com remaining, got %s", m.selected[0].Email)
	}
}

func TestCCPickerRemoveFromEmpty(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.removeLastEntry() // should not panic
	if len(m.selected) != 0 {
		t.Fatalf("expected 0, got %d", len(m.selected))
	}
}

func TestCCPickerReset(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "a@example.com"})
	m.active = true
	m = m.reset()
	if len(m.selected) != 0 {
		t.Fatalf("expected 0 after reset, got %d", len(m.selected))
	}
	if m.active {
		t.Fatal("expected inactive after reset")
	}
}

func TestLooksLikeEmail(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"alice@example.com", true},
		{"user@domain.org", true},
		{"notanemail", false},
		{"@missing.domain", true}, // simple heuristic; has @ and .
		{"no-at-sign.com", false},
		{"has@but-no-dot", false},
	}
	for _, tt := range tests {
		if got := looksLikeEmail(tt.input); got != tt.want {
			t.Errorf("looksLikeEmail(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
