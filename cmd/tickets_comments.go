package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	ticketsCmd.AddCommand(ticketsCommentsCmd)

	ticketsCommentsCmd.Flags().Int("limit", 100, "Maximum number of comments to return")
	ticketsCommentsCmd.Flags().String("cursor", "", "Pagination cursor")
	ticketsCommentsCmd.Flags().String("sort-order", "asc", "Sort order: asc or desc")
	ticketsCommentsCmd.Flags().String("include", "", "Sideload: users")
}

var ticketsCommentsCmd = &cobra.Command{
	Use:   "comments <ticket_id>",
	Short: "List comments on a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ticket ID: %s", args[0])
		}

		svc, err := newTicketService(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		cursor, _ := cmd.Flags().GetString("cursor")
		sortOrder, _ := cmd.Flags().GetString("sort-order")
		include, _ := cmd.Flags().GetString("include")

		opts := &types.ListCommentsOptions{
			Limit:     limit,
			Cursor:    cursor,
			SortOrder: sortOrder,
			Include:   include,
		}

		page, err := svc.ListComments(cmd.Context(), ticketID, opts)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())

		userMap := buildUserMap(page.Users)
		items := make([]interface{}, len(page.Comments))
		for i, c := range page.Comments {
			items[i] = enrichComment(c, userMap)
		}

		headers := []string{"id", "author_id", "public", "body", "created_at"}
		if len(page.Users) > 0 {
			headers = []string{"id", "author_name", "public", "body", "created_at"}
		}
		if err := formatter.FormatList(os.Stdout, items, headers); err != nil {
			return err
		}

		if page.Meta.HasMore {
			fmt.Fprintf(os.Stderr, "\nMore results available. Use --cursor %q to fetch next page.\n", page.Meta.AfterCursor)
		}

		return nil
	},
}

func enrichComment(comment interface{}, userMap map[int64]types.User) interface{} {
	if len(userMap) == 0 {
		return comment
	}

	b, err := json.Marshal(comment)
	if err != nil {
		return comment
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return comment
	}

	if aid, ok := m["author_id"].(float64); ok {
		if u, found := userMap[int64(aid)]; found {
			m["author_name"] = u.Name
			m["author_email"] = u.Email
		}
	}

	return m
}
