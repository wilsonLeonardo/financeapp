package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/pkg/config"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/financeapp/backend/pkg/security"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

//go:generate go tool mockgen -source=service.go -destination=../testutils/mocks/auth/service_mock.go -package mocks

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is the response after a successful auth operation.
type AuthResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// Service defines the business logic interface for auth.
type Service interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	Logout(ctx context.Context, token string) error
}

type service struct {
	repo Repository
	rdb  *redis.Client
	cfg  *config.JWTConfig
	log  *slog.Logger
}

// NewService creates a new auth service.
func NewService(repo Repository, rdb *redis.Client, cfg *config.JWTConfig, log *slog.Logger) Service {
	return &service{repo: repo, rdb: rdb, cfg: cfg, log: log}
}

func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to hash password", err)
	}

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.issueToken(user.ID)
	if err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to generate token", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.New(401, "invalid credentials")
	}

	if err := security.CheckPassword(user.Password, req.Password); err != nil {
		return nil, apperrors.New(401, "invalid credentials")
	}

	token, err := s.issueToken(user.ID)
	if err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to generate token", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	claims, err := security.ParseToken(token, s.cfg.Secret)
	if err != nil {
		return nil // already invalid
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		if err := s.rdb.Set(ctx, "blacklist:"+token, 1, ttl).Err(); err != nil {
			// The caller is told the logout worked either way, so a token that
			// stays valid until it expires must at least be visible here.
			s.log.Error("failed to revoke token", "error", err, "user_id", claims.UserID)
		}
	}
	return nil
}

// issueToken signs an access token using the configured secret and expiry.
func (s *service) issueToken(userID uuid.UUID) (string, error) {
	return security.GenerateToken(userID, s.cfg.Secret, time.Duration(s.cfg.ExpiryHours)*time.Hour)
}
