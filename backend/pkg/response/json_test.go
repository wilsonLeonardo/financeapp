package response_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/financeapp/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errCause = errors.New("pq: relation does not exist")

func newContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	return c, rec
}

func TestError_UsesTheStatusAndMessageOfAnAppError(t *testing.T) {
	c, rec := newContext(t)
	response.Error(c, apperrors.New(http.StatusConflict, "email already registered"))

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.JSONEq(t, `{"message":"email already registered"}`, rec.Body.String())
}

// An error with no status must never reach the client verbatim: the text can
// carry table names, queries or paths.
func TestError_UnknownErrorBecomesAGeneric500(t *testing.T) {
	c, rec := newContext(t)
	response.Error(c, errCause)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"message":"internal server error"}`, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "relation does not exist")
}

// An AppError wrapped by another error is still recognised, which a plain type
// assertion would have missed.
func TestError_FindsAWrappedAppError(t *testing.T) {
	c, rec := newContext(t)
	response.Error(c, fmt.Errorf("while listing: %w", apperrors.ErrNotFound))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.JSONEq(t, `{"message":"resource not found"}`, rec.Body.String())
}

func TestErrorWithFallback(t *testing.T) {
	t.Run("keeps the AppError message", func(t *testing.T) {
		c, rec := newContext(t)
		response.ErrorWithFallback(c, apperrors.New(http.StatusBadRequest, "file is empty"), "import failed")

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.JSONEq(t, `{"message":"file is empty"}`, rec.Body.String())
	})

	t.Run("uses the fallback for a plain error", func(t *testing.T) {
		c, rec := newContext(t)
		response.ErrorWithFallback(c, errCause, "import failed")

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.JSONEq(t, `{"message":"import failed"}`, rec.Body.String())
	})
}

func TestBadRequest(t *testing.T) {
	c, rec := newContext(t)
	response.BadRequest(c, "invalid id")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"message":"invalid id"}`, rec.Body.String())
}

// Every helper writes the same field name, so clients can read one shape.
func TestAllHelpersShareTheSameEnvelope(t *testing.T) {
	writers := map[string]func(*gin.Context){
		"Error":             func(c *gin.Context) { response.Error(c, errCause) },
		"ErrorWithFallback": func(c *gin.Context) { response.ErrorWithFallback(c, errCause, "nope") },
		"BadRequest":        func(c *gin.Context) { response.BadRequest(c, "nope") },
	}
	for name, write := range writers {
		t.Run(name, func(t *testing.T) {
			c, rec := newContext(t)
			write(c)

			var body response.Body
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.NotEmpty(t, body.Message)
			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
		})
	}
}
