package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(authCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Login, logout, and check authentication status with Zendesk.",
}
