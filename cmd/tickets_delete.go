package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

var (
	pendingConfirmations   = map[string]int64{}
	pendingConfirmationsMu sync.Mutex
)

func init() {
	ticketsCmd.AddCommand(ticketsDeleteCmd)

	ticketsDeleteCmd.Flags().Bool("dry-run", false, "Preview deletion and return confirmation ID")
	ticketsDeleteCmd.Flags().String("confirm", "", "Execute deletion with confirmation ID from dry-run")
	ticketsDeleteCmd.Flags().Bool("yes", false, "Skip two-step confirmation")
}

var ticketsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a ticket",
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

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		confirmID, _ := cmd.Flags().GetString("confirm")
		yes, _ := cmd.Flags().GetBool("yes")
		globalYes, _ := cmd.Root().Flags().GetBool("yes")
		yes = yes || globalYes

		if dryRun {
			ticket, err := svc.Get(cmd.Context(), id, nil)
			if err != nil {
				return err
			}

			confirmation, err := generateConfirmationID()
			if err != nil {
				return fmt.Errorf("generating confirmation ID: %w", err)
			}

			pendingConfirmationsMu.Lock()
			pendingConfirmations[confirmation] = id
			pendingConfirmationsMu.Unlock()

			formatter := formatterFromCtx(cmd.Context())
			result := map[string]interface{}{
				"action":          "delete",
				"ticket_id":       ticket.ID,
				"subject":         ticket.Subject,
				"status":          ticket.Status,
				"confirmation_id": confirmation,
				"message":         fmt.Sprintf("Run 'zd tickets delete %d --confirm %s' to execute", id, confirmation),
			}
			return formatter.Format(os.Stdout, result)
		}

		if confirmID != "" {
			pendingConfirmationsMu.Lock()
			expectedID, ok := pendingConfirmations[confirmID]
			if ok {
				delete(pendingConfirmations, confirmID)
			}
			pendingConfirmationsMu.Unlock()

			if !ok {
				return fmt.Errorf("invalid or expired confirmation ID: %s (run --dry-run first)", confirmID)
			}
			if expectedID != id {
				return fmt.Errorf("confirmation ID was for ticket %d, not %d", expectedID, id)
			}
		} else if !yes {
			return fmt.Errorf("deletion requires --yes, --dry-run/--confirm, or interactive confirmation")
		}

		if err := svc.Delete(cmd.Context(), id); err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())
		result := map[string]interface{}{
			"deleted":   true,
			"ticket_id": id,
		}
		return formatter.Format(os.Stdout, result)
	},
}

func generateConfirmationID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
