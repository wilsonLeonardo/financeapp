package importer_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/importer"
	"github.com/financeapp/backend/internal/testutils"
	impmocks "github.com/financeapp/backend/internal/testutils/mocks/importer"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newHandler(t *testing.T) (*importer.Handler, *impmocks.MockService) {
	t.Helper()
	svc := impmocks.NewMockService(gomock.NewController(t))
	return importer.NewHandler(svc), svc
}

// multipartBody builds a real multipart upload so the handler exercises
// c.Request.FormFile the same way gin would in production.
func multipartBody(t *testing.T, field, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func uploadContext(t *testing.T, userID uuid.UUID, field, filename, content string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	body, contentType := multipartBody(t, field, filename, content)
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodPost, Target: "/imports", UserID: &userID,
	})
	c.Request = httptest.NewRequest(http.MethodPost, "/imports", body)
	c.Request.Header.Set("Content-Type", contentType)
	return c, rec
}

func TestHandlerImport_OK(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()

	svc.EXPECT().Import(gomock.Any(), userID, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ multipart.File, header *multipart.FileHeader) (*domain.Import, error) {
			assert.Equal(t, "extrato.csv", header.Filename, "the original filename must reach the service")
			return &domain.Import{FileName: "extrato.csv", Imported: 2, Status: "completed"}, nil
		})

	c, rec := uploadContext(t, userID, "file", "extrato.csv", "data,descricao,valor\n2026-09-10,Mercado,10\n")
	h.Import(c)

	testutils.AssertStatus(t, rec, http.StatusOK)
	got := testutils.DecodeBody[domain.Import](t, rec)
	assert.Equal(t, 2, got.Imported)
	assert.Equal(t, "completed", got.Status)
}

func TestHandlerImport_MissingFile(t *testing.T) {
	h, _ := newHandler(t) // the service must not be reached

	c, rec := testutils.NewContext(t, testutils.Request{Method: http.MethodPost, Target: "/imports"})
	h.Import(c)

	testutils.AssertStatus(t, rec, http.StatusBadRequest)
	assert.Equal(t, "file is required", testutils.Message(t, rec))
}

// The field has to be named "file"; anything else is treated as missing.
func TestHandlerImport_WrongFieldName(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := uploadContext(t, uuid.New(), "upload", "extrato.csv", "data,descricao,valor\n")
	h.Import(c)

	testutils.AssertStatus(t, rec, http.StatusBadRequest)
	assert.Equal(t, "file is required", testutils.Message(t, rec))
}

func TestHandlerImport_MapsServiceErrorStatus(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Import(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apperrors.New(http.StatusInternalServerError, "failed to read file"))

	c, rec := uploadContext(t, uuid.New(), "file", "extrato.csv", "x")
	h.Import(c)

	testutils.AssertStatus(t, rec, http.StatusInternalServerError)
	assert.Equal(t, "failed to read file", testutils.Message(t, rec))
}

func TestHandlerImport_UnknownErrorBecomesGeneric500(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().Import(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errDB)

	c, rec := uploadContext(t, uuid.New(), "file", "extrato.csv", "x")
	h.Import(c)

	testutils.AssertStatus(t, rec, http.StatusInternalServerError)
	assert.Equal(t, "import failed", testutils.Message(t, rec), "the raw error must not leak")
}

func TestHandlerList_OK(t *testing.T) {
	h, svc := newHandler(t)
	userID := uuid.New()
	svc.EXPECT().ListImports(gomock.Any(), userID).Return([]*domain.Import{{FileName: "a.csv"}, {FileName: "b.ofx"}}, nil)

	c, rec := testutils.NewContext(t, testutils.Request{Target: "/imports", UserID: &userID})
	h.List(c)

	testutils.AssertStatus(t, rec, http.StatusOK)
	assert.Len(t, testutils.DecodeBody[[]domain.Import](t, rec), 2)
}

func TestHandlerList_ServiceError(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().ListImports(gomock.Any(), gomock.Any()).Return(nil, errDB)

	c, rec := testutils.NewContext(t, testutils.Request{Target: "/imports"})
	h.List(c)

	testutils.AssertStatus(t, rec, http.StatusInternalServerError)
	assert.Equal(t, "failed to list imports", testutils.Message(t, rec))
}

func TestHandlerRevert_OK(t *testing.T) {
	h, svc := newHandler(t)
	id, userID := uuid.New(), uuid.New()
	svc.EXPECT().RevertImport(gomock.Any(), id, userID).Return(nil)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/imports/" + id.String(), UserID: &userID,
		Params: map[string]string{"id": id.String()},
	})
	h.Revert(c)
	testutils.AssertStatus(t, rec, http.StatusOK)
}

func TestHandlerRevert_MalformedID(t *testing.T) {
	h, _ := newHandler(t)
	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/imports/abc",
		Params: map[string]string{"id": "not-a-uuid"},
	})
	h.Revert(c)

	testutils.AssertStatus(t, rec, http.StatusBadRequest)
	assert.Equal(t, "invalid id", testutils.Message(t, rec))
}

func TestHandlerRevert_NotFound(t *testing.T) {
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().RevertImport(gomock.Any(), id, gomock.Any()).Return(apperrors.ErrNotFound)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/imports/" + id.String(),
		Params: map[string]string{"id": id.String()},
	})
	h.Revert(c)
	testutils.AssertStatus(t, rec, http.StatusNotFound)
}

func TestHandlerRevert_UnknownErrorBecomesGeneric500(t *testing.T) {
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().RevertImport(gomock.Any(), id, gomock.Any()).Return(errDB)

	c, rec := testutils.NewContext(t, testutils.Request{
		Method: http.MethodDelete, Target: "/imports/" + id.String(),
		Params: map[string]string{"id": id.String()},
	})
	h.Revert(c)

	testutils.AssertStatus(t, rec, http.StatusInternalServerError)
	assert.Equal(t, "failed to revert import", testutils.Message(t, rec))
}
