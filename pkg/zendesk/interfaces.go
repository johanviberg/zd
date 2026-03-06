package zendesk

import (
	"context"

	"github.com/johanviberg/zd/internal/types"
)

//go:generate mockgen -destination=../../internal/mocks/mock_zendesk.go -package=mocks github.com/johanviberg/zd/pkg/zendesk TicketService,SearchService

type TicketService interface {
	List(ctx context.Context, opts *types.ListTicketsOptions) (*types.TicketPage, error)
	Get(ctx context.Context, id int64, opts *types.GetTicketOptions) (*types.TicketResult, error)
	Create(ctx context.Context, req *types.CreateTicketRequest) (*types.Ticket, error)
	Update(ctx context.Context, id int64, req *types.UpdateTicketRequest) (*types.Ticket, error)
	Delete(ctx context.Context, id int64) error
}

type SearchService interface {
	Search(ctx context.Context, query string, opts *types.SearchOptions) (*types.SearchPage, error)
}
