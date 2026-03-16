package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/api"
	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/browser"
	"github.com/johanviberg/zd/internal/demo"
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

func buildClient(cmd *cobra.Command) (*api.Client, error) {
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

	return api.NewClient(subdomain, creds, traceID)
}

func newTicketService(cmd *cobra.Command) (zendesk.TicketService, error) {
	if store := demoStoreFromCtx(cmd.Context()); store != nil {
		return demo.NewTicketService(store), nil
	}
	client, err := buildClient(cmd)
	if err != nil {
		return nil, err
	}
	return api.NewTicketService(client), nil
}

func newSearchService(cmd *cobra.Command) (zendesk.SearchService, error) {
	if store := demoStoreFromCtx(cmd.Context()); store != nil {
		return demo.NewSearchService(store), nil
	}
	client, err := buildClient(cmd)
	if err != nil {
		return nil, err
	}
	return api.NewSearchService(client), nil
}

func newUserService(cmd *cobra.Command) (zendesk.UserService, error) {
	if store := demoStoreFromCtx(cmd.Context()); store != nil {
		return demo.NewUserService(store), nil
	}
	client, err := buildClient(cmd)
	if err != nil {
		return nil, err
	}
	return api.NewUserService(client), nil
}

func resolveSubdomain(cmd *cobra.Command) string {
	cfg := configFromCtx(cmd.Context())
	if cfg.Subdomain != "" {
		return cfg.Subdomain
	}
	profile, _ := cmd.Flags().GetString("profile")
	creds, err := auth.ResolveCredentials(profile)
	if err != nil || creds == nil {
		return ""
	}
	return creds.Subdomain
}

func openTicketInBrowser(cmd *cobra.Command, ticketID int64) {
	subdomain := resolveSubdomain(cmd)
	if subdomain == "" {
		fmt.Fprintln(os.Stderr, "Warning: cannot open browser — subdomain not configured")
		return
	}
	browser.Open(fmt.Sprintf("https://%s.zendesk.com/agent/tickets/%d", subdomain, ticketID))
}

func buildUserMap(users []types.User) map[int64]types.User {
	m := make(map[int64]types.User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return m
}

func enrichTicket(ticket interface{}, userMap map[int64]types.User) interface{} {
	if len(userMap) == 0 {
		return ticket
	}

	b, err := json.Marshal(ticket)
	if err != nil {
		return ticket
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return ticket
	}

	if rid, ok := m["requester_id"].(float64); ok {
		if u, found := userMap[int64(rid)]; found {
			m["requester_name"] = u.Name
			m["requester_email"] = u.Email
		}
	}
	if aid, ok := m["assignee_id"].(float64); ok {
		if u, found := userMap[int64(aid)]; found {
			m["assignee_name"] = u.Name
			m["assignee_email"] = u.Email
		}
	}

	return m
}
