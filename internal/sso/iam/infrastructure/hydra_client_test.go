package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestGetLoginRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("login_challenge") != "test-challenge" {
			t.Errorf("expected login_challenge=test-challenge")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"subject": "42",
			"skip":    false,
			"client":  map[string]any{"client_id": "demo-app"},
		})
	}))
	defer srv.Close()

	client := NewHydraClient(srv.URL, zap.NewNop())
	req, err := client.GetLoginRequest(context.Background(), "test-challenge")
	if err != nil {
		t.Fatalf("GetLoginRequest failed: %v", err)
	}
	if req.Subject != "42" {
		t.Errorf("Subject = %q, want 42", req.Subject)
	}
	if req.Challenge != "test-challenge" {
		t.Errorf("Challenge = %q, want test-challenge", req.Challenge)
	}
}

func TestGetLoginRequestGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://example.com"})
	}))
	defer srv.Close()

	client := NewHydraClient(srv.URL, zap.NewNop())
	_, err := client.GetLoginRequest(context.Background(), "used-challenge")
	if err == nil {
		t.Fatal("expected ErrLoginRequestGone")
	}
	gone, ok := err.(*ErrLoginRequestGone)
	if !ok {
		t.Fatalf("expected *ErrLoginRequestGone, got %T", err)
	}
	if gone.RedirectTo != "https://example.com" {
		t.Errorf("RedirectTo = %q, want https://example.com", gone.RedirectTo)
	}
}

func TestAcceptLoginRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://hydra/callback"})
	}))
	defer srv.Close()

	client := NewHydraClient(srv.URL, zap.NewNop())
	resp, err := client.AcceptLoginRequest(context.Background(), "challenge", "42")
	if err != nil {
		t.Fatalf("AcceptLoginRequest failed: %v", err)
	}
	if resp.RedirectTo != "https://hydra/callback" {
		t.Errorf("RedirectTo = %q", resp.RedirectTo)
	}
}

func TestDeleteLoginSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewHydraClient(srv.URL, zap.NewNop())
	err := client.DeleteLoginSession(context.Background(), "42")
	if err != nil {
		t.Fatalf("DeleteLoginSession failed: %v", err)
	}
}

func TestPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	client := NewHydraClient(srv.URL, zap.NewNop())
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}
