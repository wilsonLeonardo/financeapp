// Package testutils holds helpers shared by the package tests. The generated
// mocks live under mocks/<package>.
package testutils

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// Request describes the call a handler test wants to make. A string or []byte
// Body is sent verbatim, anything else is marshalled to JSON. UserID defaults
// to a random id, since GetUserID panics without one.
type Request struct {
	Method  string
	Target  string
	Body    any
	Params  map[string]string
	UserID  *uuid.UUID
	Context map[string]any
}

// NewContext builds a gin context and the recorder it writes to.
func NewContext(t *testing.T, req Request) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var body []byte
	switch v := req.Body.(type) {
	case nil:
	case string:
		body = []byte(v)
	case []byte:
		body = v
	default:
		var err error
		body, err = json.Marshal(v)
		require.NoError(t, err, "marshalling request body")
	}

	method := req.Method
	if method == "" {
		method = "GET"
	}
	target := req.Target
	if target == "" {
		target = "/"
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}

	for k, v := range req.Params {
		c.Params = append(c.Params, gin.Param{Key: k, Value: v})
	}

	userID := uuid.New()
	if req.UserID != nil {
		userID = *req.UserID
	}
	// Mirrors middleware.Auth, whose key is unexported; spelled out once here.
	c.Set("user_id", userID)

	for k, v := range req.Context {
		c.Set(k, v)
	}

	return c, rec
}

// DecodeBody unmarshals a recorded JSON response into T.
func DecodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	require.NoErrorf(t, json.Unmarshal(rec.Body.Bytes(), &out),
		"decoding response %q", rec.Body.String())
	return out
}

// Message pulls the "message" field out of an error response.
func Message(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	return DecodeBody[struct {
		Message string `json:"message"`
	}](t, rec).Message
}

// AssertStatus fails the test when the recorded status is not want.
func AssertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	require.Equalf(t, want, rec.Code, "unexpected status (body: %s)", rec.Body.String())
}
