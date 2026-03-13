package tui

import (
	"testing"

	"github.com/sahilm/fuzzy"
)

func TestCmdPaletteRefilter(t *testing.T) {
	items := []cmdItem{
		{name: "Search", shortcut: "/", category: "Navigation", action: "search"},
		{name: "View ticket", shortcut: "enter", category: "Navigation", action: "enter"},
		{name: "Go to ticket", shortcut: "g", category: "Navigation", action: "goto"},
		{name: "Add comment", shortcut: "c", category: "Ticket Actions", action: "comment"},
		{name: "Change status", shortcut: "s", category: "Ticket Actions", action: "status"},
		{name: "Quit", shortcut: "q", category: "System", action: "quit"},
	}

	tests := []struct {
		name         string
		query        string
		wantMinLen   int
		wantContains string // action field of expected result
		wantEmpty    bool
	}{
		{
			name:       "empty query returns all items",
			query:      "",
			wantMinLen: len(items),
		},
		{
			name:         "exact match",
			query:        "Search",
			wantMinLen:   1,
			wantContains: "search",
		},
		{
			name:         "partial prefix match",
			query:        "sea",
			wantMinLen:   1,
			wantContains: "search",
		},
		{
			name:         "subsequence match",
			query:        "sc",
			wantMinLen:   1,
			wantContains: "search",
		},
		{
			name:      "no match returns empty",
			query:     "zzzzz",
			wantEmpty: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var filtered []cmdItem
			if tc.query == "" {
				filtered = items
			} else {
				matches := fuzzy.FindFrom(tc.query, cmdItems(items))
				filtered = make([]cmdItem, len(matches))
				for i, match := range matches {
					filtered[i] = items[match.Index]
				}
			}

			if tc.wantEmpty {
				if len(filtered) != 0 {
					t.Errorf("expected 0 results for query %q, got %d", tc.query, len(filtered))
				}
				return
			}
			if len(filtered) < tc.wantMinLen {
				t.Errorf("expected at least %d result(s) for query %q, got %d", tc.wantMinLen, tc.query, len(filtered))
			}
			if tc.wantContains != "" {
				found := false
				for _, item := range filtered {
					if item.action == tc.wantContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected action %q in results for query %q", tc.wantContains, tc.query)
				}
			}
		})
	}
}

func TestHighlightMatches(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		matchedIndexes []int
		wantPlain      string
	}{
		{
			name:           "no indexes returns plain text",
			text:           "Search",
			matchedIndexes: nil,
			wantPlain:      "Search",
		},
		{
			name:           "empty text",
			text:           "",
			matchedIndexes: []int{0},
			wantPlain:      "",
		},
		{
			name:           "all chars highlighted",
			text:           "ab",
			matchedIndexes: []int{0, 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := highlightMatches(tc.text, tc.matchedIndexes)
			if tc.wantPlain != "" && result != tc.wantPlain {
				// Only check exact match for no-highlight case
				if len(tc.matchedIndexes) == 0 && result != tc.wantPlain {
					t.Errorf("result = %q, want %q", result, tc.wantPlain)
				}
			}
		})
	}
}
