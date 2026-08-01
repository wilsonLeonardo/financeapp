package auth_test

import (
	"net/http"
	"testing"

	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/config"
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
)

// ── Minimal in-memory repository for testing ──────────────────────────────────

type inMemoryRepo struct {
	users map[string]*domain.User
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{users: make(map[string]*domain.User)}
}

func (r *inMemoryRepo) CreateUser(user *domain.User) error {
	for _, u := range r.users {
		if u.Email == user.Email {
			return apperrors.ErrConflict
		}
	}
	user.ID = uuid.New()
	r.users[user.ID.String()] = user
	return nil
}

func (r *inMemoryRepo) FindUserByEmail(email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (r *inMemoryRepo) FindUserByID(id string) (*domain.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, apperrors.ErrNotFound
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	repo := newInMemoryRepo()
	svc := auth.NewService(repo, nil, &config.JWTConfig{Secret: "test-secret", ExpiryHours: 1})

	resp, err := svc.Register(&auth.RegisterRequest{
		Name:     "João Silva",
		Email:    "joao@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if resp.User.Email != "joao@example.com" {
		t.Errorf("expected email joao@example.com, got %s", resp.User.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newInMemoryRepo()
	svc := auth.NewService(repo, nil, &config.JWTConfig{Secret: "test-secret", ExpiryHours: 1})

	req := &auth.RegisterRequest{Name: "Test", Email: "dup@example.com", Password: "password123"}
	_, _ = svc.Register(req)
	_, err := svc.Register(req)

	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := newInMemoryRepo()
	svc := auth.NewService(repo, nil, &config.JWTConfig{Secret: "test-secret", ExpiryHours: 1})

	_, _ = svc.Register(&auth.RegisterRequest{Name: "Test", Email: "user@example.com", Password: "correct-pass"})

	_, err := svc.Login(&auth.LoginRequest{Email: "user@example.com", Password: "wrong-pass"})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	if appErr, ok := err.(*apperrors.AppError); !ok || appErr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 error, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newInMemoryRepo()
	svc := auth.NewService(repo, nil, &config.JWTConfig{Secret: "test-secret", ExpiryHours: 1})

	_, _ = svc.Register(&auth.RegisterRequest{Name: "Test", Email: "user@example.com", Password: "mypassword"})

	resp, err := svc.Login(&auth.LoginRequest{Email: "user@example.com", Password: "mypassword"})
	if err != nil {
		t.Fatalf("expected login success, got: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected non-empty token after login")
	}
}
