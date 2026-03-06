package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(configShowCmd)
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := configFromCtx(cmd.Context())
		formatter := formatterFromCtx(cmd.Context())

		data := map[string]interface{}{
			"profile":   cfg.Profile,
			"subdomain": cfg.Subdomain,
		}
		if cfg.OAuthClientID != "" {
			data["oauth_client_id"] = cfg.OAuthClientID
		}

		return formatter.Format(os.Stdout, data)
	},
}
