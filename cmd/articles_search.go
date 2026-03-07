package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	articlesCmd.AddCommand(articlesSearchCmd)

	articlesSearchCmd.Flags().Int("limit", 25, "Maximum number of results")
	articlesSearchCmd.Flags().String("cursor", "", "Pagination cursor")
}

var articlesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Help Center articles",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		svc, err := newArticleService(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")

		opts := &types.SearchArticlesOptions{
			Limit:  limit,
			Cursor: cursor,
		}

		page, err := svc.Search(cmd.Context(), query, opts)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())

		items := make([]interface{}, len(page.Results))
		for i, a := range page.Results {
			items[i] = a
		}

		headers := []string{"id", "title", "draft", "promoted", "updated_at"}
		if err := formatter.FormatList(os.Stdout, items, headers); err != nil {
			return err
		}

		if page.Meta.HasMore {
			fmt.Fprintf(os.Stderr, "\nMore results available. Use --cursor %q to fetch next page.\n", page.Meta.AfterCursor)
		}

		fmt.Fprintf(os.Stderr, "\n%d results found\n", page.Count)

		return nil
	},
}
