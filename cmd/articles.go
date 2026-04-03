package cmd

import (
	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/api"
	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/types"
	"github.com/johanviberg/zd/pkg/zendesk"
)

func init() {
	rootCmd.AddCommand(articlesCmd)
}

var articlesCmd = &cobra.Command{
	Use:   "articles",
	Short: "Manage Help Center articles",
	Long:  "List, show, and search Zendesk Help Center articles.",
}

func newArticleService(cmd *cobra.Command) (zendesk.ArticleService, error) {
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

	client, err := api.NewClient(subdomain, creds, profile, traceID)
	if err != nil {
		return nil, err
	}
	return api.NewArticleService(client), nil
}
