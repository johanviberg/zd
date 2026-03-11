package tui

import (
	"fmt"
	"strings"

	"github.com/johanviberg/zd/internal/types"
)

// TimelineNode represents one visual node (one audit, possibly with grouped events).
type TimelineNode struct {
	Audit    types.Audit
	Comments []types.AuditEvent
	Changes  []types.AuditEvent
}

// relevantFieldNames is the set of field changes to show in the timeline.
var relevantFieldNames = map[string]bool{
	"status":      true,
	"priority":    true,
	"assignee_id": true,
	"group_id":    true,
	"subject":     true,
	"tags":        true,
}

// buildTimeline converts audits into renderable nodes, filtering irrelevant events.
func buildTimeline(audits []types.Audit) []TimelineNode {
	var nodes []TimelineNode
	for _, audit := range audits {
		var comments []types.AuditEvent
		var changes []types.AuditEvent

		for _, ev := range audit.Events {
			switch ev.Type {
			case "Comment":
				if ev.Body != "" {
					comments = append(comments, ev)
				}
			case "Change", "Create":
				if relevantFieldNames[ev.FieldName] {
					changes = append(changes, ev)
				}
			}
		}

		if len(comments) > 0 || len(changes) > 0 {
			nodes = append(nodes, TimelineNode{
				Audit:    audit,
				Comments: comments,
				Changes:  changes,
			})
		}
	}
	return nodes
}

// filterCommentNodes returns only nodes that contain comment events.
func filterCommentNodes(nodes []TimelineNode) []TimelineNode {
	var filtered []TimelineNode
	for _, n := range nodes {
		if len(n.Comments) > 0 {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// renderTimeline renders the vertical timeline string for the viewport.
func renderTimeline(nodes []TimelineNode, users map[int64]types.User, width int) string {
	if len(nodes) == 0 {
		return ""
	}

	bodyWidth := width - 5 // gutter: " │  " = 4 chars + 1 padding
	if bodyWidth < 20 {
		bodyWidth = 20
	}

	var b strings.Builder
	connector := timelineConnectorStyle.Render

	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// Determine node icon and author line
		timeStr := relativeTime(node.Audit.CreatedAt)
		author := timelineUserName(node.Audit.AuthorID, users)

		icon := nodeIcon(node)
		branch := "├─"
		if isLast {
			branch = "╰─"
		}

		// Header: branch + icon + time + author
		b.WriteString(connector(" "+branch+" ") + icon + " " +
			commentTimeStyle.Render(timeStr) +
			" " + connector("·") + " " +
			commentAuthorStyle.Render(author) + "\n")

		// Render change events
		for _, ch := range node.Changes {
			line := renderFieldChange(ch, users)
			b.WriteString(connector(" │  ") + line + "\n")
		}

		// Render comment events
		for _, c := range node.Comments {
			isPublic := c.Public == nil || *c.Public

			if !isPublic {
				b.WriteString(connector(" │  ") + internalNoteStyle.Render("(internal)") + "\n")
			}

			// Word-wrap body
			lines := wrapText(c.Body, bodyWidth)
			for _, line := range lines {
				b.WriteString(connector(" │  ") + line + "\n")
			}

			// Attachments
			for _, a := range c.Attachments {
				icon := "📎"
				style := attachmentStyle
				if a.IsImage() {
					icon = "📷"
					style = attachmentImageStyle
				}
				b.WriteString(connector(" │  ") + "  " +
					style.Render(fmt.Sprintf("%s %s (%s)", icon, a.FileName, a.HumanSize())) + "\n")
			}
		}

		// Blank line between nodes (connector continues)
		if !isLast {
			b.WriteString(connector(" │") + "\n")
		}
	}

	return b.String()
}

// nodeIcon returns the icon for a timeline node based on its content.
func nodeIcon(node TimelineNode) string {
	// If there are status changes, use the target status icon
	for _, ch := range node.Changes {
		if ch.FieldName == "status" {
			if val, ok := ch.Value.(string); ok {
				return styledStatus(val)[:len(statusIcons[val])+len(val)+1] // just get the icon
			}
		}
	}

	// For changes-only nodes (no comments), use a bullet
	if len(node.Comments) == 0 {
		return timelineChangeStyle.Render("●")
	}

	// For comment nodes
	return commentAuthorStyle.Render("●")
}

// renderFieldChange renders a single field change line.
func renderFieldChange(ev types.AuditEvent, users map[int64]types.User) string {
	arrow := timelineArrowStyle.Render(" → ")
	label := fieldLabel(ev.FieldName)
	prev := formatFieldValue(ev.FieldName, ev.PreviousValue, users)
	next := formatFieldValue(ev.FieldName, ev.Value, users)

	return timelineChangeStyle.Render(label+": ") + prev + arrow + next
}

// fieldLabel returns a human-readable label for a field name.
func fieldLabel(name string) string {
	switch name {
	case "status":
		return "Status"
	case "priority":
		return "Priority"
	case "assignee_id":
		return "Assignee"
	case "group_id":
		return "Group"
	case "subject":
		return "Subject"
	case "tags":
		return "Tags"
	default:
		return name
	}
}

// formatFieldValue formats a field value with appropriate styling.
func formatFieldValue(field string, val interface{}, users map[int64]types.User) string {
	s := fmt.Sprintf("%v", val)
	if s == "" || s == "<nil>" {
		return dimStyle.Render("none")
	}

	switch field {
	case "status":
		return styledStatus(s)
	case "priority":
		return styledPriority(s)
	case "assignee_id":
		return resolveUserValue(s, users)
	default:
		return timelineChangeStyle.Render(s)
	}
}

// resolveUserValue tries to resolve a user ID string to a name.
func resolveUserValue(val string, users map[int64]types.User) string {
	// Try parsing as int64
	var id int64
	if _, err := fmt.Sscanf(val, "%d", &id); err == nil && id > 0 {
		if u, ok := users[id]; ok {
			return commentAuthorStyle.Render(u.Name)
		}
		return dimStyle.Render(fmt.Sprintf("User #%d", id))
	}
	if val == "" || val == "0" {
		return dimStyle.Render("unassigned")
	}
	return timelineChangeStyle.Render(val)
}

// timelineUserName returns a user's name for the timeline header.
func timelineUserName(id int64, users map[int64]types.User) string {
	if id == 0 {
		return "System"
	}
	if u, ok := users[id]; ok {
		return u.Name
	}
	return fmt.Sprintf("User #%d", id)
}

// wrapText wraps text to the given width, preserving existing line breaks.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var result []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result = append(result, "")
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}

		line := words[0]
		if len(line) > width {
			for len(line) > width {
				result = append(result, line[:width])
				line = line[width:]
			}
		}
		for _, w := range words[1:] {
			if len(line)+1+len(w) > width {
				result = append(result, line)
				line = w
				// Break long words that exceed width
				for len(line) > width {
					result = append(result, line[:width])
					line = line[width:]
				}
			} else {
				line += " " + w
			}
		}
		result = append(result, line)
	}
	return result
}
