package errors

import (
	"errors"
	"net/http"
)

// AppError is a structured application error with an HTTP status code.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Common application errors.
var (
	ErrNotFound     = &AppError{Code: http.StatusNotFound, Message: "resource not found"}
	ErrUnauthorized = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden    = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrBadRequest   = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrInternal     = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
	ErrConflict     = &AppError{Code: http.StatusConflict, Message: "resource already exists"}
)

// New creates a new AppError.
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap wraps an existing error with an AppError.
func Wrap(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// IsNotFound reports whether err is a not-found error.
func IsNotFound(err error) bool {
	var e *AppError
	if errors.As(err, &e) {
		return e.Code == http.StatusNotFound
	}
	return false
}

// IsUnauthorized reports whether err is an unauthorized error.
func IsUnauthorized(err error) bool {
	var e *AppError
	if errors.As(err, &e) {
		return e.Code == http.StatusUnauthorized
	}
	return false
}
