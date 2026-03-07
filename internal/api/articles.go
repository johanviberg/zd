package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/johanviberg/zd/internal/types"
)

type ArticleService struct {
	client *Client
}

func NewArticleService(client *Client) *ArticleService {
	return &ArticleService{client: client}
}

func (s *ArticleService) List(ctx context.Context, opts *types.ListArticlesOptions) (*types.ArticlePage, error) {
	path := "/api/v2/help_center/articles"
	params := url.Values{}

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("page[size]", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("page[after]", opts.Cursor)
		}
		if opts.SortBy != "" {
			params.Set("sort_by", opts.SortBy)
		}
		if opts.SortOrder != "" {
			params.Set("sort_order", opts.SortOrder)
		}
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var page types.ArticlePage
	if err := s.client.doJSON(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (s *ArticleService) Get(ctx context.Context, id int64) (*types.ArticleResult, error) {
	path := fmt.Sprintf("/api/v2/help_center/articles/%d", id)

	var result types.ArticleResult
	if err := s.client.doJSON(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ArticleService) Search(ctx context.Context, query string, opts *types.SearchArticlesOptions) (*types.ArticleSearchPage, error) {
	path := "/api/v2/help_center/articles/search"
	params := url.Values{}
	params.Set("query", query)

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("per_page", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("page[after]", opts.Cursor)
		}
	}

	path += "?" + params.Encode()

	var page types.ArticleSearchPage
	if err := s.client.doJSON(ctx, "GET", path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}
