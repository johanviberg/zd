package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	ticketsCmd.AddCommand(ticketsListCmd)

	ticketsListCmd.Flags().Int("limit", 100, "Maximum number of tickets to return")
	ticketsListCmd.Flags().String("cursor", "", "Pagination cursor")
	ticketsListCmd.Flags().String("sort", "updated_at", "Sort field")
	ticketsListCmd.Flags().String("sort-order", "desc", "Sort order: asc or desc")
	ticketsListCmd.Flags().String("status", "", "Filter by status")
	ticketsListCmd.Flags().Int64("assignee", 0, "Filter by assignee ID")
	ticketsListCmd.Flags().Int64("group", 0, "Filter by group ID")
}

var ticketsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newTicketService(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")
		sort, _ := cmd.Flags().GetString("sort")
		sortOrder, _ := cmd.Flags().GetString("sort-order")
		status, _ := cmd.Flags().GetString("status")
		assignee, _ := cmd.Flags().GetInt64("assignee")
		group, _ := cmd.Flags().GetInt64("group")

		opts := &types.ListTicketsOptions{
			Limit:     limit,
			Cursor:    cursor,
			Sort:      sort,
			SortOrder: sortOrder,
			Status:    status,
			Assignee:  assignee,
			Group:     group,
		}

		page, err := svc.List(cmd.Context(), opts)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())

		items := make([]interface{}, len(page.Tickets))
		for i, t := range page.Tickets {
			items[i] = t
		}

		headers := []string{"id", "status", "priority", "subject", "updated_at"}
		if err := formatter.FormatList(os.Stdout, items, headers); err != nil {
			return err
		}

		if page.Meta.HasMore {
			fmt.Fprintf(os.Stderr, "\nMore results available. Use --cursor %q to fetch next page.\n", page.Meta.AfterCursor)
		}

		return nil
	},
}
