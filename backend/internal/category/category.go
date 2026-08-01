package category

import (
	"net/http"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/middleware"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Repository ────────────────────────────────────────────────────────────────

// Repository defines the persistence interface for categories.
type Repository interface {
	Create(category *domain.Category) error
	FindByID(id, userID uuid.UUID) (*domain.Category, error)
	FindAll(userID uuid.UUID) ([]*domain.Category, error)
	Update(category *domain.Category) error
	Delete(id, userID uuid.UUID) error
}

type postgresRepository struct{ db *gorm.DB }

// NewRepository creates a new category repository.
func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(category *domain.Category) error {
	return r.db.Create(category).Error
}

func (r *postgresRepository) FindByID(id, userID uuid.UUID) (*domain.Category, error) {
	var cat domain.Category
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&cat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &cat, nil
}

func (r *postgresRepository) FindAll(userID uuid.UUID) ([]*domain.Category, error) {
	var categories []*domain.Category
	err := r.db.Where("user_id = ?", userID).Order("name ASC").Find(&categories).Error
	return categories, err
}

func (r *postgresRepository) Update(category *domain.Category) error {
	return r.db.Save(category).Error
}

func (r *postgresRepository) Delete(id, userID uuid.UUID) error {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&domain.Category{})
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	return result.Error
}

// ── Service ───────────────────────────────────────────────────────────────────

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

type service struct{ repo Repository }

// NewService creates a new category service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(userID uuid.UUID, req *UpsertRequest) (*domain.Category, error) {
	cat := &domain.Category{
		UserID: userID,
		Name:   req.Name,
		Icon:   req.Icon,
		Color:  req.Color,
	}
	if err := s.repo.Create(cat); err != nil {
		return nil, apperrors.Wrap(500, "failed to create category", err)
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
		return nil, apperrors.Wrap(500, "failed to update category", err)
	}
	return cat, nil
}

func (s *service) Delete(id, userID uuid.UUID) error {
	return s.repo.Delete(id, userID)
}

// ── Handler ───────────────────────────────────────────────────────────────────

// Handler handles HTTP requests for category routes.
type Handler struct{ service Service }

// NewHandler creates a new category handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	cat, err := h.service.Create(userID, &req)
	if err != nil {
		respondWithError(c, err)
		return
	}
	c.JSON(http.StatusCreated, cat)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	categories, err := h.service.GetAll(userID)
	if err != nil {
		respondWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	cat, err := h.service.Update(id, userID, &req)
	if err != nil {
		respondWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, cat)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := h.service.Delete(id, userID); err != nil {
		respondWithError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func respondWithError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, gin.H{"message": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
}
