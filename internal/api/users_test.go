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
