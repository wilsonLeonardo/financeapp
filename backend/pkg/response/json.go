package response

import (
	"errors"
	"net/http"

	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/gin-gonic/gin"
)

// Body is the envelope every error is written in.
type Body struct {
	Message string `json:"message"`
}

// Error writes err as JSON. An AppError supplies the status and message;
// anything else becomes a generic 500, so internals never leak.
func Error(c *gin.Context, err error) {
	ErrorWithFallback(c, err, "internal server error")
}

// ErrorWithFallback is Error with a custom message for statusless errors.
func ErrorWithFallback(c *gin.Context, err error, fallback string) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.Code, Body{Message: appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, Body{Message: fallback})
}

// BadRequest writes a 400 with the given message.
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Body{Message: message})
}
