package security_test

import (
	"testing"
	"time"

	apperrors "github.com/financeapp/backend/pkg/errors"
	"github.com/financeapp/backend/pkg/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret"

func TestGenerateToken_RoundTrip(t *testing.T) {
	userID := uuid.New()
	token, err := security.GenerateToken(userID, secret, time.Hour)
	require.NoError(t, err)

	claims, err := security.ParseToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.WithinDuration(t, time.Now().Add(time.Hour), claims.ExpiresAt.Time, 5*time.Second)
	assert.WithinDuration(t, time.Now(), claims.IssuedAt.Time, 5*time.Second)
}

func TestParseToken_RejectsAnotherSecret(t *testing.T) {
	token, err := security.GenerateToken(uuid.New(), secret, time.Hour)
	require.NoError(t, err)

	_, err = security.ParseToken(token, "different-secret")
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestParseToken_RejectsExpiredToken(t *testing.T) {
	token, err := security.GenerateToken(uuid.New(), secret, -time.Hour)
	require.NoError(t, err)

	_, err = security.ParseToken(token, secret)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
}

func TestParseToken_RejectsMalformedInput(t *testing.T) {
	for _, in := range []string{"", "not-a-jwt", "a.b.c", "Bearer sometoken"} {
		t.Run(in, func(t *testing.T) {
			_, err := security.ParseToken(in, secret)
			assert.ErrorIs(t, err, apperrors.ErrUnauthorized)
		})
	}
}

// The "alg: none" trick: a token asking to skip signature verification must be
// refused, not trusted.
func TestParseToken_RejectsUnsignedToken(t *testing.T) {
	// {"alg":"none","typ":"JWT"}.{"user_id":"..."} with an empty signature.
	unsigned := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
		"eyJ1c2VyX2lkIjoiNmJhN2I4MTAtOWRhZC0xMWQxLTgwYjQtMDBjMDRmZDQzMGM4In0."

	_, err := security.ParseToken(unsigned, secret)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized, "unsigned tokens must never be accepted")
}

// Every failure collapses to the same error, so a caller cannot tell a forged
// signature from an expired token.
func TestParseToken_FailuresAreIndistinguishable(t *testing.T) {
	expired, err := security.GenerateToken(uuid.New(), secret, -time.Hour)
	require.NoError(t, err)
	forged, err := security.GenerateToken(uuid.New(), "other", time.Hour)
	require.NoError(t, err)

	_, errExpired := security.ParseToken(expired, secret)
	_, errForged := security.ParseToken(forged, secret)
	assert.Equal(t, errExpired, errForged)
}
