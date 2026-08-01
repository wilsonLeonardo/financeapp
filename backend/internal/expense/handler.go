package expense

import (
	"net/http"
	"time"

	"github.com/financeapp/backend/internal/middleware"
	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for expense routes.
type Handler struct {
	service Service
}

// NewHandler creates a new expense handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	expense, err := h.service.Create(userID, &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, expense)
}

func (h *Handler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}

	expense, err := h.service.GetByID(id, userID)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, expense)
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	result, err := h.service.List(userID, req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	expense, err := h.service.Update(id, userID, &req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, expense)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}

	if err := h.service.Delete(id, userID); err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) MonthlySummary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var months int
	if err := c.ShouldBindQuery(&struct {
		Months *int `form:"months"`
	}{}); err != nil || months == 0 {
		months = 12
	}

	summaries, err := h.service.GetMonthlySummary(userID, months)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, summaries)
}

func (h *Handler) CategorySummary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)

	if startStr := c.Query("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			start = t
		}
	}
	if endStr := c.Query("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			end = t
		}
	}

	summaries, err := h.service.GetCategorySummary(userID, start, end)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, summaries)
}

func respondWithError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, gin.H{"message": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
}
