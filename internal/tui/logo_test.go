package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderLogo(t *testing.T) {
	out := renderLogo()
	require.NotEmpty(t, out, "expected non-empty logo")
	w := lipgloss.Width(out)
	assert.True(t, w >= 3 && w <= 20, "unexpected logo width: %d", w)
}
