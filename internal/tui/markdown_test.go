package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stripANSI removes ANSI escape sequences from s so tests can do plain-text
// assertions on glamour-rendered output.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestRenderMarkdown_HTMLContent(t *testing.T) {
	html := "<p>Hello <strong>world</strong></p>"
	result := renderMarkdown(html, "Hello world", 80)
	assert.Contains(t, result, "world", "expected rendered output to contain 'world'")
}

func TestRenderMarkdown_HTMLWithLinks(t *testing.T) {
	html := `<p>Visit <a href="https://example.com">our site</a></p>`
	result := renderMarkdown(html, "Visit our site", 80)
	assert.Contains(t, result, "our site", "expected rendered output to contain 'our site'")
}

func TestRenderMarkdown_HTMLWithList(t *testing.T) {
	html := "<ul><li>First</li><li>Second</li></ul>"
	result := renderMarkdown(html, "First\nSecond", 80)
	assert.Contains(t, result, "First", "expected 'First' in output")
	assert.Contains(t, result, "Second", "expected 'Second' in output")
}

func TestRenderMarkdown_EmptyHTMLFallback(t *testing.T) {
	result := renderMarkdown("", "Plain text content", 80)
	assert.Contains(t, result, "Plain text content", "expected plain text fallback")
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	html := "<pre><code>func main() {}</code></pre>"
	result := renderMarkdown(html, "func main() {}", 80)
	// Glamour v2 syntax-highlights code blocks, wrapping each token in ANSI
	// escape sequences. Strip ANSI before asserting on plain text content.
	assert.Contains(t, stripANSI(result), "func main()", "expected code block content")
}

func TestRenderMarkdown_WidthWrapping(t *testing.T) {
	longText := strings.Repeat("word ", 50)
	result := renderMarkdown("", longText, 40)
	for _, line := range strings.Split(result, "\n") {
		assert.LessOrEqual(t, len(line), 50, "line too long (%d chars): %q", len(line), line)
	}
}
