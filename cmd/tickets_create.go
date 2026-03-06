package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	ticketsCmd.AddCommand(ticketsCreateCmd)

	ticketsCreateCmd.Flags().String("subject", "", "Ticket subject (required)")
	ticketsCreateCmd.Flags().String("comment", "", "Ticket comment body (required)")
	ticketsCreateCmd.Flags().String("priority", "", "Priority: urgent, high, normal, low")
	ticketsCreateCmd.Flags().String("type", "", "Type: problem, incident, question, task")
	ticketsCreateCmd.Flags().String("status", "", "Status: new, open, pending, hold, solved, closed")
	ticketsCreateCmd.Flags().Int64("assignee-id", 0, "Assignee user ID")
	ticketsCreateCmd.Flags().Int64("group-id", 0, "Group ID")
	ticketsCreateCmd.Flags().StringSlice("tags", nil, "Tags (comma-separated)")
	ticketsCreateCmd.Flags().StringArray("custom-field", nil, "Custom field (key=value, repeatable)")
	ticketsCreateCmd.Flags().String("requester-email", "", "Requester email")
	ticketsCreateCmd.Flags().String("requester-name", "", "Requester name")
	ticketsCreateCmd.Flags().String("idempotency-key", "", "Idempotency key for deduplication")
	ticketsCreateCmd.Flags().String("if-exists", "error", "When idempotent ticket exists: skip, update, error")

	ticketsCreateCmd.MarkFlagRequired("subject")
	ticketsCreateCmd.MarkFlagRequired("comment")
}

var ticketsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a ticket",
	RunE: func(cmd *cobra.Command, args []string) error {
		subject, _ := cmd.Flags().GetString("subject")
		comment, _ := cmd.Flags().GetString("comment")
		priority, _ := cmd.Flags().GetString("priority")
		ticketType, _ := cmd.Flags().GetString("type")
		status, _ := cmd.Flags().GetString("status")
		assigneeID, _ := cmd.Flags().GetInt64("assignee-id")
		groupID, _ := cmd.Flags().GetInt64("group-id")
		tags, _ := cmd.Flags().GetStringSlice("tags")
		customFieldStrs, _ := cmd.Flags().GetStringArray("custom-field")
		requesterEmail, _ := cmd.Flags().GetString("requester-email")
		requesterName, _ := cmd.Flags().GetString("requester-name")
		idempotencyKey, _ := cmd.Flags().GetString("idempotency-key")
		ifExists, _ := cmd.Flags().GetString("if-exists")

		customFields, err := parseCustomFields(customFieldStrs)
		if err != nil {
			return err
		}

		svc, err := newTicketService(cmd)
		if err != nil {
			return err
		}

		// Handle idempotency via tags
		if idempotencyKey != "" {
			tag := idempotencyTag(idempotencyKey)
			searchSvc, err := newSearchService(cmd)
			if err != nil {
				return err
			}

			existing, err := searchSvc.Search(cmd.Context(), fmt.Sprintf("tags:%s", tag), nil)
			if err != nil {
				return fmt.Errorf("checking idempotency: %w", err)
			}

			if existing.Count > 0 {
				switch ifExists {
				case "skip":
					formatter := formatterFromCtx(cmd.Context())
					return formatter.Format(os.Stdout, existing.Results[0].Ticket)
				case "update":
					// Fall through to update the existing ticket
					existingID := existing.Results[0].Ticket.ID
					req := &types.UpdateTicketRequest{
						Subject: subject,
						Comment: &types.Comment{Body: comment},
					}
					ticket, err := svc.Update(cmd.Context(), existingID, req)
					if err != nil {
						return err
					}
					formatter := formatterFromCtx(cmd.Context())
					return formatter.Format(os.Stdout, ticket)
				default:
					return fmt.Errorf("ticket with idempotency key %q already exists (ID: %d). Use --if-exists=skip or --if-exists=update", idempotencyKey, existing.Results[0].Ticket.ID)
				}
			}

			tags = append(tags, tag)
		}

		req := &types.CreateTicketRequest{
			Subject:        subject,
			Comment:        types.Comment{Body: comment},
			Priority:       priority,
			Type:           ticketType,
			Status:         status,
			AssigneeID:     assigneeID,
			GroupID:        groupID,
			Tags:           tags,
			CustomFields:   customFields,
			RequesterEmail: requesterEmail,
			RequesterName:  requesterName,
		}

		ticket, err := svc.Create(cmd.Context(), req)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())
		return formatter.Format(os.Stdout, ticket)
	},
}

func parseCustomFields(strs []string) ([]types.CustomField, error) {
	if len(strs) == 0 {
		return nil, nil
	}

	fields := make([]types.CustomField, 0, len(strs))
	for _, s := range strs {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid custom field format %q (expected key=value)", s)
		}

		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid custom field ID %q: must be numeric", parts[0])
		}

		fields = append(fields, types.CustomField{ID: id, Value: parts[1]})
	}
	return fields, nil
}

func idempotencyTag(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("zd-idempotent-%x", hash[:8])
}
