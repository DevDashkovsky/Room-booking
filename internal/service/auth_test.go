package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func TestAuthService_Register_EmptyEmail(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Register(context.Background(), "", "pass", "user")
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("got %v, want ErrInvalidRequest", err)
	}
}

func TestAuthService_Register_EmptyPassword(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Register(context.Background(), "a@b.com", "", "user")
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("got %v, want ErrInvalidRequest", err)
	}
}

func TestAuthService_Register_EmptyBoth(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Register(context.Background(), "", "", "user")
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("got %v, want ErrInvalidRequest", err)
	}
}

func TestAuthService_Login_EmptyBoth(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Login(context.Background(), "", "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestAuthService_Login_EmptyEmail(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Login(context.Background(), "", "pass")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestAuthService_Login_EmptyPassword(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Login(context.Background(), "a@b.com", "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestAuthService_RegisterValidation(t *testing.T) {
	svc := NewAuthService(nil, "secret")
	for _, tt := range []struct{ email, password, role string }{
		{"invalid", "pass", "user"}, {"Name <user@example.com>", "pass", "user"}, {"user@example.com", strings.Repeat("x", 73), "user"},
		{"user@example.com", strings.Repeat("я", 37), "user"}, {"user@example.com", "pass", ""}, {"user@example.com", "pass", "owner"},
	} {
		if _, err := svc.Register(context.Background(), tt.email, tt.password, tt.role); !errors.Is(err, domain.ErrInvalidRequest) {
			t.Fatalf("accepted invalid registration: %v", err)
		}
	}
}
