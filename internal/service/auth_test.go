package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/DevDashkovsky/room-booking/internal/domain"
)

type loginUserRepository struct{ user *domain.User }

func (r loginUserRepository) GetByEmail(context.Context, string) (*domain.User, error) {
	return r.user, nil
}

func (r loginUserRepository) Create(context.Context, *domain.User) error { return nil }

func TestAuthService_LoginPasswordByteLimit(t *testing.T) {
	for _, password := range []string{strings.Repeat("a", 72), strings.Repeat("я", 36)} {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		svc := NewAuthService(loginUserRepository{user: &domain.User{
			ID: "00000000-0000-0000-0000-000000000002", Role: "user", PasswordHash: string(hash),
		}}, "secret")
		if token, err := svc.Login(t.Context(), "user@example.com", password); err != nil || token == "" {
			t.Fatalf("72-byte password rejected: token=%q err=%v", token, err)
		}
		if token, err := svc.Login(t.Context(), "user@example.com", password+"suffix"); !errors.Is(err, domain.ErrUnauthorized) || token != "" {
			t.Fatalf("password suffix accepted: token present=%v err=%v", token != "", err)
		}
	}
}

func TestAcquireBcrypt_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for range 100 {
		if acquireBcrypt(ctx) {
			releaseBcrypt()
			t.Fatal("canceled request acquired bcrypt capacity")
		}
	}
}

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

func TestBcryptConcurrencyLimit(t *testing.T) {
	for range maxConcurrentBcrypt {
		if !acquireBcrypt(context.Background()) {
			t.Fatal("failed to acquire an available bcrypt slot")
		}
	}
	defer func() {
		for range maxConcurrentBcrypt {
			releaseBcrypt()
		}
	}()

	if acquireBcrypt(context.Background()) {
		t.Fatal("acquired more bcrypt slots than allowed")
	}
}
