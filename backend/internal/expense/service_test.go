package expense_test

import (
	"errors"
	"github.com/financeapp/backend/pkg/logger"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/expense"
	expmocks "github.com/financeapp/backend/internal/testutils/mocks/expense"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDB = errors.New("connection reset")

func newService(t *testing.T) (expense.Service, *expmocks.MockRepository) {
	t.Helper()
	repo := expmocks.NewMockRepository(gomock.NewController(t))
	return expense.NewService(repo, logger.Discard()), repo
}

func requireAppError(t *testing.T, err error) *apperrors.AppError {
	t.Helper()
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr, "expected a status-carrying error")
	return appErr
}

// captureFilter records the ListFilter the service builds and returns an empty page.
func captureFilter(repo *expmocks.MockRepository, into *expense.ListFilter) {
	repo.EXPECT().List(gomock.Any()).DoAndReturn(
		func(f expense.ListFilter) ([]*domain.Expense, int64, error) {
			*into = f
			return nil, 0, nil
		})
}

// A category_id in the query string must bind. uuid.UUID is a [16]byte array,
// which gin's form binder rejects outright, so ListRequest carries it as a
// string: binding it as *uuid.UUID made every filtered request fail with 400.
func TestListRequest_BindsCategoryIDFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet,
		"/expenses?category_id=6ba7b810-9dad-11d1-80b4-00c04fd430c8&type=expense&start_date=2026-09-01", nil)

	var req expense.ListRequest
	require.NoError(t, c.ShouldBindQuery(&req), "binding must not fail")
	assert.Equal(t, "6ba7b810-9dad-11d1-80b4-00c04fd430c8", req.CategoryID)
	require.NotNil(t, req.Type)
	assert.Equal(t, domain.TransactionTypeExpense, *req.Type)
}

func TestList_ForwardsCategoryIDToRepository(t *testing.T) {
	svc, repo := newService(t)
	categoryID := uuid.New()
	var got expense.ListFilter
	captureFilter(repo, &got)

	_, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: categoryID.String()})
	require.NoError(t, err)

	require.NotNil(t, got.CategoryID, "expenses were not filtered by category")
	assert.Equal(t, categoryID, *got.CategoryID)
	assert.False(t, got.Uncategorized)
}

func TestList_WithoutCategoryIDDoesNotFilter(t *testing.T) {
	svc, repo := newService(t)
	var got expense.ListFilter
	captureFilter(repo, &got)

	_, err := svc.List(uuid.New(), expense.ListRequest{})
	require.NoError(t, err)
	assert.Nil(t, got.CategoryID)
	assert.False(t, got.Uncategorized)
}

func TestList_RejectsMalformedCategoryID(t *testing.T) {
	svc, _ := newService(t)
	// No List expectation: a bad id must never reach the repository.

	_, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: "not-a-uuid"})
	assert.Equal(t, http.StatusBadRequest, requireAppError(t, err).Code)
}

func TestList_UncategorizedFilterSelectsNullCategory(t *testing.T) {
	svc, repo := newService(t)
	var got expense.ListFilter
	captureFilter(repo, &got)

	_, err := svc.List(uuid.New(), expense.ListRequest{CategoryID: expense.UncategorizedFilter})
	require.NoError(t, err, "the sentinel must not be mistaken for a malformed uuid")
	assert.True(t, got.Uncategorized)
	assert.Nil(t, got.CategoryID)
}

func TestList_ForwardsTypeAndDateRange(t *testing.T) {
	svc, repo := newService(t)
	var got expense.ListFilter
	captureFilter(repo, &got)

	income := domain.TransactionTypeIncome
	_, err := svc.List(uuid.New(), expense.ListRequest{
		Type: &income, StartDate: "2026-09-01", EndDate: "2026-09-30",
	})
	require.NoError(t, err)

	require.NotNil(t, got.Type)
	assert.Equal(t, income, *got.Type)
	require.NotNil(t, got.StartDate)
	require.NotNil(t, got.EndDate)
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), *got.StartDate)
	assert.Equal(t, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), *got.EndDate)
}

// An unparseable date is ignored rather than rejected, so the list still renders.
func TestList_IgnoresUnparseableDates(t *testing.T) {
	svc, repo := newService(t)
	var got expense.ListFilter
	captureFilter(repo, &got)

	_, err := svc.List(uuid.New(), expense.ListRequest{StartDate: "31/12/2026", EndDate: ""})
	require.NoError(t, err)
	assert.Nil(t, got.StartDate)
	assert.Nil(t, got.EndDate)
}

func TestList_ScopesToTheCaller(t *testing.T) {
	svc, repo := newService(t)
	userID := uuid.New()
	var got expense.ListFilter
	captureFilter(repo, &got)

	_, err := svc.List(userID, expense.ListRequest{})
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
}

func TestList_ClampsPagination(t *testing.T) {
	cases := []struct {
		name               string
		page, size         int
		wantPage, wantSize int
	}{
		{"defaults when zero", 0, 0, 1, 20},
		{"negative page", -5, 50, 1, 50},
		{"page size above the cap", 1, 500, 1, 20},
		{"negative page size", 1, -1, 1, 20},
		{"values within range are kept", 3, 75, 3, 75},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newService(t)
			var got expense.ListFilter
			captureFilter(repo, &got)

			resp, err := svc.List(uuid.New(), expense.ListRequest{Page: tc.page, PageSize: tc.size})
			require.NoError(t, err)

			assert.Equal(t, tc.wantPage, got.Page, "filter page")
			assert.Equal(t, tc.wantSize, got.PageSize, "filter page size")
			assert.Equal(t, tc.wantPage, resp.Page, "response must echo the clamped page")
			assert.Equal(t, tc.wantSize, resp.PageSize, "response must echo the clamped page size")
		})
	}
}

func TestList_ReturnsTotalFromRepository(t *testing.T) {
	svc, repo := newService(t)
	rows := []*domain.Expense{{Description: "a"}, {Description: "b"}}
	repo.EXPECT().List(gomock.Any()).Return(rows, int64(57), nil)

	resp, err := svc.List(uuid.New(), expense.ListRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.EqualValues(t, 57, resp.Total, "total must be the full count, not the page length")
}

func TestList_WrapsRepositoryFailureAs500(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().List(gomock.Any()).Return(nil, int64(0), errDB)

	_, err := svc.List(uuid.New(), expense.ListRequest{})
	assert.Equal(t, http.StatusInternalServerError, requireAppError(t, err).Code)
	assert.ErrorIs(t, err, errDB)
}

func validCreate() *expense.CreateRequest {
	return &expense.CreateRequest{
		Amount:      decimal.NewFromFloat(42.50),
		Type:        domain.TransactionTypeExpense,
		Description: "Compras",
		Date:        "2026-09-10",
	}
}

func TestCreate_PersistsAndReloadsWithCategory(t *testing.T) {
	svc, repo := newService(t)
	userID, newID := uuid.New(), uuid.New()
	var saved *domain.Expense

	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(e *domain.Expense) error {
		saved = e
		e.ID = newID
		return nil
	})
	// Reloaded so the response carries the preloaded Category.
	repo.EXPECT().FindByID(newID, userID).Return(&domain.Expense{ID: newID}, nil)

	got, err := svc.Create(userID, validCreate())
	require.NoError(t, err)

	assert.Equal(t, userID, saved.UserID)
	assert.Equal(t, "Compras", saved.Description)
	assert.Equal(t, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), saved.Date)
	assert.Equal(t, newID, got.ID)
}

func TestCreate_RejectsNonPositiveAmounts(t *testing.T) {
	for _, amount := range []string{"0", "-0.01", "-100"} {
		t.Run(amount, func(t *testing.T) {
			svc, _ := newService(t) // no Create expectation
			req := validCreate()
			req.Amount = decimal.RequireFromString(amount)

			_, err := svc.Create(uuid.New(), req)
			assert.Equal(t, http.StatusBadRequest, requireAppError(t, err).Code)
		})
	}
}

func TestCreate_RejectsBadDateFormat(t *testing.T) {
	svc, _ := newService(t)
	req := validCreate()
	req.Date = "10/09/2026"

	_, err := svc.Create(uuid.New(), req)
	appErr := requireAppError(t, err)
	assert.Equal(t, http.StatusBadRequest, appErr.Code)
	assert.Contains(t, appErr.Message, "YYYY-MM-DD", "the message should say what format is expected")
}

func TestCreate_WrapsRepositoryFailureAs500(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().Create(gomock.Any()).Return(errDB)

	_, err := svc.Create(uuid.New(), validCreate())
	assert.Equal(t, http.StatusInternalServerError, requireAppError(t, err).Code)
}

func TestUpdate_AppliesOnlyTheProvidedFields(t *testing.T) {
	svc, repo := newService(t)
	id, userID := uuid.New(), uuid.New()
	existing := &domain.Expense{
		ID: id, UserID: userID,
		Amount:      decimal.NewFromInt(100),
		Type:        domain.TransactionTypeExpense,
		Description: "Original",
		Date:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	repo.EXPECT().FindByID(id, userID).Return(existing, nil)
	repo.EXPECT().Update(existing).Return(nil)

	newAmount := decimal.NewFromInt(250)
	got, err := svc.Update(id, userID, &expense.UpdateRequest{Amount: &newAmount})
	require.NoError(t, err)

	assert.True(t, got.Amount.Equal(newAmount), "amount should change")
	assert.Equal(t, "Original", got.Description, "an empty description must not blank the field")
	assert.Equal(t, domain.TransactionTypeExpense, got.Type, "an empty type must not blank the field")
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), got.Date, "an empty date must not move it")
}

func TestUpdate_RejectsNonPositiveAmount(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(&domain.Expense{}, nil)
	// No Update expectation: the write must not happen.

	zero := decimal.Zero
	_, err := svc.Update(uuid.New(), uuid.New(), &expense.UpdateRequest{Amount: &zero})
	assert.Equal(t, http.StatusBadRequest, requireAppError(t, err).Code)
}

func TestUpdate_UnknownExpenseDoesNotWrite(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apperrors.ErrNotFound)

	_, err := svc.Update(uuid.New(), uuid.New(), &expense.UpdateRequest{Description: "X"})
	assert.Equal(t, http.StatusNotFound, requireAppError(t, err).Code)
}

func TestUpdate_MovesExpenseToAnotherCategory(t *testing.T) {
	svc, repo := newService(t)
	existing := &domain.Expense{}
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(existing, nil)
	repo.EXPECT().Update(existing).Return(nil)

	newCategory := uuid.New()
	got, err := svc.Update(uuid.New(), uuid.New(), &expense.UpdateRequest{CategoryID: &newCategory})
	require.NoError(t, err)
	require.NotNil(t, got.CategoryID)
	assert.Equal(t, newCategory, *got.CategoryID)
}

func TestGetByID_ForwardsBothIDs(t *testing.T) {
	svc, repo := newService(t)
	id, userID := uuid.New(), uuid.New()
	repo.EXPECT().FindByID(id, userID).Return(&domain.Expense{ID: id}, nil)

	got, err := svc.GetByID(id, userID)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestDelete_PropagatesNotFound(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(apperrors.ErrNotFound)

	err := svc.Delete(uuid.New(), uuid.New())
	assert.Equal(t, http.StatusNotFound, requireAppError(t, err).Code)
}

func TestGetMonthlySummary_ClampsTheWindow(t *testing.T) {
	cases := map[string]struct{ in, want int }{
		"zero falls back":      {0, 12},
		"negative falls back":  {-3, 12},
		"above the cap":        {99, 12},
		"within range is kept": {6, 6},
		"upper bound is kept":  {24, 24},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc, repo := newService(t)
			repo.EXPECT().MonthlySummary(gomock.Any(), tc.want).Return(nil, nil)

			_, err := svc.GetMonthlySummary(uuid.New(), tc.in)
			require.NoError(t, err)
		})
	}
}

func TestGetCategorySummary_ForwardsTheRange(t *testing.T) {
	svc, repo := newService(t)
	userID := uuid.New()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	want := []*expense.CategorySummary{{CategoryName: "Mercado", Total: 120}}
	repo.EXPECT().CategorySummary(userID, start, end).Return(want, nil)

	got, err := svc.GetCategorySummary(userID, start, end)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
