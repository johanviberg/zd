package cmd

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/cache"
	"github.com/johanviberg/zd/internal/tui"
)

func init() {
	rootCmd.AddCommand(tuiCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI for managing tickets",
	Long:  "Launch an interactive terminal interface for browsing, viewing, and managing Zendesk tickets.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketSvc, err := newTicketService(cmd)
		if err != nil {
			return err
		}

		searchSvc, err := newSearchService(cmd)
		if err != nil {
			return err
		}

		userSvc, err := newUserService(cmd)
		if err != nil {
			return err
		}

		// Wrap with cache for non-demo mode to reduce API calls
		if demoStoreFromCtx(cmd.Context()) == nil {
			c := cache.New(60 * time.Second)
			ticketSvc = cache.NewCachedTicketService(ticketSvc, c)
			searchSvc = cache.NewCachedSearchService(searchSvc, c)
		}

		cfg := configFromCtx(cmd.Context())
		profile, _ := cmd.Flags().GetString("profile")
		subdomain := cfg.Subdomain
		if subdomain == "" {
			if creds, _ := auth.ResolveCredentials(profile); creds != nil {
				subdomain = creds.Subdomain
			}
		}

		app := tui.NewApp(ticketSvc, searchSvc, userSvc, subdomain, buildVersion)
		p := tea.NewProgram(app)

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}
		return nil
	},
}
