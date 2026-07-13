package jwtx

import "testing"

func TestSignAndParse(t *testing.T) {
	secret := "test-secret-key"
	token, err := Sign(secret, 42, "alice", true, 3600)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := Parse(secret, token)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want alice", claims.Username)
	}
	if !claims.IsAdmin {
		t.Error("IsAdmin should be true")
	}
}

func TestParseInvalidToken(t *testing.T) {
	_, err := Parse("secret", "not-a-valid-token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, _ := Sign("secret-a", 1, "bob", false, 3600)
	_, err := Parse("secret-b", token)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for wrong secret, got %v", err)
	}
}

func TestSignDefaultExpire(t *testing.T) {
	// expireSeconds <= 0 时应使用默认值 86400
	token, err := Sign("secret", 1, "test", false, 0)
	if err != nil {
		t.Fatalf("Sign with 0 expire failed: %v", err)
	}
	claims, err := Parse("secret", token)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt should not be nil")
	}
}
