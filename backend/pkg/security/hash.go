package security

import "golang.org/x/crypto/bcrypt"

// HashPassword returns the bcrypt hash of a plaintext password.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword reports whether plain matches the stored hash. A non-nil error
// means they do not match.
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
