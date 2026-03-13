package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/types"
)

func init() {
	ticketsCmd.AddCommand(ticketsShowCmd)

	ticketsShowCmd.Flags().String("include", "", "Sideload: users, groups, organizations")
	ticketsShowCmd.Flags().Bool("open", false, "Open ticket in browser after displaying")
}

var ticketsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a ticket",
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

		include, _ := cmd.Flags().GetString("include")
		opts := &types.GetTicketOptions{Include: include}

		result, err := svc.Get(cmd.Context(), id, opts)
		if err != nil {
			return err
		}

		formatter := formatterFromCtx(cmd.Context())

		var data interface{} = result.Ticket
		if len(result.Users) > 0 {
			userMap := buildUserMap(result.Users)
			data = enrichTicket(result.Ticket, userMap)
		}

		err = formatter.Format(os.Stdout, data)
		if openFlag, _ := cmd.Flags().GetBool("open"); openFlag {
			openTicketInBrowser(cmd, id)
		}
		return err
	},
}
