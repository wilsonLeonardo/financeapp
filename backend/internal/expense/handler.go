package expense

import (
	"net/http"
	"time"

	"github.com/financeapp/backend/pkg/middleware"
	"github.com/financeapp/backend/pkg/response"
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

// @Summary      Create a transaction
// @Tags         expenses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CreateRequest  true  "Transaction payload"
// @Success      201   {object}  github_com_financeapp_backend_internal_domain.Expense
// @Failure      400   {object}  response.Body
// @Failure      401   {object}  response.Body
// @Router       /expenses [post]
func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	expense, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, expense)
}

// @Summary      Fetch one transaction
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Transaction ID"
// @Success      200  {object}  github_com_financeapp_backend_internal_domain.Expense
// @Failure      400  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Router       /expenses/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	expense, err := h.service.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, expense)
}

// @Summary      List transactions
// @Description  Pass category_id=none to list only the transactions with no category.
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Param        category_id  query     string  false  "Category UUID, or \"none\" for uncategorised"
// @Param        type         query     string  false  "Transaction type"  Enums(expense, income)
// @Param        start_date   query     string  false  "Start of the range (YYYY-MM-DD)"
// @Param        end_date     query     string  false  "End of the range (YYYY-MM-DD)"
// @Param        page         query     int     false  "Page number"     default(1)
// @Param        page_size    query     int     false  "Rows per page"   default(20)
// @Success      200          {object}  PaginatedResponse
// @Failure      400          {object}  response.Body
// @Router       /expenses [get]
func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.List(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Update a transaction
// @Description  Only the fields present in the payload are changed.
// @Tags         expenses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string         true  "Transaction ID"
// @Param        body  body      UpdateRequest  true  "Fields to change"
// @Success      200   {object}  github_com_financeapp_backend_internal_domain.Expense
// @Failure      400   {object}  response.Body
// @Failure      404   {object}  response.Body
// @Router       /expenses/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	expense, err := h.service.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, expense)
}

// @Summary      Delete a transaction
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Transaction ID"
// @Success      204
// @Failure      400  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Router       /expenses/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// @Summary      Totals per month
// @Description  Always returns the last 12 months; the months parameter is currently ignored.
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   MonthlySummary
// @Failure      401  {object}  response.Body
// @Router       /reports/monthly [get]
func (h *Handler) MonthlySummary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var months int
	if err := c.ShouldBindQuery(&struct {
		Months *int `form:"months"`
	}{}); err != nil || months == 0 {
		months = 12
	}

	summaries, err := h.service.GetMonthlySummary(c.Request.Context(), userID, months)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, summaries)
}

// @Summary      Totals per category
// @Description  Defaults to the current month when no range is given.
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Param        start_date  query     string  false  "Start of the range (YYYY-MM-DD)"
// @Param        end_date    query     string  false  "End of the range (YYYY-MM-DD)"
// @Success      200         {array}   CategorySummary
// @Failure      401         {object}  response.Body
// @Router       /reports/categories [get]
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

	summaries, err := h.service.GetCategorySummary(c.Request.Context(), userID, start, end)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, summaries)
}
