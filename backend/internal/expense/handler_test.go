package expense_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/testutils"
	expmocks "github.com/financeapp/backend/internal/testutils/mocks/expense"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newHandler(t *testing.T) (*expense.Handler, *expmocks.MockService) {
	t.Helper()
	svc := expmocks.NewMockService(gomock.NewController(t))
	return expense.NewHandler(svc), svc
}

func validCreateBody() map[string]any {
	return map[string]any{
		"amount": 42.5, "type": "expense", "description": "Compras", "date": "2026-09-10",
	}
}

func TestHandlerCreate_Created(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()
	svc.EXPECT().Create(gomock.Any(), userID, gomock.Any()).Return(&domain.Expense{Description: "Compras"}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/expenses", UserID: &userID, Body: validCreateBody(),
	})
	h.Create(c)

	testutils.AssertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, "Compras", testutils.DecodeBody[domain.Expense](t, rec).Description)
}

func TestHandlerCreate_RejectsInvalidPayload(t *testing.T) {
	cases := map[string]any{
		"missing description": map[string]any{"amount": 10, "type": "expense", "date": "2026-09-10"},
		"missing date":        map[string]any{"amount": 10, "type": "expense", "description": "x"},
		"missing type":        map[string]any{"amount": 10, "description": "x", "date": "2026-09-10"},
		"unknown type":        map[string]any{"amount": 10, "type": "refund", "description": "x", "date": "2026-09-10"},
		"malformed json":      `{"amount":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			// No expectation is registered, so gomock fails the test if the
			// handler forwards the request instead of rejecting it.
			h, _ := newHandler(t)
			c, rec := testutils.NewContext(t, testutils.Request{
				Method: http.MethodPost, Target: "/expenses", Body: body,
			})
			h.Create(c)
			testutils.AssertStatus(t, rec, http.StatusBadRequest)
		})
	}
}

// `binding:"required"` has no effect on Amount: decimal.Decimal is a struct, and
// the validator does not treat its zero value as missing. Both a missing and a
// zero amount therefore reach the service, which is what actually rejects them.
// The request is still refused, just one layer later than the tag suggests.
func TestHandlerCreate_AmountIsValidatedByTheServiceNotTheBinder(t *testing.T) {
	cases := map[string]map[string]any{
		"missing amount": {"type": "expense", "description": "x", "date": "2026-09-10"},
		"zero amount":    {"amount": 0, "type": "expense", "description": "x", "date": "2026-09-10"},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			h, svc := newHandler(t)
			svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, req *expense.CreateRequest) (*domain.Expense, error) {
					assert.True(t, req.Amount.IsZero(), "the binder let a zero amount through")
					return nil, apperrors.New(http.StatusBadRequest, "amount must be greater than zero")
				})

			c, rec := testutils.NewContext(t, testutils.Request{
				Method: http.MethodPost, Target: "/expenses", Body: body,
			})
			h.Create(c)

			testutils.AssertStatus(t, rec, http.StatusBadRequest)
			assert.Equal(t, "amount must be greater than zero", testutils.Message(t, rec))
		})
	}
}

func TestHandlerCreate_MapsServiceError(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apperrors.New(http.StatusBadRequest, "amount must be greater than zero"))

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/expenses", Body: validCreateBody(),
	})
	h.Create(c)

	testutils.AssertStatus(t, rec, http.StatusBadRequest)
	assert.Equal(t, "amount must be greater than zero", testutils.Message(t, rec))
}

func TestHandlerList_PassesQueryFiltersThrough(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()
	categoryID := uuid.New()

	svc.EXPECT().List(gomock.Any(), userID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, req expense.ListRequest) (*expense.PaginatedResponse, error) {
			assert.Equal(t, categoryID.String(), req.CategoryID)
			assert.Equal(t, "2026-09-01", req.StartDate)
			assert.Equal(t, "2026-09-30", req.EndDate)
			assert.Equal(t, 2, req.Page)
			assert.Equal(t, 50, req.PageSize)
			require.NotNil(t, req.Type)
			assert.Equal(t, domain.TransactionTypeExpense, *req.Type)
			return &expense.PaginatedResponse{Total: 0}, nil
		})

	c, rec := testutils.NewContext(t, testutils.Request{
		Target: "/expenses?category_id=" + categoryID.String() +
			"&type=expense&start_date=2026-09-01&end_date=2026-09-30&page=2&page_size=50",
		UserID: &userID,
	})
	h.List(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

// The "no category" filter travels as a sentinel that is not a valid uuid.
func TestHandlerList_AcceptsUncategorizedSentinel(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, req expense.ListRequest) (*expense.PaginatedResponse, error) {
			assert.Equal(t, expense.UncategorizedFilter, req.CategoryID)
			return &expense.PaginatedResponse{}, nil
		})

	c, rec := testutils.NewContext(t, testutils.Request{
		Target: "/expenses?category_id=" + expense.UncategorizedFilter,
	})
	h.List(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

func TestHandlerList_RejectsUnbindableQuery(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{Target: "/expenses?page=abc"})
	h.List(c)
	testutils.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlerList_MapsServiceError(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apperrors.New(http.StatusBadRequest, "invalid category_id"))

	c, rec := testutils.NewContext(t, testutils.Request{Target: "/expenses?category_id=nope"})
	h.List(c)

	testutils.AssertStatus(t, rec, http.StatusBadRequest)
	assert.Equal(t, "invalid category_id", testutils.Message(t, rec))
}

func TestHandlerGetByID_OK(t *testing.T) {
	h, svc := newHandler(t)
	id, userID := uuid.New(), uuid.New()
	svc.EXPECT().GetByID(gomock.Any(), id, userID).Return(&domain.Expense{ID: id}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Target: "/expenses/" + id.String(), UserID: &userID,
		Params: map[string]string{"id": id.String()},
	})
	h.GetByID(c)

	testutils.AssertStatus(t, rec, http.StatusOK)
	assert.Equal(t, id, testutils.DecodeBody[domain.Expense](t, rec).ID)
}

func TestHandler_MalformedIDIsRejectedBeforeTheService(t *testing.T) {
	// Every id-bearing route must answer 400 without calling the service.
	cases := map[string]struct {
		method string
		call   func(h *expense.Handler, c *gin.Context)
	}{
		"get":    {http.MethodGet, func(h *expense.Handler, c *gin.Context) { h.GetByID(c) }},
		"update": {http.MethodPut, func(h *expense.Handler, c *gin.Context) { h.Update(c) }},
		"delete": {http.MethodDelete, func(h *expense.Handler, c *gin.Context) { h.Delete(c) }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h, _ := newHandler(t)
			c, rec := testutils.NewContext(t, testutils.Request{
				Method: tc.method, Target: "/expenses/abc",
				Params: map[string]string{"id": "not-a-uuid"},
				Body:   map[string]any{"description": "x"},
			})
			tc.call(h, c)

			testutils.AssertStatus(t, rec, http.StatusBadRequest)
			assert.Equal(t, "invalid id", testutils.Message(t, rec))
		})
	}
}

func TestHandlerUpdate_OK(t *testing.T) {
	h, svc := newHandler(t)
	id, userID := uuid.New(), uuid.New()
	svc.EXPECT().Update(gomock.Any(), id, userID, gomock.Any()).Return(&domain.Expense{ID: id}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPut, Target: "/expenses/" + id.String(), UserID: &userID,
		Params: map[string]string{"id": id.String()},
		Body:   map[string]any{"description": "Atualizado"},
	})
	h.Update(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

func TestHandlerUpdate_RejectsUnknownType(t *testing.T) {
	h, _ := newHandler(t)
	id := uuid.New()
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPut, Target: "/expenses/" + id.String(),
		Params: map[string]string{"id": id.String()},
		Body:   map[string]any{"type": "refund"},
	})
	h.Update(c)
	testutils.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlerDelete_NoContent(t *testing.T) {
	h, svc := newHandler(t)
	id, userID := uuid.New(), uuid.New()
	svc.EXPECT().Delete(gomock.Any(), id, userID).Return(nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/expenses/" + id.String(), UserID: &userID,
		Params: map[string]string{"id": id.String()},
	})
	h.Delete(c)

	testutils.AssertStatus(t, rec, http.StatusNoContent)
	assert.Zero(t, rec.Body.Len(), "204 must have an empty body")
}

func TestHandlerDelete_NotFound(t *testing.T) {
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Delete(gomock.Any(), id, gomock.Any()).Return(apperrors.ErrNotFound)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/expenses/" + id.String(),
		Params: map[string]string{"id": id.String()},
	})
	h.Delete(c)
	testutils.AssertStatus(t, rec, http.StatusNotFound)
}

// BUG: the handler binds ?months into a throwaway anonymous struct and never
// reads the result, so the local `months` stays 0 and is always replaced by 12.
// This test pins the behaviour as it is today; fixing the handler should make it
// fail, which is the signal to update it.
func TestHandlerMonthlySummary_IgnoresTheMonthsQueryParam(t *testing.T) {
	for _, query := range []string{"", "?months=3", "?months=24", "?months=garbage"} {
		t.Run("query="+query, func(t *testing.T) {
			h, svc := newHandler(t)
			svc.EXPECT().GetMonthlySummary(gomock.Any(), gomock.Any(), 12).Return(nil, nil)

			c, rec := testutils.NewContext(t, testutils.Request{Target: "/reports/monthly" + query})
			h.MonthlySummary(c)
			testutils.AssertStatus(t, rec, http.StatusOK)
		})
	}
}

func TestHandlerCategorySummary_ParsesTheDateRange(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()
	svc.EXPECT().GetCategorySummary(gomock.Any(), userID,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	).Return(nil, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Target: "/reports/categories?start_date=2026-03-01&end_date=2026-03-31", UserID: &userID,
	})
	h.CategorySummary(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

// Without a range the handler falls back to the current month.
func TestHandlerCategorySummary_DefaultsToCurrentMonth(t *testing.T) {
	h, svc := newHandler(t)
	now := time.Now()
	wantStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	svc.EXPECT().GetCategorySummary(gomock.Any(), gomock.Any(), wantStart, wantStart.AddDate(0, 1, -1)).Return(nil, nil)

	c, rec := testutils.NewContext(t, testutils.Request{Target: "/reports/categories"})
	h.CategorySummary(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

// An unparseable date is ignored, keeping that side of the default range.
func TestHandlerCategorySummary_IgnoresUnparseableDates(t *testing.T) {
	h, svc := newHandler(t)
	now := time.Now()
	wantStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	svc.EXPECT().GetCategorySummary(gomock.Any(), gomock.Any(), wantStart, gomock.Any()).Return(nil, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Target: "/reports/categories?start_date=01-03-2026",
	})
	h.CategorySummary(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

// The handler must hand the request's own context to the service, so a client
// that gives up cancels the work downstream instead of leaving the query running.
func TestHandlerList_PassesTheRequestContextDownstream(t *testing.T) {
	h, svc := newHandler(t)

	var seen context.Context
	svc.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ uuid.UUID, _ expense.ListRequest) (*expense.PaginatedResponse, error) {
			seen = ctx
			return &expense.PaginatedResponse{}, nil
		})

	c, _ := testutils.NewContext(t, testutils.Request{Target: "/expenses"})
	ctx, cancel := context.WithCancel(c.Request.Context())
	c.Request = c.Request.WithContext(ctx)

	h.List(c)

	require.NotNil(t, seen, "the service never received a context")
	require.NoError(t, seen.Err(), "the context should still be live during the call")

	// Cancelling the request must be visible to whatever the service kept.
	cancel()
	assert.ErrorIs(t, seen.Err(), context.Canceled,
		"the service got a detached context; cancellation would not reach the database")
}
