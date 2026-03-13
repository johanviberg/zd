package api

import (
	"testing"

	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.expected, HasNextPage(tt.meta))
		})
	}
}

func TestNextCursor(t *testing.T) {
	meta := types.PageMeta{AfterCursor: "cursor123"}
	assert.Equal(t, "cursor123", NextCursor(meta))
}
