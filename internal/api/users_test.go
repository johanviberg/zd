package api

import (
	"context"
	"net/http"
	"testing"
)

func TestGetMe(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/users/me" {
			t.Errorf("expected path /api/v2/users/me, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"user":{"id":123,"name":"Test User","email":"test@example.com","role":"admin","active":true}}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	user, err := svc.GetMe(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 123 {
		t.Errorf("expected ID 123, got %d", user.ID)
	}
	if user.Name != "Test User" {
		t.Errorf("expected name 'Test User', got %q", user.Name)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", user.Email)
	}
}

func TestAutocompleteUsers(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/users/autocomplete" {
			t.Errorf("expected path /api/v2/users/autocomplete, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "sarah" {
			t.Errorf("expected name=sarah, got %s", r.URL.Query().Get("name"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"users":[{"id":101,"name":"Sarah Chen","email":"sarah@example.com","role":"agent","active":true}]}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	users, err := svc.AutocompleteUsers(context.Background(), "sarah")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Name != "Sarah Chen" {
		t.Errorf("expected name 'Sarah Chen', got %q", users[0].Name)
	}
}

func TestAutocompleteUsers_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"users":[]}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	users, err := svc.AutocompleteUsers(context.Background(), "nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestGetMe_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	})

	client := testClient(t, handler)
	svc := NewUserService(client)

	_, err := svc.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
