package auth_test

import (
	"errors"
	"github.com/financeapp/backend/pkg/logger"
	"net/http"
	"testing"
	"time"

	"github.com/financeapp/backend/internal/auth"
	"github.com/financeapp/backend/internal/domain"
	authmocks "github.com/financeapp/backend/internal/testutils/mocks/auth"
	"github.com/financeapp/backend/pkg/config"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/financeapp/backend/pkg/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

var jwtCfg = &config.JWTConfig{Secret: "test-secret", ExpiryHours: 1}

// errBoom is an error with no HTTP status attached, used to check that the
// layers above fall back to 500 instead of leaking it.
var errBoom = errors.New("boom")

func newService(t *testing.T) (auth.Service, *authmocks.MockRepository) {
	t.Helper()
	repo := authmocks.NewMockRepository(gomock.NewController(t))
	return auth.NewService(repo, nil, jwtCfg, logger.Discard()), repo
}

// storedUser returns a user whose password hash matches plain.
func storedUser(t *testing.T, email, plain string) *domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	require.NoError(t, err, "hashing fixture password")
	return &domain.User{ID: uuid.New(), Name: "Test", Email: email, Password: string(hash)}
}

// requireAppError asserts err is an *apperrors.AppError and returns it.
func requireAppError(t *testing.T, err error) *apperrors.AppError {
	t.Helper()
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr, "expected a status-carrying error")
	return appErr
}

func TestRegister_Success(t *testing.T) {
	svc, repo := newService(t)
	newID := uuid.New()

	// The repository assigns the id, and the token must be minted from it.
	repo.EXPECT().CreateUser(gomock.Any()).DoAndReturn(func(u *domain.User) error {
		u.ID = newID
		return nil
	})

	resp, err := svc.Register(&auth.RegisterRequest{
		Name: "João Silva", Email: "joao@example.com", Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "joao@example.com", resp.User.Email)
	assert.NotEmpty(t, resp.Token)

	claims, err := security.ParseToken(resp.Token, jwtCfg.Secret)
	require.NoError(t, err, "issued token must be parseable")
	assert.Equal(t, newID, claims.UserID, "token must be minted for the new user")
}

func TestRegister_HashesPasswordBeforeStoring(t *testing.T) {
	svc, repo := newService(t)
	var stored *domain.User

	repo.EXPECT().CreateUser(gomock.Any()).DoAndReturn(func(u *domain.User) error {
		stored = u
		return nil
	})

	_, err := svc.Register(&auth.RegisterRequest{Name: "Test", Email: "a@b.com", Password: "supersecret"})
	require.NoError(t, err)

	require.NotNil(t, stored)
	assert.NotEqual(t, "supersecret", stored.Password, "password must not be stored in plain text")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("supersecret")),
		"stored hash must match the password")
}

func TestRegister_PropagatesRepositoryError(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().CreateUser(gomock.Any()).Return(apperrors.ErrConflict)

	_, err := svc.Register(&auth.RegisterRequest{
		Name: "Test", Email: "dup@example.com", Password: "password123",
	})
	assert.Equal(t, http.StatusConflict, requireAppError(t, err).Code)
}

func TestLogin_Success(t *testing.T) {
	svc, repo := newService(t)
	user := storedUser(t, "user@example.com", "mypassword")
	repo.EXPECT().FindUserByEmail("user@example.com").Return(user, nil)

	resp, err := svc.Login(&auth.LoginRequest{Email: "user@example.com", Password: "mypassword"})
	require.NoError(t, err)

	claims, err := security.ParseToken(resp.Token, jwtCfg.Secret)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindUserByEmail(gomock.Any()).
		Return(storedUser(t, "user@example.com", "correct-pass"), nil)

	_, err := svc.Login(&auth.LoginRequest{Email: "user@example.com", Password: "wrong-pass"})
	assert.Equal(t, http.StatusUnauthorized, requireAppError(t, err).Code)
}

// An unknown email must not be distinguishable from a wrong password, or the
// endpoint turns into an account-enumeration oracle.
func TestLogin_UnknownEmailLooksLikeWrongPassword(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindUserByEmail("ghost@example.com").Return(nil, apperrors.ErrNotFound)

	_, err := svc.Login(&auth.LoginRequest{Email: "ghost@example.com", Password: "whatever"})
	appErr := requireAppError(t, err)
	assert.Equal(t, http.StatusUnauthorized, appErr.Code, "404 would leak which emails exist")
	assert.Equal(t, "invalid credentials", appErr.Message,
		"must match the wrong-password message exactly")
}

// A token that does not parse is already useless, so there is nothing to
// denylist and no Redis call to make.
func TestLogout_InvalidTokenIsANoOp(t *testing.T) {
	svc, _ := newService(t)
	assert.NoError(t, svc.Logout("not-a-jwt"))
}

func TestLogout_TokenSignedWithAnotherSecret(t *testing.T) {
	svc, _ := newService(t)
	other, err := security.GenerateToken(uuid.New(), "different", time.Hour)
	require.NoError(t, err)
	assert.NoError(t, svc.Logout(other), "a token we cannot verify is already rejected at the gate")
}

// An expired token has no remaining TTL, so it never reaches Redis either —
// which is why a nil client is safe in these tests.
func TestLogout_ExpiredToken(t *testing.T) {
	svc, _ := newService(t)
	expired, err := security.GenerateToken(uuid.New(), jwtCfg.Secret, -time.Hour)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.NoError(t, svc.Logout(expired))
}
