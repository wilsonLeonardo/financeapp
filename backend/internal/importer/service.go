package importer

import (
	"bytes"
	"fmt"
	"log/slog"
	"mime/multipart"
	"strings"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/importer/parsers"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/google/uuid"
)

//go:generate go tool mockgen -source=service.go -destination=../testutils/mocks/importer/service_mock.go -package mocks

type Service interface {
	Import(userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Import, error)
	ListImports(userID uuid.UUID) ([]*domain.Import, error)
	RevertImport(id, userID uuid.UUID) error
}

type service struct {
	repo        Repository
	expenseRepo expense.Repository
	log         *slog.Logger
}

func NewService(repo Repository, expenseRepo expense.Repository, log *slog.Logger) Service {
	return &service{repo: repo, expenseRepo: expenseRepo, log: log}
}

func (s *service) Import(userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Import, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to read file", err)
	}

	fileType := parsers.DetectFileType(header.Filename, bytes.NewReader(buf.Bytes()))

	imp := &domain.Import{
		UserID:   userID,
		FileName: header.Filename,
		FileType: fileType,
		Status:   "processing",
	}
	if err := s.repo.Create(imp); err != nil {
		return nil, apperrors.WrapLogged(s.log, "failed to create import record", err)
	}

	var expenses []*domain.Expense
	var errs []string

	switch fileType {
	case "ofx":
		expenses, errs = parsers.ParseOFX(bytes.NewReader(buf.Bytes()), userID, imp.ID.String())
	default:
		expenses, errs = parsers.ParseCSV(bytes.NewReader(buf.Bytes()), userID, imp.ID.String())
	}

	imp.TotalRows = len(expenses) + len(errs)
	imp.Errors = len(errs)
	imp.ErrorLog = strings.Join(errs, "\n")

	if len(expenses) > 0 {
		if err := s.expenseRepo.CreateBatch(expenses); err != nil {
			imp.Status = "failed"
			imp.ErrorLog += fmt.Sprintf("\nbatch insert error: %v", err)
		} else {
			imp.Imported = len(expenses)
		}
	}

	now := time.Now()
	imp.ImportedAt = &now
	imp.Status = "completed"
	if imp.Errors > 0 && imp.Imported == 0 {
		imp.Status = "failed"
	}

	if err := s.repo.Update(imp); err != nil {
		// The rows were already written; only the job row is stale.
		s.log.Error("failed to record import result", "error", err, "import_id", imp.ID)
	}
	return imp, nil
}

func (s *service) ListImports(userID uuid.UUID) ([]*domain.Import, error) {
	return s.repo.FindAll(userID)
}

func (s *service) RevertImport(id, userID uuid.UUID) error {
	if _, err := s.repo.FindByID(id, userID); err != nil {
		return err
	}
	if err := s.expenseRepo.DeleteByImportID(id.String()); err != nil {
		return apperrors.WrapLogged(s.log, "failed to delete imported expenses", err)
	}
	return s.repo.Delete(id)
}
