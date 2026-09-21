package category

import (
	"context"
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//go:generate go tool mockgen -source=repository.go -destination=../testutils/mocks/category/repository_mock.go -package mocks

// Repository defines the persistence interface for categories.
type Repository interface {
	Create(ctx context.Context, category *domain.Category) error
	FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Category, error)
	FindAll(ctx context.Context, userID uuid.UUID) ([]*domain.Category, error)
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type postgresRepository struct{ db *gorm.DB }

// NewRepository creates a new category repository.
func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, category *domain.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *postgresRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Category, error) {
	var cat domain.Category
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&cat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &cat, nil
}

func (r *postgresRepository) FindAll(ctx context.Context, userID uuid.UUID) ([]*domain.Category, error) {
	var categories []*domain.Category
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *postgresRepository) Update(ctx context.Context, category *domain.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

func (r *postgresRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&domain.Category{})
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	return result.Error
}
