package jwt

import (
	"testing"
)

const testSecret = "test-secret"

func TestGenerateAndParse(t *testing.T) {
	token, err := GenerateToken("user-123", "admin", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	userID, role, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("userID = %q, want %q", userID, "user-123")
	}
	if role != "admin" {
		t.Errorf("role = %q, want %q", role, "admin")
	}
}

func TestParseInvalidToken(t *testing.T) {
	_, _, err := ParseToken("garbage", testSecret)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, _ := GenerateToken("u", "user", testSecret)
	_, _, err := ParseToken(token, "wrong-secret")
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}
