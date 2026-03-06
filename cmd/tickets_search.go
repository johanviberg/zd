package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	ticketsCmd.AddCommand(ticketsSearchCmd)

	ticketsSearchCmd.Flags().Int("limit", 100, "Maximum number of results")
	ticketsSearchCmd.Flags().String("cursor", "", "Pagination cursor")
	ticketsSearchCmd.Flags().String("sort-by", "", "Sort field")
	ticketsSearchCmd.Flags().String("sort-order", "desc", "Sort order: asc or desc")
	ticketsSearchCmd.Flags().Bool("export", false, "Use export endpoint for >1000 results")
	ticketsSearchCmd.Flags().String("include", "", "Sideload: users, groups, organizations")
}

var ticketsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tickets",
	Long: `Search tickets using Zendesk search syntax.

Query parameters:
  status:       new, open, pending, hold, solved, closed
  priority:     urgent, high, normal, low
  type:         problem, incident, question, task
  assignee:     agent name or email
  group:        group name
  requester:    requester name or email
  subject:      text in subject line
  description:  text in first comment
  tags:         tag name
  created:      date (e.g., >2024-01-01, <2024-06-01)
  updated:      date range
  organization: organization name

Combine with spaces (AND) or "OR":
  zd tickets search "status:open priority:high"
  zd tickets search "status:open OR status:pending"
  zd tickets search "tags:vip assignee:jane"
  zd tickets search "created>2024-01-01 status:open"

Full reference: https://support.zendesk.com/hc/en-us/articles/4408886879258`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		svc, err := newSearchService(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")
		sortBy, _ := cmd.Flags().GetString("sort-by")
		sortOrder, _ := cmd.Flags().GetString("sort-order")
		export, _ := cmd.Flags().GetBool("export")
		include, _ := cmd.Flags().GetString("include")

		opts := &types.SearchOptions{
			Limit:     limit,
			Cursor:    cursor,
			SortBy:    sortBy,
			SortOrder: sortOrder,
			Export:    export,
			Include:   include,
		}

		page, err := svc.Search(cmd.Context(), query, opts)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())

		userMap := buildUserMap(page.Users)
		items := make([]interface{}, len(page.Results))
		for i, r := range page.Results {
			items[i] = enrichTicket(r.Ticket, userMap)
		}

		headers := []string{"id", "status", "priority", "subject", "updated_at"}
		if len(page.Users) > 0 {
			headers = []string{"id", "status", "priority", "requester_name", "assignee_name", "subject", "updated_at"}
		}
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
