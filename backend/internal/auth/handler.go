package auth

import (
	"net/http"

	"github.com/financeapp/backend/pkg/response"
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

// Register creates an account and returns a token.
//
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest  true  "Registration payload"
// @Success      201   {object}  AuthResponse
// @Failure      400   {object}  response.Body
// @Failure      409   {object}  response.Body
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login exchanges credentials for a token.
//
// @Summary      Log in
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest  true  "Login payload"
// @Success      200   {object}  AuthResponse
// @Failure      400   {object}  response.Body
// @Failure      401   {object}  response.Body
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	resp, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout revokes the current token.
//
// @Summary      Log out
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Get("token")
	if tokenStr, ok := token.(string); ok {
		h.service.Logout(c.Request.Context(), tokenStr)
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
