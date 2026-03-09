package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func parseCollaborators(values []string) []types.CollaboratorEntry {
	var entries []types.CollaboratorEntry
	for _, v := range values {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			entries = append(entries, types.CollaboratorEntry{UserID: id})
		} else {
			entries = append(entries, types.CollaboratorEntry{Email: v})
		}
	}
	return entries
}

func init() {
	ticketsCmd.AddCommand(ticketsUpdateCmd)

	ticketsUpdateCmd.Flags().String("subject", "", "Ticket subject")
	ticketsUpdateCmd.Flags().String("comment", "", "Comment body")
	ticketsUpdateCmd.Flags().Bool("public", true, "Whether comment is public")
	ticketsUpdateCmd.Flags().String("priority", "", "Priority: urgent, high, normal, low")
	ticketsUpdateCmd.Flags().String("status", "", "Status: new, open, pending, hold, solved, closed")
	ticketsUpdateCmd.Flags().Int64("assignee-id", 0, "Assignee user ID")
	ticketsUpdateCmd.Flags().Int64("group-id", 0, "Group ID")
	ticketsUpdateCmd.Flags().StringSlice("tags", nil, "Replace all tags")
	ticketsUpdateCmd.Flags().StringSlice("add-tags", nil, "Add tags")
	ticketsUpdateCmd.Flags().StringSlice("remove-tags", nil, "Remove tags")
	ticketsUpdateCmd.Flags().StringArray("custom-field", nil, "Custom field (key=value, repeatable)")
	ticketsUpdateCmd.Flags().Bool("safe-update", false, "Use safe update (conflict detection)")
	ticketsUpdateCmd.Flags().StringSlice("cc", nil, "Add CCs (emails or user IDs, comma-separated)")
}

var ticketsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ticket ID: %s", args[0])
		}

		svc, err := newTicketService(cmd)
		if err != nil {
			return err
		}

		req := &types.UpdateTicketRequest{}

		if cmd.Flags().Changed("subject") {
			req.Subject, _ = cmd.Flags().GetString("subject")
		}
		if cmd.Flags().Changed("priority") {
			req.Priority, _ = cmd.Flags().GetString("priority")
		}
		if cmd.Flags().Changed("status") {
			req.Status, _ = cmd.Flags().GetString("status")
		}
		if cmd.Flags().Changed("assignee-id") {
			v, _ := cmd.Flags().GetInt64("assignee-id")
			req.AssigneeID = &v
		}
		if cmd.Flags().Changed("group-id") {
			v, _ := cmd.Flags().GetInt64("group-id")
			req.GroupID = &v
		}
		if cmd.Flags().Changed("tags") {
			req.Tags, _ = cmd.Flags().GetStringSlice("tags")
		}
		if cmd.Flags().Changed("add-tags") {
			req.AddTags, _ = cmd.Flags().GetStringSlice("add-tags")
		}
		if cmd.Flags().Changed("remove-tags") {
			req.RemoveTags, _ = cmd.Flags().GetStringSlice("remove-tags")
		}
		if cmd.Flags().Changed("safe-update") {
			req.SafeUpdate, _ = cmd.Flags().GetBool("safe-update")
		}

		if cmd.Flags().Changed("comment") {
			body, _ := cmd.Flags().GetString("comment")
			public, _ := cmd.Flags().GetBool("public")
			req.Comment = &types.Comment{
				Body:   body,
				Public: &public,
			}
		}

		if cmd.Flags().Changed("cc") {
			ccVals, _ := cmd.Flags().GetStringSlice("cc")
			req.AdditionalCollaborators = parseCollaborators(ccVals)
			if req.Comment != nil && req.Comment.Public != nil && !*req.Comment.Public {
				fmt.Fprintln(os.Stderr, "Warning: --cc has no effect on internal notes")
			}
		}

		if cmd.Flags().Changed("custom-field") {
			strs, _ := cmd.Flags().GetStringArray("custom-field")
			fields, err := parseCustomFields(strs)
			if err != nil {
				return err
			}
			req.CustomFields = fields
		}

		ticket, err := svc.Update(cmd.Context(), id, req)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())
		return formatter.Format(os.Stdout, ticket)
	},
}
