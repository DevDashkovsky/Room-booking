package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func TestGenerateAndParse(t *testing.T) {
	token, err := GenerateToken("00000000-0000-0000-0000-000000000001", "admin", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	userID, role, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if userID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("userID = %q, want %q", userID, "00000000-0000-0000-0000-000000000001")
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

func TestParseToken_RequiredClaimsAndAlgorithm(t *testing.T) {
	for _, tt := range []struct {
		name   string
		claims jwt.MapClaims
		method jwt.SigningMethod
	}{
		{"missing expiry", jwt.MapClaims{"user_id": "00000000-0000-0000-0000-000000000001", "role": "user"}, jwt.SigningMethodHS256},
		{"missing id", jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "role": "user"}, jwt.SigningMethodHS256},
		{"invalid id", jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "user_id": "bad", "role": "user"}, jwt.SigningMethodHS256},
		{"invalid role", jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "user_id": "00000000-0000-0000-0000-000000000001", "role": "owner"}, jwt.SigningMethodHS256},
		{"expired", jwt.MapClaims{"exp": time.Now().Add(-time.Hour).Unix(), "user_id": "00000000-0000-0000-0000-000000000001", "role": "user"}, jwt.SigningMethodHS256},
		{"HS384", jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix(), "user_id": "00000000-0000-0000-0000-000000000001", "role": "user"}, jwt.SigningMethodHS384},
	} {
		t.Run(tt.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(tt.method, tt.claims).SignedString([]byte(testSecret))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := ParseToken(token, testSecret); err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
}

func TestParseToken_NormalizesUserUUID(t *testing.T) {
	token, err := GenerateToken("ABCDEF01-ABCD-ABCD-ABCD-ABCDEF012345", "user", testSecret)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if id != "abcdef01-abcd-abcd-abcd-abcdef012345" {
		t.Fatalf("noncanonical UUID %s", id)
	}
}
