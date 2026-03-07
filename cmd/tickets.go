package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/johanviberg/zd/internal/api"
	"github.com/johanviberg/zd/internal/auth"
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

func newTicketService(cmd *cobra.Command) (zendesk.TicketService, error) {
	if store := demoStoreFromCtx(cmd.Context()); store != nil {
		return demo.NewTicketService(store), nil
	}
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
	if store := demoStoreFromCtx(cmd.Context()); store != nil {
		return demo.NewSearchService(store), nil
	}
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

func newUserService(cmd *cobra.Command) (zendesk.UserService, error) {
	if store := demoStoreFromCtx(cmd.Context()); store != nil {
		return demo.NewUserService(store), nil
	}
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
	return api.NewUserService(client), nil
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
