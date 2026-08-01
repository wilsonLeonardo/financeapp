package importer

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/financeapp/backend/internal/domain"
	"github.com/financeapp/backend/internal/expense"
	"github.com/financeapp/backend/internal/importer/parsers"
	"github.com/financeapp/backend/internal/middleware"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Repository ────────────────────────────────────────────────────────────────

type Repository interface {
	Create(imp *domain.Import) error
	Update(imp *domain.Import) error
	FindAll(userID uuid.UUID) ([]*domain.Import, error)
	FindByID(id, userID uuid.UUID) (*domain.Import, error)
	Delete(id uuid.UUID) error
}

type postgresRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(imp *domain.Import) error {
	return r.db.Create(imp).Error
}

func (r *postgresRepository) Update(imp *domain.Import) error {
	return r.db.Save(imp).Error
}

func (r *postgresRepository) FindAll(userID uuid.UUID) ([]*domain.Import, error) {
	var imports []*domain.Import
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&imports).Error
	return imports, err
}

func (r *postgresRepository) FindByID(id, userID uuid.UUID) (*domain.Import, error) {
	var imp domain.Import
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&imp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &imp, nil
}

func (r *postgresRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Import{}).Error
}

// ── Service ───────────────────────────────────────────────────────────────────

type Service interface {
	Import(userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Import, error)
	ListImports(userID uuid.UUID) ([]*domain.Import, error)
	RevertImport(id, userID uuid.UUID) error
}

type service struct {
	repo        Repository
	expenseRepo expense.Repository
}

func NewService(repo Repository, expenseRepo expense.Repository) Service {
	return &service{repo: repo, expenseRepo: expenseRepo}
}

func (s *service) Import(userID uuid.UUID, file multipart.File, header *multipart.FileHeader) (*domain.Import, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, apperrors.Wrap(500, "failed to read file", err)
	}

	fileType := parsers.DetectFileType(header.Filename, bytes.NewReader(buf.Bytes()))

	imp := &domain.Import{
		UserID:   userID,
		FileName: header.Filename,
		FileType: fileType,
		Status:   "processing",
	}
	if err := s.repo.Create(imp); err != nil {
		return nil, apperrors.Wrap(500, "failed to create import record", err)
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

	s.repo.Update(imp)
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
		return apperrors.Wrap(500, "failed to delete imported expenses", err)
	}
	return s.repo.Delete(id)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct{ service Service }

func NewHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) Import(c *gin.Context) {
	userID := middleware.GetUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "file is required"})
		return
	}
	defer file.Close()

	imp, err := h.service.Import(userID, file, header)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.Code, gin.H{"message": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "import failed"})
		return
	}

	c.JSON(http.StatusOK, imp)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	imports, err := h.service.ListImports(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list imports"})
		return
	}
	c.JSON(http.StatusOK, imports)
}

func (h *Handler) Revert(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}

	if err := h.service.RevertImport(id, userID); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.Code, gin.H{"message": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to revert import"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "import reverted successfully"})
}
