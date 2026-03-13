package tui

import (
	"strings"

	"charm.land/glamour/v2"
	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// renderMarkdown converts HTML content to styled terminal output.
// Falls back to plain text wrapping if HTML is empty or conversion fails.
func renderMarkdown(htmlBody, plainBody string, width int) string {
	if htmlBody != "" {
		md, err := htmlToMarkdown(htmlBody)
		if err == nil && strings.TrimSpace(md) != "" {
			rendered, err := renderGlamour(md, width)
			if err == nil {
				return strings.TrimSpace(rendered)
			}
		}
	}
	// Fallback to plain text
	return strings.Join(wrapText(plainBody, width), "\n")
}

// htmlToMarkdown converts HTML to markdown.
func htmlToMarkdown(html string) (string, error) {
	return htmltomd.ConvertString(html)
}

// renderGlamour renders markdown through glamour for terminal output.
func renderGlamour(md string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(md)
}
