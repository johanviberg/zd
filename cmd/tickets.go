package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/api"
	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

func init() {
	rootCmd.AddCommand(ticketsCmd)
}

var ticketsCmd = &cobra.Command{
	Use:   "tickets",
	Short: "Manage Zendesk tickets",
	Long:  "List, show, create, update, delete, and search Zendesk tickets.",
}

func newTicketService(cmd *cobra.Command) (zendesk.TicketService, error) {
	cfg := configFromCtx(cmd.Context())
	profile, _ := cmd.Flags().GetString("profile")
	traceID, _ := cmd.Flags().GetString("trace-id")

	creds, err := auth.ResolveCredentials(profile)
	if err != nil {
		return nil, err
	}
	if creds == nil {
		return nil, types.NewAuthError("not authenticated — run 'zd auth login' first")
	}

	subdomain := cfg.Subdomain
	if subdomain == "" {
		subdomain = creds.Subdomain
	}
	if subdomain == "" {
		return nil, types.NewArgError("subdomain is required")
	}

	client, err := api.NewClient(subdomain, creds, traceID)
	if err != nil {
		return nil, err
	}
	return api.NewTicketService(client), nil
}

func newSearchService(cmd *cobra.Command) (zendesk.SearchService, error) {
	cfg := configFromCtx(cmd.Context())
	profile, _ := cmd.Flags().GetString("profile")
	traceID, _ := cmd.Flags().GetString("trace-id")

	creds, err := auth.ResolveCredentials(profile)
	if err != nil {
		return nil, err
	}
	if creds == nil {
		return nil, types.NewAuthError("not authenticated — run 'zd auth login' first")
	}

	subdomain := cfg.Subdomain
	if subdomain == "" {
		subdomain = creds.Subdomain
	}
	if subdomain == "" {
		return nil, types.NewArgError("subdomain is required")
	}

	client, err := api.NewClient(subdomain, creds, traceID)
	if err != nil {
		return nil, err
	}
	return api.NewSearchService(client), nil
}
