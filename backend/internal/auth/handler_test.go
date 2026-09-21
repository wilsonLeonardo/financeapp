package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/testutils"
	authmocks "github.com/financeapp/backend/internal/testutils/mocks/auth"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newHandler(t *testing.T) (*auth.Handler, *authmocks.MockService) {
	t.Helper()
	svc := authmocks.NewMockService(gomock.NewController(t))
	return auth.NewHandler(svc), svc
}

func TestHandlerRegister_Created(t *testing.T) {
	h, svc := newHandler(t)
	want := &auth.AuthResponse{Token: "jwt-token", User: &domain.User{ID: uuid.New(), Email: "a@b.com"}}

	svc.EXPECT().Register(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *auth.RegisterRequest) (*auth.AuthResponse, error) {
		assert.Equal(t, "a@b.com", req.Email, "handler must forward the parsed payload")
		return want, nil
	})

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/register",
		Body: map[string]string{"name": "Ana", "email": "a@b.com", "password": "password123"},
	})
	h.Register(c)

	testutils.AssertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, "jwt-token", testutils.DecodeBody[auth.AuthResponse](t, rec).Token)
}

func TestHandlerRegister_RejectsInvalidPayload(t *testing.T) {
	// The service must never be reached, so no EXPECT is registered: gomock
	// fails the test if the handler calls it anyway.
	h, _ := newHandler(t)

	cases := map[string]map[string]string{
		"short password": {"name": "Ana", "email": "a@b.com", "password": "short"},
		"bad email":      {"name": "Ana", "email": "not-an-email", "password": "password123"},
		"missing name":   {"email": "a@b.com", "password": "password123"},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c, rec := testutils.NewContext(t, testutils.Request{
				Method: http.MethodPost, Target: "/auth/register", Body: body,
			})
			h.Register(c)
			testutils.AssertStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestHandlerRegister_MalformedJSON(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/register", Body: `{"name":`,
	})
	h.Register(c)
	testutils.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlerRegister_MapsServiceErrorStatus(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil, apperrors.ErrConflict)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/register",
		Body: map[string]string{"name": "Ana", "email": "a@b.com", "password": "password123"},
	})
	h.Register(c)

	testutils.AssertStatus(t, rec, http.StatusConflict)
	assert.Equal(t, "resource already exists", testutils.Message(t, rec),
		"the AppError message must reach the client")
}

// A plain error carries no status, so it must not leak as anything but a 500.
func TestHandlerRegister_UnknownErrorBecomes500(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil, errBoom)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/register",
		Body: map[string]string{"name": "Ana", "email": "a@b.com", "password": "password123"},
	})
	h.Register(c)

	testutils.AssertStatus(t, rec, http.StatusInternalServerError)
	assert.Equal(t, "internal server error", testutils.Message(t, rec),
		"an unknown error must not leak internals")
}

func TestHandlerLogin_OK(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Login(gomock.Any(), gomock.Any()).Return(&auth.AuthResponse{Token: "t", User: &domain.User{}}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/login",
		Body: map[string]string{"email": "a@b.com", "password": "whatever"},
	})
	h.Login(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

func TestHandlerLogin_InvalidCredentials(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, apperrors.New(http.StatusUnauthorized, "invalid credentials"))

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/login",
		Body: map[string]string{"email": "a@b.com", "password": "wrong"},
	})
	h.Login(c)
	testutils.AssertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandlerLogin_MissingPassword(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/login",
		Body: map[string]string{"email": "a@b.com"},
	})
	h.Login(c)
	testutils.AssertStatus(t, rec, http.StatusBadRequest)
}

func TestHandlerLogout_RevokesTokenFromContext(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Logout(gomock.Any(), "the-token").Return(nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/auth/logout",
		Context: map[string]any{"token": "the-token"},
	})
	h.Logout(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

// Without a token in the context there is nothing to revoke, and the handler
// must still answer 200 rather than panicking on the missing key.
func TestHandlerLogout_NoTokenInContext(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{Method: http.MethodPost, Target: "/auth/logout"})
	h.Logout(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}
