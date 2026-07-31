package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// User represents an application user.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Category represents an expense category.
type Category struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TransactionType defines the direction of a transaction.
type TransactionType string

const (
	TransactionTypeExpense TransactionType = "expense"
	TransactionTypeIncome  TransactionType = "income"
)

// Expense represents a financial transaction.
type Expense struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	CategoryID  *uuid.UUID      `gorm:"type:uuid;index" json:"category_id,omitempty"`
	Category    *Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Amount      decimal.Decimal `gorm:"type:numeric(15,2);not null" json:"amount"`
	Type        TransactionType `gorm:"type:varchar(10);not null" json:"type"`
	Description string          `gorm:"not null" json:"description"`
	Date        time.Time       `gorm:"not null;index" json:"date"`
	ImportID    *string         `gorm:"index" json:"import_id,omitempty"`
	Tags        string          `json:"tags"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Import represents a bank statement import job.
type Import struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	FileName   string     `gorm:"not null" json:"file_name"`
	FileType   string     `gorm:"not null" json:"file_type"`
	Status     string     `gorm:"not null;default:'pending'" json:"status"`
	TotalRows  int        `json:"total_rows"`
	Imported   int        `json:"imported"`
	Errors     int        `json:"errors"`
	ErrorLog   string     `gorm:"type:text" json:"error_log,omitempty"`
	ImportedAt *time.Time `json:"imported_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// BeforeCreate hooks to set UUIDs before inserting.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (e *Expense) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (i *Import) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
