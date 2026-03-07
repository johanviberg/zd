package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	articlesCmd.AddCommand(articlesShowCmd)
}

var articlesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a Help Center article",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid article ID: %s", args[0])
		}

		svc, err := newArticleService(cmd)
		if err != nil {
			return err
		}

		result, err := svc.Get(cmd.Context(), id)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())
		return formatter.Format(os.Stdout, result.Article)
	},
}
