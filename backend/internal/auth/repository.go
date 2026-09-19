package auth

import (
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"gorm.io/gorm"
	"log/slog"
)

//go:generate go tool mockgen -source=repository.go -destination=../testutils/mocks/auth/repository_mock.go -package mocks

// Repository defines the persistence interface for auth operations.
type Repository interface {
	CreateUser(user *domain.User) error
	FindUserByEmail(email string) (*domain.User, error)
	FindUserByID(id string) (*domain.User, error)
}

type postgresRepository struct {
	db  *gorm.DB
	log *slog.Logger
}

// NewRepository creates a new PostgreSQL auth repository.
func NewRepository(db *gorm.DB, log *slog.Logger) Repository {
	return &postgresRepository{db: db, log: log}
}

func (r *postgresRepository) CreateUser(user *domain.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return apperrors.Wrap(409, "email already registered", err)
	}
	return nil
}

func (r *postgresRepository) FindUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.WrapLogged(r.log, "failed to find user", err)
	}
	return &user, nil
}

func (r *postgresRepository) FindUserByID(id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.WrapLogged(r.log, "failed to find user", err)
	}
	return &user, nil
}
