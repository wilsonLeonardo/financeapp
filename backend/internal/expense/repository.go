package expense

import (
	"time"

	"github.com/financeapp/backend/internal/domain"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListFilter holds query filters for listing expenses.
type ListFilter struct {
	UserID     uuid.UUID
	CategoryID *uuid.UUID
	Type       *domain.TransactionType
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

// MonthlySummary holds aggregated data for a month.
type MonthlySummary struct {
	Month       string  `json:"month"`
	TotalSpent  float64 `json:"total_spent"`
	TotalEarned float64 `json:"total_earned"`
}

// CategorySummary holds aggregated data per category.
type CategorySummary struct {
	CategoryID   *uuid.UUID `json:"category_id"`
	CategoryName string     `json:"category_name"`
	Total        float64    `json:"total"`
	Count        int        `json:"count"`
}

// Repository defines the persistence interface for expenses.
type Repository interface {
	Create(expense *domain.Expense) error
	FindByID(id, userID uuid.UUID) (*domain.Expense, error)
	List(filter ListFilter) ([]*domain.Expense, int64, error)
	Update(expense *domain.Expense) error
	Delete(id, userID uuid.UUID) error
	DeleteByImportID(importID string) error
	MonthlySummary(userID uuid.UUID, months int) ([]*MonthlySummary, error)
	CategorySummary(userID uuid.UUID, start, end time.Time) ([]*CategorySummary, error)
	CreateBatch(expenses []*domain.Expense) error
}

type postgresRepository struct {
	db *gorm.DB
}

// NewRepository creates a new expense repository.
func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(expense *domain.Expense) error {
	return r.db.Create(expense).Error
}

func (r *postgresRepository) FindByID(id, userID uuid.UUID) (*domain.Expense, error) {
	var expense domain.Expense
	err := r.db.Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&expense).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &expense, nil
}

func (r *postgresRepository) List(f ListFilter) ([]*domain.Expense, int64, error) {
	query := r.db.Model(&domain.Expense{}).
		Preload("Category").
		Where("user_id = ?", f.UserID)

	if f.CategoryID != nil {
		query = query.Where("category_id = ?", *f.CategoryID)
	}
	if f.Type != nil {
		query = query.Where("type = ?", *f.Type)
	}
	if f.StartDate != nil {
		query = query.Where("date >= ?", *f.StartDate)
	}
	if f.EndDate != nil {
		query = query.Where("date <= ?", *f.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PageSize
	var expenses []*domain.Expense
	err := query.Order("date DESC").
		Limit(f.PageSize).
		Offset(offset).
		Find(&expenses).Error

	return expenses, total, err
}

func (r *postgresRepository) Update(expense *domain.Expense) error {
	return r.db.Save(expense).Error
}

func (r *postgresRepository) Delete(id, userID uuid.UUID) error {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&domain.Expense{})
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	return result.Error
}

func (r *postgresRepository) MonthlySummary(userID uuid.UUID, months int) ([]*MonthlySummary, error) {
	var results []*MonthlySummary
	err := r.db.Raw(`
		SELECT
			TO_CHAR(date, 'YYYY-MM') AS month,
			SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) AS total_spent,
			SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END) AS total_earned
		FROM expenses
		WHERE user_id = ?
			AND date >= NOW() - INTERVAL '1 month' * ?
		GROUP BY TO_CHAR(date, 'YYYY-MM')
		ORDER BY month ASC
	`, userID, months).Scan(&results).Error
	return results, err
}

func (r *postgresRepository) CategorySummary(userID uuid.UUID, start, end time.Time) ([]*CategorySummary, error) {
	var results []*CategorySummary
	err := r.db.Raw(`
		SELECT
			e.category_id,
			COALESCE(c.name, 'Uncategorized') AS category_name,
			SUM(e.amount) AS total,
			COUNT(*) AS count
		FROM expenses e
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE e.user_id = ?
			AND e.type = 'expense'
			AND e.date BETWEEN ? AND ?
		GROUP BY e.category_id, c.name
		ORDER BY total DESC
	`, userID, start, end).Scan(&results).Error
	return results, err
}

func (r *postgresRepository) DeleteByImportID(importID string) error {
	// import_id is stored as "csv-YYYYMMDD-N", we match the import UUID prefix
	return r.db.Where("import_id LIKE ?", importID+"%").Delete(&domain.Expense{}).Error
}

func (r *postgresRepository) CreateBatch(expenses []*domain.Expense) error {
	return r.db.CreateInBatches(expenses, 100).Error
}
