package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderLogo(t *testing.T) {
	out := renderLogo()
	if out == "" {
		t.Fatal("expected non-empty logo")
	}
	w := lipgloss.Width(out)
	if w < 3 || w > 20 {
		t.Errorf("unexpected logo width: %d", w)
	}
}
