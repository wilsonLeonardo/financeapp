package auth

import (
	"context"
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"gorm.io/gorm"
	"log/slog"
)

//go:generate go tool mockgen -source=repository.go -destination=../testutils/mocks/auth/repository_mock.go -package mocks

// Repository defines the persistence interface for auth operations.
type Repository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
	FindUserByID(ctx context.Context, id string) (*domain.User, error)
}

type postgresRepository struct {
	db  *gorm.DB
	log *slog.Logger
}

// NewRepository creates a new PostgreSQL auth repository.
func NewRepository(db *gorm.DB, log *slog.Logger) Repository {
	return &postgresRepository{db: db, log: log}
}

func (r *postgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return apperrors.Wrap(409, "email already registered", err)
	}
	return nil
}

func (r *postgresRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.WrapLogged(r.log, "failed to find user", err)
	}
	return &user, nil
}

func (r *postgresRepository) FindUserByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.WrapLogged(r.log, "failed to find user", err)
	}
	return &user, nil
}
