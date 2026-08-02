package expense

import (
	"time"

	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateRequest is the payload for creating an expense.
type CreateRequest struct {
	CategoryID  *uuid.UUID             `json:"category_id"`
	Amount      decimal.Decimal        `json:"amount" binding:"required"`
	Type        domain.TransactionType `json:"type" binding:"required,oneof=expense income"`
	Description string                 `json:"description" binding:"required,min=1,max=255"`
	Date        string                 `json:"date" binding:"required"`
	Tags        string                 `json:"tags"`
}

// UpdateRequest is the payload for updating an expense.
type UpdateRequest struct {
	CategoryID  *uuid.UUID             `json:"category_id"`
	Amount      *decimal.Decimal       `json:"amount"`
	Type        domain.TransactionType `json:"type" binding:"omitempty,oneof=expense income"`
	Description string                 `json:"description" binding:"omitempty,min=1,max=255"`
	Date        string                 `json:"date"`
	Tags        string                 `json:"tags"`
}

// UncategorizedFilter is the category_id value that selects transactions with
// no category. It is not a valid UUID, so it cannot collide with a real id.
const UncategorizedFilter = "none"

// ListRequest holds pagination and filter params.
type ListRequest struct {
	// CategoryID is bound as a string because gin cannot map a query param onto
	// uuid.UUID (it is a [16]byte array); it is parsed in List.
	CategoryID string                  `form:"category_id"`
	Type       *domain.TransactionType `form:"type"`
	StartDate  string                  `form:"start_date"`
	EndDate    string                  `form:"end_date"`
	Page       int                     `form:"page,default=1"`
	PageSize   int                     `form:"page_size,default=20"`
}

// PaginatedResponse wraps paginated results.
type PaginatedResponse struct {
	Data     []*domain.Expense `json:"data"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// Service defines the business logic interface for expenses.
type Service interface {
	Create(userID uuid.UUID, req *CreateRequest) (*domain.Expense, error)
	GetByID(id, userID uuid.UUID) (*domain.Expense, error)
	List(userID uuid.UUID, req ListRequest) (*PaginatedResponse, error)
	Update(id, userID uuid.UUID, req *UpdateRequest) (*domain.Expense, error)
	Delete(id, userID uuid.UUID) error
	GetMonthlySummary(userID uuid.UUID, months int) ([]*MonthlySummary, error)
	GetCategorySummary(userID uuid.UUID, start, end time.Time) ([]*CategorySummary, error)
}

type service struct {
	repo Repository
}

// NewService creates a new expense service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(userID uuid.UUID, req *CreateRequest) (*domain.Expense, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, apperrors.New(400, "amount must be greater than zero")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, apperrors.New(400, "invalid date format, use YYYY-MM-DD")
	}

	expense := &domain.Expense{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Description,
		Date:        date,
		Tags:        req.Tags,
	}

	if err := s.repo.Create(expense); err != nil {
		return nil, apperrors.Wrap(500, "failed to create expense", err)
	}

	return s.repo.FindByID(expense.ID, userID)
}

func (s *service) GetByID(id, userID uuid.UUID) (*domain.Expense, error) {
	return s.repo.FindByID(id, userID)
}

func (s *service) List(userID uuid.UUID, req ListRequest) (*PaginatedResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	filter := ListFilter{
		UserID:   userID,
		Type:     req.Type,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	switch {
	case req.CategoryID == UncategorizedFilter:
		filter.Uncategorized = true
	case req.CategoryID != "":
		categoryID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return nil, apperrors.New(400, "invalid category_id")
		}
		filter.CategoryID = &categoryID
	}

	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			filter.StartDate = &t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			filter.EndDate = &t
		}
	}

	expenses, total, err := s.repo.List(filter)
	if err != nil {
		return nil, apperrors.Wrap(500, "failed to list expenses", err)
	}

	return &PaginatedResponse{
		Data:     expenses,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *service) Update(id, userID uuid.UUID, req *UpdateRequest) (*domain.Expense, error) {
	expense, err := s.repo.FindByID(id, userID)
	if err != nil {
		return nil, err
	}

	if req.Amount != nil {
		if req.Amount.LessThanOrEqual(decimal.Zero) {
			return nil, apperrors.New(400, "amount must be greater than zero")
		}
		expense.Amount = *req.Amount
	}
	if req.Type != "" {
		expense.Type = req.Type
	}
	if req.Description != "" {
		expense.Description = req.Description
	}
	if req.Date != "" {
		if d, err := time.Parse("2006-01-02", req.Date); err == nil {
			expense.Date = d
		}
	}
	if req.CategoryID != nil {
		expense.CategoryID = req.CategoryID
	}
	expense.Tags = req.Tags

	if err := s.repo.Update(expense); err != nil {
		return nil, apperrors.Wrap(500, "failed to update expense", err)
	}
	return expense, nil
}

func (s *service) Delete(id, userID uuid.UUID) error {
	return s.repo.Delete(id, userID)
}

func (s *service) GetMonthlySummary(userID uuid.UUID, months int) ([]*MonthlySummary, error) {
	if months < 1 || months > 24 {
		months = 12
	}
	return s.repo.MonthlySummary(userID, months)
}

func (s *service) GetCategorySummary(userID uuid.UUID, start, end time.Time) ([]*CategorySummary, error) {
	return s.repo.CategorySummary(userID, start, end)
}
