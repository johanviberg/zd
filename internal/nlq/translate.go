package nlq

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// knownFields are Zendesk search fields used to detect existing query syntax.
var knownFields = []string{
	"status", "priority", "type", "assignee", "group", "requester",
	"subject", "description", "tags", "organization", "created", "updated",
}

var syntaxPattern *regexp.Regexp

func init() {
	// Match patterns like field:value, field>value, field<value, -field:value
	escaped := make([]string, len(knownFields))
	for i, f := range knownFields {
		escaped[i] = regexp.QuoteMeta(f)
	}
	syntaxPattern = regexp.MustCompile(`(?:^|\s)-?(?:` + strings.Join(escaped, "|") + `)\s*[:<>]`)
}

// noiseWords are stripped from natural language input before processing.
var noiseWords = map[string]bool{
	"show": true, "all": true, "me": true, "my": true, "the": true,
	"find": true, "get": true, "list": true, "search": true, "for": true, "in": true,
	"with": true, "tickets": true, "ticket": true, "that": true,
	"are": true, "is": true, "a": true, "an": true,
}

// statusKeywords maps standalone words to status values.
var statusKeywords = map[string]string{
	"open": "open", "pending": "pending", "new": "new",
	"solved": "solved", "closed": "closed", "hold": "hold",
}

// typeKeywords maps words to Zendesk ticket types (singular form).
var typeKeywords = map[string]string{
	"problem": "problem", "problems": "problem",
	"incident": "incident", "incidents": "incident",
	"question": "question", "questions": "question",
	"task": "task", "tasks": "task",
}

// Translate converts a natural language query into Zendesk search syntax.
// If the query already uses Zendesk syntax, it is returned unchanged.
func Translate(query string) string {
	return translateWithTime(query, time.Now())
}

func translateWithTime(query string, now time.Time) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return trimmed
	}

	// If it already looks like Zendesk syntax, return unchanged.
	if syntaxPattern.MatchString(trimmed) {
		return trimmed
	}

	input := strings.ToLower(trimmed)

	var clauses []string

	// Extract phrase patterns (longest first) before single-keyword matching.
	input, clauses = extractPhrases(input, clauses, now)

	// Split remaining into tokens and strip noise words.
	tokens := strings.Fields(input)
	var remaining []string
	for _, tok := range tokens {
		if noiseWords[tok] {
			continue
		}
		remaining = append(remaining, tok)
	}

	// Match single keywords.
	var unmatched []string
	for _, tok := range remaining {
		if v, ok := statusKeywords[tok]; ok {
			clauses = append(clauses, "status:"+v)
		} else if tok == "urgent" {
			clauses = append(clauses, "priority:urgent")
		} else if v, ok := typeKeywords[tok]; ok {
			clauses = append(clauses, "type:"+v)
		} else if tok == "created" || tok == "updated" || tok == "hour" || tok == "hours" {
			// Skip date-related words already consumed or standalone.
			continue
		} else {
			unmatched = append(unmatched, tok)
		}
	}

	// Append unmatched tokens as bare text for full-text search.
	clauses = append(clauses, unmatched...)

	result := strings.Join(clauses, " ")
	result = strings.TrimSpace(result)
	if result == "" {
		return trimmed
	}
	return result
}

// extractPhrases matches multi-word patterns and returns the modified input and accumulated clauses.
func extractPhrases(input string, clauses []string, now time.Time) (string, []string) {
	// "on hold" → status:hold (before single-word matching eats "hold")
	if strings.Contains(input, "on hold") {
		clauses = append(clauses, "status:hold")
		input = strings.Replace(input, "on hold", " ", 1)
	}

	// "unresolved" → status<solved
	if strings.Contains(input, "unresolved") {
		clauses = append(clauses, "status<solved")
		input = strings.Replace(input, "unresolved", " ", 1)
	}

	// Priority phrases: "high priority", "low priority", "normal priority"
	for _, level := range []string{"high", "low", "normal", "urgent"} {
		phrase := level + " priority"
		if strings.Contains(input, phrase) {
			clauses = append(clauses, "priority:"+level)
			input = strings.Replace(input, phrase, " ", 1)
		}
		phrase2 := "priority " + level
		if strings.Contains(input, phrase2) {
			clauses = append(clauses, "priority:"+level)
			input = strings.Replace(input, phrase2, " ", 1)
		}
	}

	// "assigned to <name>"
	re := regexp.MustCompile(`assigned\s+to\s+(\S+)`)
	if m := re.FindStringSubmatch(input); m != nil {
		clauses = append(clauses, "assignee:"+m[1])
		input = re.ReplaceAllString(input, " ")
	}

	// "requested by <name>"
	re = regexp.MustCompile(`requested\s+by\s+(\S+)`)
	if m := re.FindStringSubmatch(input); m != nil {
		clauses = append(clauses, "requester:"+m[1])
		input = re.ReplaceAllString(input, " ")
	}

	// "from <group>" — only match if not at the very start (contextual)
	re = regexp.MustCompile(`\bfrom\s+(\S+)`)
	if m := re.FindStringSubmatch(input); m != nil {
		clauses = append(clauses, "group:"+m[1])
		input = re.ReplaceAllString(input, " ")
	}

	// "tagged <tag>" / "tag <tag>"
	re = regexp.MustCompile(`\b(?:tagged|tag)\s+(\S+)`)
	if m := re.FindStringSubmatch(input); m != nil {
		clauses = append(clauses, "tags:"+m[1])
		input = re.ReplaceAllString(input, " ")
	}

	// "about <subject>" — remainder becomes bare text
	re = regexp.MustCompile(`\babout\s+(.+)`)
	if m := re.FindStringSubmatch(input); m != nil {
		clauses = append(clauses, strings.TrimSpace(m[1]))
		input = re.ReplaceAllString(input, " ")
	}

	// Date phrases
	input, clauses = extractDatePhrases(input, clauses, now)

	return input, clauses
}

// extractDatePhrases handles temporal expressions.
func extractDatePhrases(input string, clauses []string, now time.Time) (string, []string) {
	// "last month" / "past month"
	if strings.Contains(input, "last month") || strings.Contains(input, "past month") {
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		firstOfLastMonth := firstOfThisMonth.AddDate(0, -1, 0)
		clauses = append(clauses, "created>"+firstOfLastMonth.Format("2006-01-02"))
		clauses = append(clauses, "created<"+firstOfThisMonth.Format("2006-01-02"))
		input = strings.Replace(input, "last month", " ", 1)
		input = strings.Replace(input, "past month", " ", 1)
	}

	// "this month"
	if strings.Contains(input, "this month") {
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		clauses = append(clauses, "created>"+firstOfMonth.Format("2006-01-02"))
		input = strings.Replace(input, "this month", " ", 1)
	}

	// "last week" / "past week"
	if strings.Contains(input, "last week") || strings.Contains(input, "past week") {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startOfThisWeek := now.AddDate(0, 0, -(weekday - 1))
		startOfLastWeek := startOfThisWeek.AddDate(0, 0, -7)
		clauses = append(clauses, "created>"+startOfLastWeek.Format("2006-01-02"))
		clauses = append(clauses, "created<"+startOfThisWeek.Format("2006-01-02"))
		input = strings.Replace(input, "last week", " ", 1)
		input = strings.Replace(input, "past week", " ", 1)
	}

	// "this week"
	if strings.Contains(input, "this week") {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startOfWeek := now.AddDate(0, 0, -(weekday - 1))
		clauses = append(clauses, "created>"+startOfWeek.Format("2006-01-02"))
		input = strings.Replace(input, "this week", " ", 1)
	}

	// "last/past N hour(s)" — e.g. "past 3 hours"
	re := regexp.MustCompile(`(?:last|past)\s+(\d+)\s+hours?`)
	if m := re.FindStringSubmatch(input); m != nil {
		n, _ := strconv.Atoi(m[1])
		t := now.Add(-time.Duration(n) * time.Hour)
		clauses = append(clauses, "created>"+t.UTC().Format(time.RFC3339))
		input = re.ReplaceAllString(input, " ")
	}

	// "last/past hour" (no number = 1 hour)
	if strings.Contains(input, "last hour") || strings.Contains(input, "past hour") {
		t := now.Add(-time.Hour)
		clauses = append(clauses, "created>"+t.UTC().Format(time.RFC3339))
		input = strings.Replace(input, "last hour", " ", 1)
		input = strings.Replace(input, "past hour", " ", 1)
	}

	// "last N days" / "past N days"
	re = regexp.MustCompile(`(?:last|past)\s+(\d+)\s+days?`)
	if m := re.FindStringSubmatch(input); m != nil {
		n, _ := strconv.Atoi(m[1])
		date := now.AddDate(0, 0, -n)
		clauses = append(clauses, "created>"+date.Format("2006-01-02"))
		input = re.ReplaceAllString(input, " ")
	}

	// "yesterday"
	if strings.Contains(input, "yesterday") {
		yesterday := now.AddDate(0, 0, -1)
		clauses = append(clauses, "created>"+yesterday.Format("2006-01-02"))
		input = strings.Replace(input, "yesterday", " ", 1)
	}

	// "today"
	if strings.Contains(input, "today") {
		clauses = append(clauses, "created>"+now.Format("2006-01-02"))
		input = strings.Replace(input, "today", " ", 1)
	}

	return input, clauses
}

// FormatExamples returns example translations for help text.
func FormatExamples() string {
	return fmt.Sprintf(`Natural language examples:
  "show all open tickets"          → status:open
  "urgent tickets"                 → priority:urgent
  "tickets assigned to jane"       → assignee:jane
  "open tickets from billing"      → status:open group:billing
  "unresolved tickets"             → status<solved
  "tickets created this week"      → created>YYYY-MM-DD
  "high priority incidents"        → priority:high type:incident

Zendesk syntax is also accepted directly:
  "status:open priority:high"      → passed through unchanged`)
}
