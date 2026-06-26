package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

func TestAuthService_Register_EmptyEmail(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Register(context.Background(), "", "pass")
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("got %v, want ErrInvalidRequest", err)
	}
}

func TestAuthService_Register_EmptyPassword(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Register(context.Background(), "a@b.com", "")
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("got %v, want ErrInvalidRequest", err)
	}
}

func TestAuthService_Register_EmptyBoth(t *testing.T) {
	svc := &AuthService{}
	_, err := svc.Register(context.Background(), "", "")
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
