package auth

import (
	"net/http"

	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for auth routes.
type Handler struct {
	service Service
}

// NewHandler creates a new auth handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RegisterRequest true "Registration payload"
// @Success 201 {object} AuthResponse
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	resp, err := h.service.Register(&req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login godoc
// @Summary Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login payload"
// @Success 200 {object} AuthResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	resp, err := h.service.Login(&req)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout godoc
// @Summary Logout and invalidate token
// @Tags auth
// @Security BearerAuth
// @Success 200
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Get("token")
	if tokenStr, ok := token.(string); ok {
		h.service.Logout(tokenStr)
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// respondWithError maps application errors to HTTP responses.
func respondWithError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, gin.H{"message": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
}
