package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"

	"golang.org/x/crypto/bcrypt"

	"github.com/DevDashkovsky/room-booking/internal/domain"
	"github.com/DevDashkovsky/room-booking/internal/jwt"
)

const maxConcurrentBcrypt = 4

var bcryptSlots = make(chan struct{}, maxConcurrentBcrypt)

type userRepository interface {
	Create(context.Context, *domain.User) error
	GetByEmail(context.Context, string) (*domain.User, error)
}

type AuthService struct {
	userRepo  userRepository
	jwtSecret string
}

func NewAuthService(userRepo userRepository, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, email, password, role string) (*domain.User, error) {
	address, emailErr := mail.ParseAddress(email)
	if emailErr != nil || address.Address != email || password == "" || len(password) > 72 || (role != "admin" && role != "user") {
		return nil, domain.ErrInvalidRequest
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrEmailExists
	}

	if !acquireBcrypt(ctx) {
		return nil, domain.ErrUnavailable
	}
	hash, err := func() ([]byte, error) {
		defer releaseBcrypt()
		return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	}()
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrEmailExists) {
			return nil, domain.ErrEmailExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", domain.ErrUnauthorized
	}

	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	if u == nil {
		return "", domain.ErrUnauthorized
	}

	if !acquireBcrypt(ctx) {
		return "", domain.ErrUnavailable
	}
	err = func() error {
		defer releaseBcrypt()
		return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	}()
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	token, err := jwt.GenerateToken(u.ID, u.Role, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

func acquireBcrypt(ctx context.Context) bool {
	select {
	case bcryptSlots <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	default:
		return false
	}
}

func releaseBcrypt() {
	<-bcryptSlots
}
