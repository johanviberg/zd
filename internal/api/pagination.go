package api

import (
	"github.com/johanviberg/zd/internal/types"
)

// HasNextPage checks if there are more pages to fetch.
func HasNextPage(meta types.PageMeta) bool {
	return meta.HasMore
}

// NextCursor returns the cursor for the next page.
func NextCursor(meta types.PageMeta) string {
	return meta.AfterCursor
}
