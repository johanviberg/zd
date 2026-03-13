package nlq

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fixedNow is 2026-03-07 (Saturday), used for all deterministic date tests.
var fixedNow = time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

func TestTranslateWithTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// --- Passthrough: already Zendesk syntax ---
		{
			name:     "passthrough status:open",
			input:    "status:open",
			expected: "status:open",
		},
		{
			name:     "passthrough multiple fields",
			input:    "priority:high assignee:jane",
			expected: "priority:high assignee:jane",
		},
		{
			name:     "passthrough date comparison",
			input:    "created>2024-01-01",
			expected: "created>2024-01-01",
		},
		{
			name:     "passthrough negated field",
			input:    "-status:closed",
			expected: "-status:closed",
		},
		{
			name:     "passthrough OR expression",
			input:    "status:open OR status:pending",
			expected: "status:open OR status:pending",
		},

		// --- Status keywords ---
		{
			name:     "open status phrase",
			input:    "show all open tickets",
			expected: "status:open",
		},
		{
			name:     "pending status",
			input:    "pending tickets",
			expected: "status:pending",
		},
		{
			name:     "new status",
			input:    "new tickets",
			expected: "status:new",
		},
		{
			name:     "solved status",
			input:    "solved tickets",
			expected: "status:solved",
		},
		{
			name:     "closed status",
			input:    "closed tickets",
			expected: "status:closed",
		},

		// --- Priority phrases ---
		{
			name:     "high priority phrase",
			input:    "high priority tickets",
			expected: "priority:high",
		},
		{
			name:     "low priority phrase",
			input:    "low priority",
			expected: "priority:low",
		},
		{
			name:     "urgent tickets keyword",
			input:    "urgent tickets",
			expected: "priority:urgent",
		},
		{
			name:     "normal priority phrase",
			input:    "normal priority tickets",
			expected: "priority:normal",
		},

		// --- Type keywords ---
		{
			name:     "type problems",
			input:    "problems",
			expected: "type:problem",
		},
		{
			name:     "type incidents",
			input:    "incidents",
			expected: "type:incident",
		},
		{
			name:     "type questions",
			input:    "questions",
			expected: "type:question",
		},
		{
			name:     "type tasks",
			input:    "tasks",
			expected: "type:task",
		},

		// --- Compound queries ---
		{
			name:     "high priority incidents compound",
			input:    "high priority incidents",
			expected: "priority:high type:incident",
		},
		{
			name:  "open tickets from group compound",
			input: "open tickets from billing",
			// "from billing" is a phrase matched before single-keyword matching,
			// so group:billing is appended first, then status:open from keyword pass.
			expected: "group:billing status:open",
		},

		// --- Phrase patterns ---
		{
			name:     "assigned to name",
			input:    "tickets assigned to jane",
			expected: "assignee:jane",
		},
		{
			name:     "requested by name",
			input:    "requested by john",
			expected: "requester:john",
		},
		{
			name:     "tagged vip",
			input:    "tagged vip",
			expected: "tags:vip",
		},
		{
			name:     "on hold",
			input:    "on hold",
			expected: "status:hold",
		},
		{
			name:     "unresolved tickets",
			input:    "unresolved tickets",
			expected: "status<solved",
		},

		// --- Date phrases (fixed now = 2026-03-07 Saturday) ---
		{
			name:     "created today",
			input:    "tickets created today",
			expected: "created>2026-03-07",
		},
		{
			name:     "created yesterday",
			input:    "tickets created yesterday",
			expected: "created>2026-03-06",
		},
		{
			name:  "created this week",
			input: "created this week",
			// Monday of current week: 2026-03-07 - 5 days = 2026-03-02
			expected: "created>2026-03-02",
		},
		{
			name:  "last week",
			input: "last week",
			// start of last week: 2026-03-02 - 7 = 2026-02-23; end: 2026-03-02
			expected: "created>2026-02-23 created<2026-03-02",
		},
		{
			name:     "this month",
			input:    "this month",
			expected: "created>2026-03-01",
		},
		{
			name:  "closed tickets last month",
			input: "closed tickets last month",
			// Date phrases are extracted first (inside extractPhrases), then "closed"
			// is matched as a status keyword in the subsequent token pass.
			expected: "created>2026-02-01 created<2026-03-01 status:closed",
		},
		{
			name:  "last 7 days",
			input: "last 7 days",
			// 2026-03-07 - 7 = 2026-02-28
			expected: "created>2026-02-28",
		},
		{
			name:  "past 30 days",
			input: "past 30 days",
			// 2026-03-07 - 30 = 2026-02-05
			expected: "created>2026-02-05",
		},

		// --- Hour phrases (fixed now = 2026-03-07T12:00:00Z) ---
		{
			name:     "past hour",
			input:    "tickets created in the past hour",
			expected: "created>2026-03-07T11:00:00Z",
		},
		{
			name:     "past 3 hours",
			input:    "past 3 hours",
			expected: "created>2026-03-07T09:00:00Z",
		},
		{
			name:     "last 1 hour",
			input:    "last 1 hour",
			expected: "created>2026-03-07T11:00:00Z",
		},
		{
			name:     "open tickets last 6 hours",
			input:    "open tickets last 6 hours",
			expected: "created>2026-03-07T06:00:00Z status:open",
		},
		{
			name:     "last hour bare",
			input:    "last hour",
			expected: "created>2026-03-07T11:00:00Z",
		},

		// --- Fallback / passthrough for unrecognized input ---
		{
			name:     "unrecognized input passes through",
			input:    "something weird",
			expected: "something weird",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},

		// --- Full-text search alongside structured ---
		{
			name:  "about phrase produces bare text",
			input: "about billing issue",
			// "about X" captures the rest as bare text for full-text search
		},

		// --- Case Sensitivity ---
		{
			name:     "uppercase OPEN TICKETS",
			input:    "OPEN TICKETS",
			expected: "status:open",
		},
		{
			name:     "mixed case High Priority",
			input:    "High Priority",
			expected: "priority:high",
		},
		{
			name:     "mixed case Assigned To Jane",
			input:    "Assigned To Jane",
			expected: "assignee:jane",
		},

		// --- Whitespace Handling ---
		{
			name:     "leading/trailing whitespace",
			input:    "  open tickets  ",
			expected: "status:open",
		},
		{
			name:     "multiple internal spaces",
			input:    "open   tickets",
			expected: "status:open",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "multiple spaces in phrase",
			input:    "assigned  to  jane",
			expected: "assignee:jane",
		},

		// --- Singular Type Keywords ---
		{
			name:     "type problem singular",
			input:    "problem",
			expected: "type:problem",
		},
		{
			name:     "type incident singular",
			input:    "incident",
			expected: "type:incident",
		},
		{
			name:     "type question singular",
			input:    "question",
			expected: "type:question",
		},
		{
			name:     "type task singular",
			input:    "task",
			expected: "type:task",
		},

		// --- "tag" keyword (regex branch) ---
		{
			name:     "tag keyword (not tagged)",
			input:    "tag vip",
			expected: "tags:vip",
		},

		// --- "hold" standalone keyword ---
		{
			name:     "hold standalone keyword",
			input:    "hold tickets",
			expected: "status:hold",
		},

		// --- "past" alternatives ---
		{
			name:     "past month",
			input:    "past month",
			expected: "created>2026-02-01 created<2026-03-01",
		},
		{
			name:     "past week",
			input:    "past week",
			expected: "created>2026-02-23 created<2026-03-02",
		},

		// --- Singular "day" ---
		{
			name:     "past 1 day singular",
			input:    "past 1 day",
			expected: "created>2026-03-06",
		},

		// --- Reverse Priority Order ("priority <level>") ---
		{
			name:     "priority high (reverse order)",
			input:    "priority high",
			expected: "priority:high",
		},
		{
			name:     "priority low (reverse order)",
			input:    "priority low",
			expected: "priority:low",
		},
		{
			name:     "priority urgent (reverse order)",
			input:    "priority urgent",
			expected: "priority:urgent",
		},
		{
			name:     "priority normal (reverse order)",
			input:    "priority normal",
			expected: "priority:normal",
		},

		// --- Token Skipping ---
		{
			name:     "skip 'created' token",
			input:    "created open",
			expected: "status:open",
		},
		{
			name:     "skip 'updated' token",
			input:    "updated open",
			expected: "status:open",
		},
		{
			name:     "skip 'hour' token",
			input:    "hour open",
			expected: "status:open",
		},
		{
			name:     "skip 'hours' token",
			input:    "hours open",
			expected: "status:open",
		},

		// --- All-Noise / Empty Result Fallback ---
		{
			name:     "all noise returns original",
			input:    "show me all the tickets",
			expected: "show me all the tickets",
		},
		{
			name:     "single noise word returns original",
			input:    "tickets",
			expected: "tickets",
		},

		// --- Multiple Phrase Patterns Combined ---
		{
			name:     "open + assigned + tagged",
			input:    "open tickets assigned to jane tagged vip",
			expected: "assignee:jane tags:vip status:open",
		},
		{
			name:     "high priority incidents from group",
			input:    "high priority incidents from billing",
			expected: "priority:high group:billing type:incident",
		},
		{
			name:     "unresolved + assigned + today",
			input:    "unresolved tickets assigned to bob today",
			expected: "status<solved assignee:bob created>2026-03-07",
		},
		{
			name:     "on hold + tagged + requested by",
			input:    "tickets on hold tagged urgent requested by alice",
			expected: "status:hold requester:alice tags:urgent",
		},

		// --- "about" with Structured Clauses ---
		{
			name:     "about with open status",
			input:    "open tickets about billing",
			expected: "billing status:open",
		},
		{
			name:     "about multi-word subject",
			input:    "about network connectivity issue",
			expected: "network connectivity issue",
		},

		// --- Clause Ordering (phrases before keywords) ---
		{
			name:     "phrase before keyword ordering",
			input:    "open tagged vip",
			expected: "tags:vip status:open",
		},
		{
			name:     "date phrase before keyword ordering",
			input:    "open today",
			expected: "created>2026-03-07 status:open",
		},

		// --- "from" at Beginning ---
		{
			name:     "from at start of input",
			input:    "from billing",
			expected: "group:billing",
		},

		// --- Numeric/Special Bare Tokens ---
		{
			name:     "numeric passthrough",
			input:    "12345",
			expected: "12345",
		},
		{
			name:     "ticket number with noise",
			input:    "ticket 42",
			expected: "42",
		},

		// --- Incomplete Phrase Patterns ---
		{
			name:     "about alone (no text after)",
			input:    "about",
			expected: "about",
		},
		{
			name:     "from alone (no group)",
			input:    "from",
			expected: "from",
		},
		{
			name:     "assigned without to",
			input:    "assigned jane",
			expected: "assigned jane",
		},
		{
			name:     "requested without by",
			input:    "requested john",
			expected: "requested john",
		},

		// --- Large N Values ---
		{
			name:     "past 365 days",
			input:    "past 365 days",
			expected: "created>2025-03-07",
		},
		{
			name:     "last 0 hours (zero = now)",
			input:    "last 0 hours",
			expected: "created>2026-03-07T12:00:00Z",
		},

		// --- Passthrough Edge Cases ---
		{
			name:     "space before colon triggers passthrough",
			input:    "status :open",
			expected: "status :open",
		},
		{
			name:     "mixed syntax and NL passthrough",
			input:    "status:open urgent tickets",
			expected: "status:open urgent tickets",
		},
	}

	for _, tc := range tests {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			got := translateWithTime(tc.input, fixedNow)

			// Special case for the "about" test: check containment, not equality.
			if tc.name == "about phrase produces bare text" {
				assert.Contains(t, got, "billing issue", "translateWithTime(%q) = %q, want it to contain %q", tc.input, got, "billing issue")
				return
			}

			assert.Equal(t, tc.expected, got, "translateWithTime(%q)", tc.input)
		})
	}
}

// TestTranslate_PassthroughSyntaxDetection verifies the regex-based passthrough
// works for every known Zendesk field operator combination.
func TestTranslate_PassthroughSyntaxDetection(t *testing.T) {
	passthroughs := []string{
		"status:open",
		"priority:high assignee:jane",
		"created>2024-01-01",
		"-status:closed",
		"status:open OR status:pending",
		"type:incident",
		"group:support",
		"requester:alice",
		"tags:vip",
		"organization:acme",
		"updated<2025-01-01",
		"subject:payment description:error",
	}

	for _, q := range passthroughs {
		t.Run(q, func(t *testing.T) {
			got := translateWithTime(q, fixedNow)
			assert.Equal(t, q, got, "translateWithTime(%q) should be unchanged", q)
		})
	}
}

// TestTranslatePublicFunction verifies the exported Translate function delegates
// correctly (non-deterministic for dates, so we only test non-date queries).
func TestTranslatePublicFunction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"status:open", "status:open"},
		{"show all open tickets", "status:open"},
		{"high priority incidents", "priority:high type:incident"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := Translate(tc.input)
			assert.Equal(t, tc.expected, got, "Translate(%q)", tc.input)
		})
	}
}

// TestTranslateWithTime_DateEdgeCases tests date logic with alternate `now` values.
func TestTranslateWithTime_DateEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		now      time.Time
		expected string
	}{
		// --- Sunday (weekday=0 branch) ---
		{
			name:     "this week on Sunday",
			input:    "this week",
			now:      time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC), // Sunday
			expected: "created>2026-03-02",
		},
		{
			name:     "last week on Sunday",
			input:    "last week",
			now:      time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
			expected: "created>2026-02-23 created<2026-03-02",
		},

		// --- Monday (offset=0) ---
		{
			name:     "this week on Monday",
			input:    "this week",
			now:      time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC), // Monday
			expected: "created>2026-03-02",
		},
		{
			name:     "last week on Monday",
			input:    "last week",
			now:      time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
			expected: "created>2026-02-23 created<2026-03-02",
		},

		// --- January 1 (year boundary) ---
		{
			name:     "last month crosses year boundary",
			input:    "last month",
			now:      time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), // Jan 1
			expected: "created>2025-12-01 created<2026-01-01",
		},
		{
			name:     "yesterday crosses year boundary",
			input:    "yesterday",
			now:      time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "created>2025-12-31",
		},
		{
			name:     "last 7 days crosses year boundary",
			input:    "last 7 days",
			now:      time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: "created>2025-12-25",
		},
		{
			name:     "last week crosses year boundary",
			input:    "last week",
			now:      time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), // Thursday
			expected: "created>2025-12-22 created<2025-12-29",
		},

		// --- Early morning UTC (hour crossing midnight) ---
		{
			name:     "past 3 hours crosses midnight",
			input:    "past 3 hours",
			now:      time.Date(2026, 3, 7, 2, 0, 0, 0, time.UTC),
			expected: "created>2026-03-06T23:00:00Z",
		},
		{
			name:     "past hour stays same date",
			input:    "past hour",
			now:      time.Date(2026, 3, 7, 2, 0, 0, 0, time.UTC),
			expected: "created>2026-03-07T01:00:00Z",
		},

		// --- Leap year ---
		{
			name:     "yesterday is Feb 29 in leap year",
			input:    "yesterday",
			now:      time.Date(2028, 3, 1, 12, 0, 0, 0, time.UTC), // 2028 is leap year
			expected: "created>2028-02-29",
		},
		{
			name:     "last month in leap year March",
			input:    "last month",
			now:      time.Date(2028, 3, 1, 12, 0, 0, 0, time.UTC),
			expected: "created>2028-02-01 created<2028-03-01",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := translateWithTime(tc.input, tc.now)
			assert.Equal(t, tc.expected, got, "translateWithTime(%q, %v)", tc.input, tc.now)
		})
	}
}

// TestFormatExamples verifies the function returns a non-empty help string.
func TestFormatExamples(t *testing.T) {
	out := FormatExamples()
	assert.NotEmpty(t, out, "FormatExamples() returned empty string")
	// It should mention at least one Zendesk field.
	assert.True(t, strings.Contains(out, "status:open"), "FormatExamples() output does not contain expected example %q", "status:open")
}
