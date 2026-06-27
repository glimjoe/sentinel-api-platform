// Package password wraps golang.org/x/crypto/bcrypt so callers don't import
// crypto packages directly. Centralising the cost parameter also makes it
// easier to tune in one place.
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Hash returns a bcrypt hash of plain using the configured cost.
// An empty plain produces ErrEmpty; bcrypt refuses short inputs at low cost
// and would otherwise panic above 31 bytes, so we sanity-check first.
func Hash(plain string, cost int) (string, error) {
	if plain == "" {
		return "", errors.New("password: empty plaintext")
	}
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", fmt.Errorf("password: cost %d out of range [%d,%d]",
			cost, bcrypt.MinCost, bcrypt.MaxCost)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(h), nil
}

// Verify reports whether plain matches the stored hash.
// Returns false (not an error) when the hash is malformed or the password
// mismatches, so callers can collapse both into a single ErrInvalidCredentials.
func Verify(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}