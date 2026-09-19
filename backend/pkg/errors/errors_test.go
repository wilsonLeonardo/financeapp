package errors_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/financeapp/backend/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errCause = errors.New("connection reset")

func TestNew(t *testing.T) {
	err := apperrors.New(http.StatusTeapot, "short and stout")
	assert.Equal(t, http.StatusTeapot, err.Code)
	assert.Equal(t, "short and stout", err.Error())
}

// Error() prefers the cause so logs show what actually failed.
func TestWrap_KeepsTheCauseReachable(t *testing.T) {
	err := apperrors.Wrap(http.StatusInternalServerError, "failed to list expenses", errCause)

	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.Equal(t, "failed to list expenses", err.Message, "the client-facing message")
	assert.Equal(t, errCause.Error(), err.Error(), "the logged text")
	assert.ErrorIs(t, err, errCause)
	assert.Equal(t, errCause, errors.Unwrap(err))
}

func TestIsNotFound(t *testing.T) {
	assert.True(t, apperrors.IsNotFound(apperrors.ErrNotFound))
	assert.True(t, apperrors.IsNotFound(apperrors.New(http.StatusNotFound, "gone")))
	assert.False(t, apperrors.IsNotFound(apperrors.ErrConflict))
	assert.False(t, apperrors.IsNotFound(errCause))
	assert.False(t, apperrors.IsNotFound(nil))
}

func TestIsUnauthorized(t *testing.T) {
	assert.True(t, apperrors.IsUnauthorized(apperrors.ErrUnauthorized))
	assert.False(t, apperrors.IsUnauthorized(apperrors.ErrForbidden))
	assert.False(t, apperrors.IsUnauthorized(errCause))
}

// The helpers use errors.As, so a wrapped AppError is still recognised.
func TestHelpers_SeeThroughWrapping(t *testing.T) {
	wrapped := apperrors.Wrap(http.StatusInternalServerError, "outer", apperrors.ErrNotFound)
	assert.False(t, apperrors.IsNotFound(wrapped), "the outermost status wins")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(wrapped, &appErr))
	assert.Equal(t, http.StatusInternalServerError, appErr.Code)
}

func TestSentinels(t *testing.T) {
	cases := map[*apperrors.AppError]int{
		apperrors.ErrNotFound:     http.StatusNotFound,
		apperrors.ErrUnauthorized: http.StatusUnauthorized,
		apperrors.ErrForbidden:    http.StatusForbidden,
		apperrors.ErrBadRequest:   http.StatusBadRequest,
		apperrors.ErrInternal:     http.StatusInternalServerError,
		apperrors.ErrConflict:     http.StatusConflict,
	}
	for err, want := range cases {
		assert.Equal(t, want, err.Code, err.Message)
	}
}

func TestWrapLogged(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	err := apperrors.WrapLogged(log, "failed to create expense", errCause)

	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.Equal(t, "failed to create expense", err.Message, "the client-facing message")
	assert.ErrorIs(t, err, errCause, "the cause stays reachable for the caller")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "ERROR", entry["level"])
	assert.Equal(t, "failed to create expense", entry["msg"])
	assert.Equal(t, errCause.Error(), entry["error"],
		"the cause must reach the log; the client never sees it")
}

// The generic message sent to the client must not carry the cause text, which
// can name tables, queries or paths.
func TestWrapLogged_MessageDoesNotLeakTheCause(t *testing.T) {
	err := apperrors.WrapLogged(logger.Discard(), "failed to list expenses", errCause)
	assert.NotContains(t, err.Message, errCause.Error())
}
