package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/types"
)

func init() {
	authCmd.AddCommand(authStatusCmd)
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")

		creds, err := auth.ResolveCredentials(profile)
		if err != nil {
			return fmt.Errorf("resolving credentials: %w", err)
		}

		if creds == nil {
			return types.NewAuthError("not authenticated — run 'zd auth login' first")
		}

		formatter := formatterFromCtx(cmd.Context())

		status := map[string]interface{}{
			"profile":   profile,
			"method":    creds.Method,
			"subdomain": creds.Subdomain,
		}
		if creds.Email != "" {
			status["email"] = creds.Email
		}
		status["authenticated"] = true

		if creds.Method == "oauth" {
			if creds.TokenExpiresAt != nil {
				status["token_expires_at"] = creds.TokenExpiresAt.Format("2006-01-02T15:04:05Z07:00")
				if creds.IsTokenExpired() {
					status["token_status"] = "expired"
				} else {
					status["token_status"] = "valid"
				}
			}
			status["auto_refresh"] = creds.RefreshToken != ""
		}

		return formatter.Format(os.Stdout, status)
	},
}
