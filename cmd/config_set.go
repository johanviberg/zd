package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/config"
)

func init() {
	configCmd.AddCommand(configSetCmd)
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		profile, _ := cmd.Flags().GetString("profile")

		if err := config.SetValue(profile, key, value); err != nil {
			return fmt.Errorf("setting config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Set %s = %s (profile: %s)\n", key, value, profile)
		return nil
	},
}
