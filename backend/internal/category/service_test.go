package category_test

import (
	"errors"
	"github.com/financeapp/backend/pkg/logger"
	"net/http"
	"testing"

	"github.com/financeapp/backend/internal/category"
	"github.com/financeapp/backend/internal/domain"
	catmocks "github.com/financeapp/backend/internal/testutils/mocks/category"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDB = errors.New("connection reset")

func newService(t *testing.T) (category.Service, *catmocks.MockRepository) {
	t.Helper()
	repo := catmocks.NewMockRepository(gomock.NewController(t))
	return category.NewService(repo, logger.Discard()), repo
}

func requireAppError(t *testing.T, err error) *apperrors.AppError {
	t.Helper()
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr, "expected a status-carrying error")
	return appErr
}

func TestCreate_PersistsTheRequestFieldsForTheUser(t *testing.T) {
	svc, repo := newService(t)
	userID := uuid.New()
	var saved *domain.Category

	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(c *domain.Category) error {
		saved = c
		return nil
	})

	got, err := svc.Create(userID, &category.UpsertRequest{Name: "Mercado", Icon: "🛒", Color: "#22c55e"})
	require.NoError(t, err)

	require.NotNil(t, saved)
	assert.Equal(t, userID, saved.UserID, "category must belong to the caller")
	assert.Equal(t, "Mercado", saved.Name)
	assert.Equal(t, "🛒", saved.Icon)
	assert.Equal(t, "#22c55e", saved.Color)
	assert.Same(t, saved, got, "the persisted category must be returned to the caller")
}

func TestCreate_WrapsRepositoryFailureAs500(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().Create(gomock.Any()).Return(errDB)

	_, err := svc.Create(uuid.New(), &category.UpsertRequest{Name: "Casa"})
	assert.Equal(t, http.StatusInternalServerError, requireAppError(t, err).Code)
	assert.ErrorIs(t, err, errDB, "the underlying error must stay wrapped for logging")
}

func TestGetAll_ScopesToTheUser(t *testing.T) {
	svc, repo := newService(t)
	userID := uuid.New()
	repo.EXPECT().FindAll(userID).Return([]*domain.Category{{Name: "Casa"}, {Name: "Mercado"}}, nil)

	got, err := svc.GetAll(userID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestGetAll_PropagatesRepositoryError(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindAll(gomock.Any()).Return(nil, errDB)

	_, err := svc.GetAll(uuid.New())
	assert.ErrorIs(t, err, errDB)
}

func TestUpdate_OverwritesTheEditableFields(t *testing.T) {
	svc, repo := newService(t)
	id, userID := uuid.New(), uuid.New()
	existing := &domain.Category{ID: id, UserID: userID, Name: "Antigo", Icon: "📦", Color: "#000000"}

	repo.EXPECT().FindByID(id, userID).Return(existing, nil)
	repo.EXPECT().Update(existing).Return(nil)

	got, err := svc.Update(id, userID, &category.UpsertRequest{Name: "Novo", Icon: "🏠", Color: "#ffffff"})
	require.NoError(t, err)
	assert.Equal(t, "Novo", got.Name)
	assert.Equal(t, "🏠", got.Icon)
	assert.Equal(t, "#ffffff", got.Color)
	assert.Equal(t, userID, got.UserID, "ownership must not change on update")
	assert.Equal(t, id, got.ID, "identity must not change on update")
}

// A category that is missing, or belongs to someone else, must not be written.
func TestUpdate_UnknownCategoryDoesNotWrite(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apperrors.ErrNotFound)
	// No Update expectation: gomock fails the test if the service writes anyway.

	_, err := svc.Update(uuid.New(), uuid.New(), &category.UpsertRequest{Name: "X"})
	assert.Equal(t, http.StatusNotFound, requireAppError(t, err).Code)
}

func TestUpdate_WrapsWriteFailureAs500(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(&domain.Category{}, nil)
	repo.EXPECT().Update(gomock.Any()).Return(errDB)

	_, err := svc.Update(uuid.New(), uuid.New(), &category.UpsertRequest{Name: "X"})
	assert.Equal(t, http.StatusInternalServerError, requireAppError(t, err).Code)
}

func TestDelete_ForwardsBothIDs(t *testing.T) {
	svc, repo := newService(t)
	id, userID := uuid.New(), uuid.New()
	repo.EXPECT().Delete(id, userID).Return(nil)

	assert.NoError(t, svc.Delete(id, userID))
}

func TestDelete_PropagatesNotFound(t *testing.T) {
	svc, repo := newService(t)
	repo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(apperrors.ErrNotFound)

	err := svc.Delete(uuid.New(), uuid.New())
	assert.Equal(t, http.StatusNotFound, requireAppError(t, err).Code)
}
