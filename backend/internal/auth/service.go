package auth

import (
	"context"
	"time"

	"github.com/financeapp/backend/internal/config"
	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/middleware"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

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

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go

// Service defines the business logic interface for auth.
type Service interface {
	Register(req *RegisterRequest) (*AuthResponse, error)
	Login(req *LoginRequest) (*AuthResponse, error)
	Logout(token string) error
}

type service struct {
	repo Repository
	rdb  *redis.Client
	cfg  *config.JWTConfig
}

// NewService creates a new auth service.
func NewService(repo Repository, rdb *redis.Client, cfg *config.JWTConfig) Service {
	return &service{repo: repo, rdb: rdb, cfg: cfg}
}

func (s *service) Register(req *RegisterRequest) (*AuthResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Wrap(500, "failed to hash password", err)
	}

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}

	token, err := middleware.GenerateToken(user.ID, s.cfg)
	if err != nil {
		return nil, apperrors.Wrap(500, "failed to generate token", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *service) Login(req *LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return nil, apperrors.New(401, "invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, apperrors.New(401, "invalid credentials")
	}

	token, err := middleware.GenerateToken(user.ID, s.cfg)
	if err != nil {
		return nil, apperrors.Wrap(500, "failed to generate token", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *service) Logout(token string) error {
	claims, err := middleware.ParseToken(token, s.cfg)
	if err != nil {
		return nil // already invalid
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		s.rdb.Set(context.Background(), "blacklist:"+token, 1, ttl)
	}
	return nil
}
