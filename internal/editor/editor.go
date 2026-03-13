package editor

import (
	"fmt"
	"os"
	"strings"

	charmEditor "github.com/charmbracelet/x/editor"
)

const ticketTemplate = `# Enter ticket details below.
# Lines starting with # are comments and will be ignored.
# The first non-comment line becomes the subject.
# All remaining non-comment lines become the comment body.
# Save and close the editor to create the ticket.
# Leave empty (or all comments) to cancel.

`

// EditTicket opens the user's $EDITOR with a template and returns the
// subject and body parsed from the result. Returns empty strings if
// the user cancels (empty content).
func EditTicket() (subject, body string, err error) {
	content, err := editWithTemplate(ticketTemplate)
	if err != nil {
		return "", "", fmt.Errorf("editor: %w", err)
	}
	return parseTicketContent(content)
}

// EditComment opens the user's $EDITOR for composing a comment body.
func EditComment() (string, error) {
	template := `# Enter your comment below.
# Lines starting with # are comments and will be ignored.
# Save and close the editor to submit.
# Leave empty to cancel.

`
	content, err := editWithTemplate(template)
	if err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}
	return parseCommentContent(content), nil
}

// editWithTemplate writes template to a temp file, opens the user's $EDITOR,
// waits for it to exit, reads back the file, and returns its contents.
func editWithTemplate(template string) (string, error) {
	f, err := os.CreateTemp("", "zd-*.md")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	path := f.Name()
	defer os.Remove(path)

	if _, err := f.WriteString(template); err != nil {
		f.Close()
		return "", fmt.Errorf("writing template: %w", err)
	}
	f.Close()

	app := resolveEditor()
	cmd, err := charmEditor.Cmd(app, path)
	if err != nil {
		return "", fmt.Errorf("building editor command: %w", err)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running editor: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}
	return string(data), nil
}

// resolveEditor returns the editor to use, preferring $VISUAL, then $EDITOR,
// then falling back to "vi".
func resolveEditor() string {
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	if v := os.Getenv("EDITOR"); v != "" {
		return v
	}
	return "vi"
}

// parseTicketContent parses editor output: first non-comment line = subject,
// rest = body. Returns empty strings if all content is comments/empty.
func parseTicketContent(content string) (subject, body string, err error) {
	lines := strings.Split(content, "\n")
	var nonCommentLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		nonCommentLines = append(nonCommentLines, line)
	}

	// Trim leading/trailing empty lines.
	for len(nonCommentLines) > 0 && strings.TrimSpace(nonCommentLines[0]) == "" {
		nonCommentLines = nonCommentLines[1:]
	}
	for len(nonCommentLines) > 0 && strings.TrimSpace(nonCommentLines[len(nonCommentLines)-1]) == "" {
		nonCommentLines = nonCommentLines[:len(nonCommentLines)-1]
	}

	if len(nonCommentLines) == 0 {
		return "", "", nil // cancelled
	}

	subject = strings.TrimSpace(nonCommentLines[0])
	if len(nonCommentLines) > 1 {
		body = strings.TrimSpace(strings.Join(nonCommentLines[1:], "\n"))
	}
	return subject, body, nil
}

// parseCommentContent strips comment lines and returns the trimmed body.
func parseCommentContent(content string) string {
	lines := strings.Split(content, "\n")
	var nonCommentLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		nonCommentLines = append(nonCommentLines, line)
	}
	return strings.TrimSpace(strings.Join(nonCommentLines, "\n"))
}
