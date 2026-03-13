package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMe(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/users/me", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"user":{"id":123,"name":"Test User","email":"test@example.com","role":"admin","active":true}}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	user, err := svc.GetMe(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(123), user.ID)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestAutocompleteUsers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/users/autocomplete", r.URL.Path)
		assert.Equal(t, "sarah", r.URL.Query().Get("name"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"users":[{"id":101,"name":"Sarah Chen","email":"sarah@example.com","role":"agent","active":true}]}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	users, err := svc.AutocompleteUsers(context.Background(), "sarah")
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "Sarah Chen", users[0].Name)
}

func TestAutocompleteUsers_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"users":[]}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	users, err := svc.AutocompleteUsers(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestGetMe_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	_, err := svc.GetMe(context.Background())
	require.Error(t, err)
}
