package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/johanviberg/zd/internal/types"
)

func TestCCPickerAddEntry(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "alice@example.com"})
	require.Len(t, m.selected, 1, "expected 1 selected")
	assert.Equal(t, "alice@example.com", m.selected[0].Email)
}

func TestCCPickerAddUserByID(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{UserID: 123, Name: "Alice", Email: "alice@example.com"})
	require.Len(t, m.selected, 1, "expected 1 selected")
	assert.Equal(t, int64(123), m.selected[0].UserID, "expected user ID 123")
}

func TestCCPickerDuplicateEmail(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "alice@example.com"})
	m = m.addEntry(types.CollaboratorEntry{Email: "alice@example.com"})
	require.Len(t, m.selected, 1, "expected 1 after duplicate")
}

func TestCCPickerDuplicateUserID(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{UserID: 123, Email: "a@example.com"})
	m = m.addEntry(types.CollaboratorEntry{UserID: 123, Email: "b@example.com"})
	require.Len(t, m.selected, 1, "expected 1 after duplicate user ID")
}

func TestCCPickerMaxLimit(t *testing.T) {
	m := newCCPickerModel(nil)
	for i := 0; i < maxCCs+5; i++ {
		m = m.addEntry(types.CollaboratorEntry{Email: fmt.Sprintf("user%d@example.com", i)})
	}
	require.Len(t, m.selected, maxCCs, "expected %d max", maxCCs)
}

func TestCCPickerRemoveLast(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "a@example.com"})
	m = m.addEntry(types.CollaboratorEntry{Email: "b@example.com"})
	m = m.removeLastEntry()
	require.Len(t, m.selected, 1, "expected 1 after remove")
	assert.Equal(t, "a@example.com", m.selected[0].Email, "expected a@example.com remaining")
}

func TestCCPickerRemoveFromEmpty(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.removeLastEntry() // should not panic
	require.Len(t, m.selected, 0, "expected 0")
}

func TestCCPickerReset(t *testing.T) {
	m := newCCPickerModel(nil)
	m = m.addEntry(types.CollaboratorEntry{Email: "a@example.com"})
	m.active = true
	m = m.reset()
	require.Len(t, m.selected, 0, "expected 0 after reset")
	assert.False(t, m.active, "expected inactive after reset")
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
		assert.Equal(t, tt.want, looksLikeEmail(tt.input), "looksLikeEmail(%q)", tt.input)
	}
}
