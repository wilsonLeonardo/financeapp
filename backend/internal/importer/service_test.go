package importer_test

import (
	"bytes"
	"errors"
	"github.com/financeapp/backend/pkg/logger"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/importer"
	expmocks "github.com/financeapp/backend/internal/testutils/mocks/expense"
	impmocks "github.com/financeapp/backend/internal/testutils/mocks/importer"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errDB = errors.New("connection reset")

// fakeFile satisfies multipart.File over an in-memory buffer.
type fakeFile struct{ *bytes.Reader }

func (fakeFile) Close() error { return nil }

func upload(content, filename string) (multipart.File, *multipart.FileHeader) {
	return fakeFile{bytes.NewReader([]byte(content))},
		&multipart.FileHeader{Filename: filename, Size: int64(len(content))}
}

const csvTwoRows = "data,descricao,valor\n" +
	"2026-09-10,Mercado,150.25\n" +
	"2026-09-11,Farmacia,32.90\n"

func newService(t *testing.T) (importer.Service, *impmocks.MockRepository, *expmocks.MockRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := impmocks.NewMockRepository(ctrl)
	expenses := expmocks.NewMockRepository(ctrl)
	return importer.NewService(repo, expenses, logger.Discard()), repo, expenses
}

func requireAppError(t *testing.T, err error) *apperrors.AppError {
	t.Helper()
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	return appErr
}

func TestImport_ParsesRowsAndRecordsTheJob(t *testing.T) {
	svc, repo, expenses := newService(t)
	userID, importID := uuid.New(), uuid.New()
	var batch []*domain.Expense

	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(imp *domain.Import) error {
		assert.Equal(t, userID, imp.UserID)
		assert.Equal(t, "extrato.csv", imp.FileName)
		assert.Equal(t, "csv", imp.FileType)
		assert.Equal(t, "processing", imp.Status, "the job starts as processing")
		imp.ID = importID
		return nil
	})
	expenses.EXPECT().CreateBatch(gomock.Any()).DoAndReturn(func(e []*domain.Expense) error {
		batch = e
		return nil
	})
	repo.EXPECT().Update(gomock.Any()).Return(nil)

	file, header := upload(csvTwoRows, "extrato.csv")
	imp, err := svc.Import(userID, file, header)
	require.NoError(t, err)

	assert.Len(t, batch, 2, "both CSV rows should be parsed")
	assert.Equal(t, 2, imp.Imported)
	assert.Equal(t, 2, imp.TotalRows)
	assert.Zero(t, imp.Errors)
	assert.Equal(t, "completed", imp.Status)
	require.NotNil(t, imp.ImportedAt, "a finished job must be timestamped")

	// Every row is tagged with a marker that starts with the import id. The
	// revert matches it with "LIKE <id>%", so the prefix is the contract: lose
	// it and reverting an import silently deletes nothing.
	for _, e := range batch {
		require.NotNil(t, e.ImportID)
		assert.True(t, strings.HasPrefix(*e.ImportID, importID.String()),
			"import marker %q must start with the import id %s", *e.ImportID, importID)
		assert.Equal(t, userID, e.UserID)
	}
}

func TestImport_DetectsOFXFromTheExtension(t *testing.T) {
	svc, repo, _ := newService(t)
	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(imp *domain.Import) error {
		assert.Equal(t, "ofx", imp.FileType)
		imp.ID = uuid.New()
		return nil
	})
	repo.EXPECT().Update(gomock.Any()).Return(nil)

	file, header := upload("OFXHEADER:100\n", "extrato.ofx")
	_, err := svc.Import(uuid.New(), file, header)
	require.NoError(t, err)
}

// A file with no usable rows produces no batch insert and a failed job.
func TestImport_MarksJobFailedWhenNothingIsImported(t *testing.T) {
	svc, repo, _ := newService(t)
	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(imp *domain.Import) error {
		imp.ID = uuid.New()
		return nil
	})
	// No CreateBatch expectation: there is nothing to insert.
	repo.EXPECT().Update(gomock.Any()).Return(nil)

	file, header := upload("data,descricao,valor\nlinha,invalida,xx\n", "ruim.csv")
	imp, err := svc.Import(uuid.New(), file, header)
	require.NoError(t, err)

	assert.Zero(t, imp.Imported)
	assert.Positive(t, imp.Errors)
	assert.Equal(t, "failed", imp.Status)
	assert.NotEmpty(t, imp.ErrorLog, "the rejected rows must be reported back")
}

func TestImport_BatchFailureIsRecordedInTheJob(t *testing.T) {
	svc, repo, expenses := newService(t)
	repo.EXPECT().Create(gomock.Any()).DoAndReturn(func(imp *domain.Import) error {
		imp.ID = uuid.New()
		return nil
	})
	expenses.EXPECT().CreateBatch(gomock.Any()).Return(errDB)

	var updated *domain.Import
	repo.EXPECT().Update(gomock.Any()).DoAndReturn(func(imp *domain.Import) error {
		updated = imp
		return nil
	})

	file, header := upload(csvTwoRows, "extrato.csv")
	imp, err := svc.Import(uuid.New(), file, header)
	require.NoError(t, err, "a failed batch is reported in the job, not as a call error")

	assert.Zero(t, imp.Imported, "nothing was stored")
	assert.Contains(t, imp.ErrorLog, "batch insert error", "the failure must be traceable")
	assert.Same(t, imp, updated)
}

func TestImport_CreateFailureAborts(t *testing.T) {
	svc, repo, _ := newService(t)
	repo.EXPECT().Create(gomock.Any()).Return(errDB)
	// Neither the parse result nor an update should follow.

	file, header := upload(csvTwoRows, "extrato.csv")
	_, err := svc.Import(uuid.New(), file, header)
	assert.Equal(t, http.StatusInternalServerError, requireAppError(t, err).Code)
}

func TestListImports_ScopesToTheUser(t *testing.T) {
	svc, repo, _ := newService(t)
	userID := uuid.New()
	repo.EXPECT().FindAll(userID).Return([]*domain.Import{{FileName: "a.csv"}}, nil)

	got, err := svc.ListImports(userID)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestRevertImport_DeletesExpensesThenTheJob(t *testing.T) {
	svc, repo, expenses := newService(t)
	id, userID := uuid.New(), uuid.New()

	gomock.InOrder(
		repo.EXPECT().FindByID(id, userID).Return(&domain.Import{ID: id}, nil),
		expenses.EXPECT().DeleteByImportID(id.String()).Return(nil),
		repo.EXPECT().Delete(id).Return(nil),
	)

	assert.NoError(t, svc.RevertImport(id, userID))
}

// Reverting someone else's import must not touch any data.
func TestRevertImport_UnknownJobDeletesNothing(t *testing.T) {
	svc, repo, _ := newService(t)
	repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apperrors.ErrNotFound)
	// No DeleteByImportID and no Delete expectations.

	err := svc.RevertImport(uuid.New(), uuid.New())
	assert.Equal(t, http.StatusNotFound, requireAppError(t, err).Code)
}

// If the expenses cannot be removed, the job row must survive so the revert can
// be retried instead of leaving orphaned transactions behind.
func TestRevertImport_KeepsJobWhenExpenseDeletionFails(t *testing.T) {
	svc, repo, expenses := newService(t)
	id := uuid.New()
	repo.EXPECT().FindByID(id, gomock.Any()).Return(&domain.Import{ID: id}, nil)
	expenses.EXPECT().DeleteByImportID(id.String()).Return(errDB)
	// No Delete expectation.

	err := svc.RevertImport(id, uuid.New())
	assert.Equal(t, http.StatusInternalServerError, requireAppError(t, err).Code)
	assert.ErrorIs(t, err, errDB)
}
