package types

import "time"

type Article struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Body       string    `json:"body,omitempty"`
	AuthorID   int64     `json:"author_id,omitempty"`
	SectionID  int64     `json:"section_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Promoted   bool      `json:"promoted"`
	Draft      bool      `json:"draft"`
	HTMLURL    string    `json:"html_url,omitempty"`
	LabelNames []string  `json:"label_names,omitempty"`
	Locale     string    `json:"locale,omitempty"`
}

type ArticlePage struct {
	Articles []Article `json:"articles"`
	Meta     PageMeta  `json:"meta"`
	Links    PageLinks `json:"links"`
	Count    int       `json:"count,omitempty"`
}

type ArticleResult struct {
	Article Article `json:"article"`
}

type ArticleSearchPage struct {
	Results []Article `json:"results"`
	Meta    PageMeta  `json:"meta"`
	Links   PageLinks `json:"links"`
	Count   int       `json:"count,omitempty"`
}

type ListArticlesOptions struct {
	Limit     int
	Cursor    string
	SortBy    string
	SortOrder string
}

type SearchArticlesOptions struct {
	Limit  int
	Cursor string
}
