package importer

import (
	"context"
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//go:generate go tool mockgen -source=repository.go -destination=../testutils/mocks/importer/repository_mock.go -package mocks

type Repository interface {
	Create(ctx context.Context, imp *domain.Import) error
	Update(ctx context.Context, imp *domain.Import) error
	FindAll(ctx context.Context, userID uuid.UUID) ([]*domain.Import, error)
	FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Import, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type postgresRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, imp *domain.Import) error {
	return r.db.WithContext(ctx).Create(imp).Error
}

func (r *postgresRepository) Update(ctx context.Context, imp *domain.Import) error {
	return r.db.WithContext(ctx).Save(imp).Error
}

func (r *postgresRepository) FindAll(ctx context.Context, userID uuid.UUID) ([]*domain.Import, error) {
	var imports []*domain.Import
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&imports).Error
	return imports, err
}

func (r *postgresRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Import, error) {
	var imp domain.Import
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&imp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &imp, nil
}

func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Import{}).Error
}
