package api

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/johanviberg/zd/internal/types"
)

type SearchService struct {
	client *Client
}

func NewSearchService(client *Client) *SearchService {
	return &SearchService{client: client}
}

func (s *SearchService) Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchPage, error) {
	var path string
	params := url.Values{}
	params.Set("query", query+" type:ticket")

	if opts != nil && opts.Export {
		path = "/api/v2/search/export"
		params.Set("filter[type]", "ticket")
		if opts.Limit > 0 {
			params.Set("page[size]", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("page[after]", opts.Cursor)
		}
	} else {
		path = "/api/v2/search"
		if opts != nil {
			if opts.Limit > 0 {
				params.Set("per_page", strconv.Itoa(opts.Limit))
			}
			if opts.SortBy != "" {
				params.Set("sort_by", opts.SortBy)
			}
			if opts.SortOrder != "" {
				params.Set("sort_order", opts.SortOrder)
			}
		}
	}

	if opts != nil && opts.Include != "" {
		params.Set("include", opts.Include)
	}

	// Zendesk expects literal brackets in filter[type] and page[size]/page[after].
	// url.Values.Encode() percent-encodes them, so restore the brackets.
	encoded := params.Encode()
	encoded = strings.ReplaceAll(encoded, "%5B", "[")
	encoded = strings.ReplaceAll(encoded, "%5D", "]")
	path += "?" + encoded

	var page types.SearchPage
	if err := s.client.doJSON(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
