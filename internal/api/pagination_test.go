package api

import (
	"testing"

	"github.com/johanviberg/zd/internal/types"
)

func TestHasNextPage(t *testing.T) {
	tests := []struct {
		name     string
		meta     types.PageMeta
		expected bool
	}{
		{"has more", types.PageMeta{HasMore: true, AfterCursor: "abc"}, true},
		{"no more", types.PageMeta{HasMore: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasNextPage(tt.meta); got != tt.expected {
				t.Errorf("HasNextPage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNextCursor(t *testing.T) {
	meta := types.PageMeta{AfterCursor: "cursor123"}
	if got := NextCursor(meta); got != "cursor123" {
		t.Errorf("NextCursor() = %q, want %q", got, "cursor123")
	}
}
