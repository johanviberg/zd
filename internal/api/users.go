package api

import (
	"context"

	"github.com/johanviberg/zd/internal/types"
)

type UserService struct {
	client *Client
}

func NewUserService(client *Client) *UserService {
	return &UserService{client: client}
}

func (s *UserService) GetMe(ctx context.Context) (*types.User, error) {
	var result struct {
		User types.User `json:"user"`
	}
	if err := s.client.doJSON(ctx, "GET", "/api/v2/users/me", nil, &result); err != nil {
		return nil, err
	}
	return &result.User, nil
}
