package category

import (
	"net/http"

	"github.com/financeapp/backend/pkg/middleware"
	"github.com/financeapp/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for category routes.
type Handler struct{ service Service }

// NewHandler creates a new category handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// @Summary      Create a category
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      UpsertRequest  true  "Category payload"
// @Success      201   {object}  github_com_financeapp_backend_internal_domain.Category
// @Failure      400   {object}  response.Body
// @Failure      401   {object}  response.Body
// @Router       /categories [post]
func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cat, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// @Summary      List categories
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   github_com_financeapp_backend_internal_domain.Category
// @Failure      401  {object}  response.Body
// @Router       /categories [get]
func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	categories, err := h.service.GetAll(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

// @Summary      Update a category
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string         true  "Category ID"
// @Param        body  body      UpsertRequest  true  "Category payload"
// @Success      200   {object}  github_com_financeapp_backend_internal_domain.Category
// @Failure      400   {object}  response.Body
// @Failure      404   {object}  response.Body
// @Router       /categories/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cat, err := h.service.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, cat)
}

// @Summary      Delete a category
// @Description  Transactions in the category are kept, with no category.
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Category ID"
// @Success      204
// @Failure      400  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Router       /categories/{id} [delete]
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
