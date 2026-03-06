package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/auth"
)

func init() {
	authCmd.AddCommand(authLogoutCmd)
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")

		if err := auth.DeleteCredentials(profile); err != nil {
			return fmt.Errorf("deleting credentials: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Logged out of profile %q\n", profile)
		return nil
	},
}
