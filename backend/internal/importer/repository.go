package importer

import (
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//go:generate go tool mockgen -source=repository.go -destination=../testutils/mocks/importer/repository_mock.go -package mocks

type Repository interface {
	Create(imp *domain.Import) error
	Update(imp *domain.Import) error
	FindAll(userID uuid.UUID) ([]*domain.Import, error)
	FindByID(id, userID uuid.UUID) (*domain.Import, error)
	Delete(id uuid.UUID) error
}

type postgresRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(imp *domain.Import) error {
	return r.db.Create(imp).Error
}

func (r *postgresRepository) Update(imp *domain.Import) error {
	return r.db.Save(imp).Error
}

func (r *postgresRepository) FindAll(userID uuid.UUID) ([]*domain.Import, error) {
	var imports []*domain.Import
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&imports).Error
	return imports, err
}

func (r *postgresRepository) FindByID(id, userID uuid.UUID) (*domain.Import, error) {
	var imp domain.Import
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&imp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &imp, nil
}

func (r *postgresRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Import{}).Error
}
