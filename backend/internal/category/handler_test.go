package category_test

import (
	"net/http"
	"testing"

	"github.com/financeapp/backend/internal/category"
	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/testutils"
	catmocks "github.com/financeapp/backend/internal/testutils/mocks/category"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newHandler(t *testing.T) (*category.Handler, *catmocks.MockService) {
	t.Helper()
	svc := catmocks.NewMockService(gomock.NewController(t))
	return category.NewHandler(svc), svc
}

func TestHandlerCreate_Created(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()
	svc.EXPECT().Create(gomock.Any(), userID, gomock.Any()).Return(&domain.Category{Name: "Mercado"}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/categories", UserID: &userID,
		Body: map[string]string{"name": "Mercado", "icon": "🛒", "color": "#22c55e"},
	})
	h.Create(c)

	testutils.AssertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, "Mercado", testutils.DecodeBody[domain.Category](t, rec).Name)
}

// The handler must pass the caller's id, not one taken from the payload.
func TestHandlerCreate_UsesTheAuthenticatedUser(t *testing.T) {
	h, svc := newHandler(t)
	authenticated := uuid.New()
	svc.EXPECT().Create(gomock.Any(), authenticated, gomock.Any()).Return(&domain.Category{}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/categories", UserID: &authenticated,
		Body: map[string]any{"name": "Mercado", "user_id": uuid.New().String()},
	})
	h.Create(c)
	testutils.AssertStatus(t, rec, http.StatusCreated)
}

func TestHandlerCreate_RejectsInvalidPayload(t *testing.T) {
	h, _ := newHandler(t)

	cases := map[string]any{
		"empty name":    map[string]string{"name": ""},
		"missing name":  map[string]string{"icon": "🛒"},
		"name too long": map[string]string{"name": string(make([]byte, 51))},
		"malformed":     `{"name":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c, rec := testutils.NewContext(t, testutils.Request{
				Method: http.MethodPost, Target: "/categories", Body: body,
			})
			h.Create(c)
			testutils.AssertStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestHandlerList_OK(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()
	svc.EXPECT().GetAll(gomock.Any(), userID).Return([]*domain.Category{{Name: "Casa"}, {Name: "Mercado"}}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{Target: "/categories", UserID: &userID})
	h.List(c)

	testutils.AssertStatus(t, rec, http.StatusOK)
	assert.Len(t, testutils.DecodeBody[[]domain.Category](t, rec), 2)
}

func TestHandlerList_ServiceError(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().GetAll(gomock.Any(), gomock.Any()).Return(nil, apperrors.ErrInternal)

	c, rec := testutils.NewContext(t, testutils.Request{Target: "/categories"})
	h.List(c)
	testutils.AssertStatus(t, rec, http.StatusInternalServerError)
}

func TestHandlerUpdate_OK(t *testing.T) {
	h, svc := newHandler(t)
	id, userID := uuid.New(), uuid.New()
	svc.EXPECT().Update(gomock.Any(), id, userID, gomock.Any()).Return(&domain.Category{Name: "Novo"}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPut, Target: "/categories/" + id.String(), UserID: &userID,
		Params: map[string]string{"id": id.String()},
		Body:   map[string]string{"name": "Novo"},
	})
	h.Update(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

func TestHandlerUpdate_MalformedID(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPut, Target: "/categories/abc",
		Params: map[string]string{"id": "not-a-uuid"},
		Body:   map[string]string{"name": "Novo"},
	})
	h.Update(c)

	testutils.AssertStatus(t, rec, http.StatusBadRequest)
	assert.Equal(t, "invalid id", testutils.Message(t, rec))
}

// The id is validated before the body, so a bad id short-circuits.
func TestHandlerUpdate_NotFound(t *testing.T) {
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Update(gomock.Any(), id, gomock.Any(), gomock.Any()).Return(nil, apperrors.ErrNotFound)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPut, Target: "/categories/" + id.String(),
		Params: map[string]string{"id": id.String()},
		Body:   map[string]string{"name": "Novo"},
	})
	h.Update(c)
	testutils.AssertStatus(t, rec, http.StatusNotFound)
}

func TestHandlerDelete_NoContent(t *testing.T) {
	h, svc := newHandler(t)
	id, userID := uuid.New(), uuid.New()
	svc.EXPECT().Delete(gomock.Any(), id, userID).Return(nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/categories/" + id.String(), UserID: &userID,
		Params: map[string]string{"id": id.String()},
	})
	h.Delete(c)

	testutils.AssertStatus(t, rec, http.StatusNoContent)
	assert.Zero(t, rec.Body.Len(), "204 must have an empty body")
}

func TestHandlerDelete_MalformedID(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/categories/abc",
		Params: map[string]string{"id": "nope"},
	})
	h.Delete(c)
	testutils.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlerDelete_NotFound(t *testing.T) {
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Delete(gomock.Any(), id, gomock.Any()).Return(apperrors.ErrNotFound)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/categories/" + id.String(),
		Params: map[string]string{"id": id.String()},
	})
	h.Delete(c)
	testutils.AssertStatus(t, rec, http.StatusNotFound)
}
