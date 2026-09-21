package importer

import (
	"net/http"

	"github.com/financeapp/backend/pkg/middleware"
	"github.com/financeapp/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ service Service }

func NewHandler(service Service) *Handler { return &Handler{service: service} }

// @Summary      Import a bank statement
// @Description  Accepts CSV and OFX/QFX. Returns the import job with its row counts.
// @Tags         imports
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file  formData  file  true  "Statement file"
// @Success      200   {object}  github_com_financeapp_backend_internal_domain.Import
// @Failure      400   {object}  response.Body
// @Failure      500   {object}  response.Body
// @Router       /imports [post]
func (h *Handler) Import(c *gin.Context) {
	userID := middleware.GetUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	imp, err := h.service.Import(c.Request.Context(), userID, file, header)
	if err != nil {
		response.ErrorWithFallback(c, err, "import failed")
		return
	}

	c.JSON(http.StatusOK, imp)
}

// @Summary      List past imports
// @Tags         imports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   github_com_financeapp_backend_internal_domain.Import
// @Failure      500  {object}  response.Body
// @Router       /imports [get]
func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	imports, err := h.service.ListImports(c.Request.Context(), userID)
	if err != nil {
		response.ErrorWithFallback(c, err, "failed to list imports")
		return
	}
	c.JSON(http.StatusOK, imports)
}

// @Summary      Revert an import
// @Description  Deletes the transactions created by the import, then the job itself.
// @Tags         imports
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Import ID"
// @Success      200  {object}  response.Body
// @Failure      400  {object}  response.Body
// @Failure      404  {object}  response.Body
// @Router       /imports/{id} [delete]
func (h *Handler) Revert(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	if err := h.service.RevertImport(c.Request.Context(), id, userID); err != nil {
		response.ErrorWithFallback(c, err, "failed to revert import")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "import reverted successfully"})
}
