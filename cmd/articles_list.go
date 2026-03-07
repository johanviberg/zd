package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	articlesCmd.AddCommand(articlesListCmd)

	articlesListCmd.Flags().Int("limit", 25, "Maximum number of articles to return")
	articlesListCmd.Flags().String("cursor", "", "Pagination cursor")
	articlesListCmd.Flags().String("sort-by", "", "Sort field: title, created_at, updated_at")
	articlesListCmd.Flags().String("sort-order", "desc", "Sort order: asc or desc")
}

var articlesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Help Center articles",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newArticleService(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		sortOrder, _ := cmd.Flags().GetString("sort-order")

		opts := &types.ListArticlesOptions{
			Limit:     limit,
			Cursor:    cursor,
			SortBy:    sortBy,
			SortOrder: sortOrder,
		}

		page, err := svc.List(cmd.Context(), opts)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())

		items := make([]interface{}, len(page.Articles))
		for i, a := range page.Articles {
			items[i] = a
		}

		headers := []string{"id", "title", "draft", "promoted", "updated_at"}
		if err := formatter.FormatList(os.Stdout, items, headers); err != nil {
			return err
		}

		if page.Meta.HasMore {
			fmt.Fprintf(os.Stderr, "\nMore results available. Use --cursor %q to fetch next page.\n", page.Meta.AfterCursor)
		}

		return nil
	},
}
