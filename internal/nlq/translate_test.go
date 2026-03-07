package nlq

import (
	"strings"
	"testing"
	"time"
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
	}

	for _, tc := range tests {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			got := translateWithTime(tc.input, fixedNow)

			// Special case for the "about" test: check containment, not equality.
			if tc.name == "about phrase produces bare text" {
				if !strings.Contains(got, "billing issue") {
					t.Errorf("translateWithTime(%q) = %q, want it to contain %q", tc.input, got, "billing issue")
				}
				return
			}

			if got != tc.expected {
				t.Errorf("translateWithTime(%q)\n  got:  %q\n  want: %q", tc.input, got, tc.expected)
			}
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
			if got != q {
				t.Errorf("translateWithTime(%q) = %q, want unchanged", q, got)
			}
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
			if got != tc.expected {
				t.Errorf("Translate(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestFormatExamples verifies the function returns a non-empty help string.
func TestFormatExamples(t *testing.T) {
	out := FormatExamples()
	if out == "" {
		t.Error("FormatExamples() returned empty string")
	}
	// It should mention at least one Zendesk field.
	if !strings.Contains(out, "status:open") {
		t.Errorf("FormatExamples() output does not contain expected example %q", "status:open")
	}
}
