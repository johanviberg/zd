package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/config"
)

func init() {
	authCmd.AddCommand(authLoginCmd)

	authLoginCmd.Flags().String("method", "oauth", "Authentication method: oauth or token")
	authLoginCmd.Flags().String("email", "", "Zendesk email (for token auth)")
	authLoginCmd.Flags().String("api-token", "", "Zendesk API token (for token auth)")
	authLoginCmd.Flags().String("subdomain", "", "Zendesk subdomain")
	authLoginCmd.Flags().String("client-id", "", "OAuth client ID")
	authLoginCmd.Flags().String("client-secret", "", "OAuth client secret")
	authLoginCmd.Flags().String("scope", "read write", "OAuth scope")
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Zendesk",
	Long:  "Login to Zendesk using OAuth or API token authentication.",
	RunE: func(cmd *cobra.Command, args []string) error {
		method, _ := cmd.Flags().GetString("method")
		subdomain, _ := cmd.Flags().GetString("subdomain")
		profile, _ := cmd.Flags().GetString("profile")

		cfg := configFromCtx(cmd.Context())
		if subdomain == "" {
			subdomain = cfg.Subdomain
		}

		if subdomain == "" {
			return fmt.Errorf("subdomain is required: use --subdomain or set ZENDESK_SUBDOMAIN")
		}

		switch method {
		case "token":
			return loginWithToken(cmd, profile, subdomain)
		case "oauth":
			return loginWithOAuth(cmd, profile, subdomain, cfg)
		default:
			return fmt.Errorf("unknown auth method: %s (use 'oauth' or 'token')", method)
		}
	},
}

func loginWithToken(cmd *cobra.Command, profile, subdomain string) error {
	email, _ := cmd.Flags().GetString("email")
	apiToken, _ := cmd.Flags().GetString("api-token")

	if email == "" {
		return fmt.Errorf("--email is required for token auth")
	}
	if apiToken == "" {
		return fmt.Errorf("--api-token is required for token auth")
	}

	creds := &auth.ProfileCredentials{
		Method:    "token",
		Subdomain: subdomain,
		Email:     email,
		APIToken:  apiToken,
	}

	if err := auth.SaveCredentials(profile, creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	cfg := &config.Config{Subdomain: subdomain}
	if err := config.Save(profile, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Authenticated as %s on %s.zendesk.com (token auth)\n", email, subdomain)
	return nil
}

func loginWithOAuth(cmd *cobra.Command, profile, subdomain string, cfg *config.Config) error {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")

	// Use stored config value for client ID
	if clientID == "" {
		clientID = cfg.OAuthClientID
	}

	// Fall back to credentials for stored OAuth client ID
	if clientID == "" {
		if existingCreds, _ := auth.LoadCredentials(profile); existingCreds != nil {
			if clientID == "" {
				clientID = existingCreds.OAuthClientID
			}
		}
	}

	if clientID == "" {
		return fmt.Errorf("--client-id is required for first-time OAuth login")
	}

	scope, _ := cmd.Flags().GetString("scope")
	if scope == "" {
		return fmt.Errorf("--scope must not be empty")
	}

	result, err := auth.OAuthFlow(subdomain, clientID, clientSecret, scope)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	creds := &auth.ProfileCredentials{
		Method:         "oauth",
		Subdomain:      subdomain,
		OAuthToken:     result.AccessToken,
		OAuthClientID:  clientID,
		RefreshToken:   result.RefreshToken,
		TokenExpiresAt: result.ExpiresAt,
	}

	if err := auth.SaveCredentials(profile, creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	saveCfg := &config.Config{
		Subdomain:     subdomain,
		OAuthClientID: clientID,
	}
	if err := config.Save(profile, saveCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	msg := fmt.Sprintf("Authenticated on %s.zendesk.com (OAuth)", subdomain)
	if result.RefreshToken != "" {
		msg += " — token will auto-refresh"
	}
	fmt.Fprintln(os.Stderr, msg)
	return nil
}
