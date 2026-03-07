package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/auth"
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

		cfg := configFromCtx(cmd.Context())
		profile, _ := cmd.Flags().GetString("profile")
		subdomain := cfg.Subdomain
		if subdomain == "" {
			if creds, _ := auth.ResolveCredentials(profile); creds != nil {
				subdomain = creds.Subdomain
			}
		}

		app := tui.NewApp(ticketSvc, searchSvc, userSvc, subdomain)
		p := tea.NewProgram(app, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}
		return nil
	},
}
