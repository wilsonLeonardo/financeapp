package expense_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/expense"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Stub repository that records the filter it was called with ────────────────

type stubRepo struct {
	lastFilter expense.ListFilter
}

func (r *stubRepo) List(f expense.ListFilter) ([]*domain.Expense, int64, error) {
	r.lastFilter = f
	return nil, 0, nil
}

func (r *stubRepo) Create(*domain.Expense) error                     { return nil }
func (r *stubRepo) FindByID(_, _ uuid.UUID) (*domain.Expense, error) { return nil, nil }
func (r *stubRepo) Update(*domain.Expense) error                     { return nil }
func (r *stubRepo) Delete(_, _ uuid.UUID) error                      { return nil }
func (r *stubRepo) DeleteByImportID(string) error                    { return nil }
func (r *stubRepo) CreateBatch([]*domain.Expense) error              { return nil }
func (r *stubRepo) MonthlySummary(uuid.UUID, int) ([]*expense.MonthlySummary, error) {
	return nil, nil
}
func (r *stubRepo) CategorySummary(uuid.UUID, time.Time, time.Time) ([]*expense.CategorySummary, error) {
	return nil, nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// A category_id in the query string must bind. uuid.UUID is a [16]byte array,
// which gin's form binder rejects outright, so ListRequest carries it as a
// string: binding it as *uuid.UUID made every filtered request fail with 400.
func TestListRequest_BindsCategoryIDFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet,
		"/expenses?category_id=6ba7b810-9dad-11d1-80b4-00c04fd430c8&type=expense&start_date=2026-09-01", nil)

	var req expense.ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		t.Fatalf("binding failed: %v", err)
	}
	if req.CategoryID != "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		t.Errorf("CategoryID = %q, want the uuid from the query", req.CategoryID)
	}
	if req.Type == nil || *req.Type != domain.TransactionTypeExpense {
		t.Errorf("Type = %v, want expense", req.Type)
	}
}

func TestList_ForwardsCategoryIDToRepository(t *testing.T) {
	repo := &stubRepo{}
	svc := expense.NewService(repo)
	categoryID := uuid.New()

	if _, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: categoryID.String()}); err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if repo.lastFilter.CategoryID == nil {
		t.Fatal("filter.CategoryID is nil, expenses were not filtered by category")
	}
	if *repo.lastFilter.CategoryID != categoryID {
		t.Errorf("filter.CategoryID = %v, want %v", *repo.lastFilter.CategoryID, categoryID)
	}
}

func TestList_WithoutCategoryIDDoesNotFilter(t *testing.T) {
	repo := &stubRepo{}
	svc := expense.NewService(repo)

	if _, err := svc.List(uuid.New(), expense.ListRequest{}); err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if repo.lastFilter.CategoryID != nil {
		t.Errorf("filter.CategoryID = %v, want nil", repo.lastFilter.CategoryID)
	}
}

func TestList_RejectsMalformedCategoryID(t *testing.T) {
	svc := expense.NewService(&stubRepo{})

	_, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: "not-a-uuid"})
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("err = %v, want *apperrors.AppError", err)
	}
	if appErr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", appErr.Code, http.StatusBadRequest)
	}
}

func TestList_UncategorizedFilterSelectsNullCategory(t *testing.T) {
	repo := &stubRepo{}
	svc := expense.NewService(repo)

	if _, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: expense.UncategorizedFilter}); err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if !repo.lastFilter.Uncategorized {
		t.Error("filter.Uncategorized = false, want true")
	}
	if repo.lastFilter.CategoryID != nil {
		t.Errorf("filter.CategoryID = %v, want nil", repo.lastFilter.CategoryID)
	}
}

// The sentinel must never be mistaken for a malformed uuid.
func TestList_UncategorizedFilterIsNotRejected(t *testing.T) {
	svc := expense.NewService(&stubRepo{})

	if _, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: expense.UncategorizedFilter}); err != nil {
		t.Fatalf("List rejected the uncategorized sentinel: %v", err)
	}
}
