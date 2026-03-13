package tui

import (
	"strings"
	"testing"
)

func TestRenderMarkdown_HTMLContent(t *testing.T) {
	html := "<p>Hello <strong>world</strong></p>"
	result := renderMarkdown(html, "Hello world", 80)
	if !strings.Contains(result, "world") {
		t.Errorf("expected rendered output to contain 'world', got: %q", result)
	}
}

func TestRenderMarkdown_HTMLWithLinks(t *testing.T) {
	html := `<p>Visit <a href="https://example.com">our site</a></p>`
	result := renderMarkdown(html, "Visit our site", 80)
	if !strings.Contains(result, "our site") {
		t.Errorf("expected rendered output to contain 'our site', got: %q", result)
	}
}

func TestRenderMarkdown_HTMLWithList(t *testing.T) {
	html := "<ul><li>First</li><li>Second</li></ul>"
	result := renderMarkdown(html, "First\nSecond", 80)
	if !strings.Contains(result, "First") || !strings.Contains(result, "Second") {
		t.Errorf("expected list items in output, got: %q", result)
	}
}

func TestRenderMarkdown_EmptyHTMLFallback(t *testing.T) {
	result := renderMarkdown("", "Plain text content", 80)
	if !strings.Contains(result, "Plain text content") {
		t.Errorf("expected plain text fallback, got: %q", result)
	}
}

func TestRenderMarkdown_CodeBlock(t *testing.T) {
	html := "<pre><code>func main() {}</code></pre>"
	result := renderMarkdown(html, "func main() {}", 80)
	if !strings.Contains(result, "func main()") {
		t.Errorf("expected code block content, got: %q", result)
	}
}

func TestRenderMarkdown_WidthWrapping(t *testing.T) {
	longText := strings.Repeat("word ", 50)
	result := renderMarkdown("", longText, 40)
	for _, line := range strings.Split(result, "\n") {
		if len(line) > 50 {
			t.Errorf("line too long (%d chars): %q", len(line), line)
		}
	}
}
