package category

import (
	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"log/slog"
)

//go:generate go tool mockgen -source=service.go -destination=../testutils/mocks/category/service_mock.go -package mocks

// UpsertRequest is the payload for create/update.
type UpsertRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=50"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// Service defines the business logic interface for categories.
type Service interface {
	Create(userID uuid.UUID, req *UpsertRequest) (*domain.Category, error)
	GetAll(userID uuid.UUID) ([]*domain.Category, error)
	Update(id, userID uuid.UUID, req *UpsertRequest) (*domain.Category, error)
	Delete(id, userID uuid.UUID) error
}

type service struct {
	repo Repository
	log  *slog.Logger
}

// NewService creates a new category service.
func NewService(repo Repository, log *slog.Logger) Service {
	return &service{repo: repo, log: log}
}

func (s *service) Create(userID uuid.UUID, req *UpsertRequest) (*domain.Category, error) {
	cat := &domain.Category{
		UserID: userID,
		Name:   req.Name,
		Icon:   req.Icon,
		Color:  req.Color,
	}
	if err := s.repo.Create(cat); err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to create category", err)
	}
	return cat, nil
}

func (s *service) GetAll(userID uuid.UUID) ([]*domain.Category, error) {
	return s.repo.FindAll(userID)
}

func (s *service) Update(id, userID uuid.UUID, req *UpsertRequest) (*domain.Category, error) {
	cat, err := s.repo.FindByID(id, userID)
	if err != nil {
		return nil, err
	}
	cat.Name = req.Name
	cat.Icon = req.Icon
	cat.Color = req.Color
	if err := s.repo.Update(cat); err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to update category", err)
	}
	return cat, nil
}

func (s *service) Delete(id, userID uuid.UUID) error {
	return s.repo.Delete(id, userID)
}
