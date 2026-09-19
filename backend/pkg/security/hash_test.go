package security_test

import (
	"strings"
	"testing"

	"github.com/financeapp/backend/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := security.HashPassword("correct horse battery staple")
	require.NoError(t, err)

	assert.NotEqual(t, "correct horse battery staple", hash, "must not be reversible plaintext")
	assert.NoError(t, security.CheckPassword(hash, "correct horse battery staple"))
	assert.Error(t, security.CheckPassword(hash, "wrong password"))
}

// bcrypt salts every hash, so the same password never yields the same digest.
func TestHashPassword_IsSalted(t *testing.T) {
	first, err := security.HashPassword("same-password")
	require.NoError(t, err)
	second, err := security.HashPassword("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.NoError(t, security.CheckPassword(first, "same-password"))
	assert.NoError(t, security.CheckPassword(second, "same-password"))
}

func TestCheckPassword_RejectsGarbageHash(t *testing.T) {
	assert.Error(t, security.CheckPassword("not-a-bcrypt-hash", "whatever"))
	assert.Error(t, security.CheckPassword("", "whatever"))
}

// bcrypt ignores everything past 72 bytes; the call must fail loudly rather
// than silently truncating the password.
func TestHashPassword_LongPassword(t *testing.T) {
	_, err := security.HashPassword(strings.Repeat("a", 100))
	assert.Error(t, err, "bcrypt refuses inputs longer than 72 bytes")
}
