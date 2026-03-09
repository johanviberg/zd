package demo

import (
	"context"
	"strings"

	"github.com/johanviberg/zd/internal/types"
)

type UserService struct {
	store *Store
}

func NewUserService(store *Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) GetMe(ctx context.Context) (*types.User, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	for i := range s.store.Users {
		if s.store.Users[i].Role == "agent" {
			u := s.store.Users[i]
			return &u, nil
		}
	}
	return nil, types.NewNotFoundError("no agent user found")
}

func (s *UserService) AutocompleteUsers(ctx context.Context, name string) ([]types.User, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	query := strings.ToLower(name)
	var matches []types.User
	for _, u := range s.store.Users {
		if strings.Contains(strings.ToLower(u.Name), query) || strings.Contains(strings.ToLower(u.Email), query) {
			matches = append(matches, u)
		}
	}
	return matches, nil
}
