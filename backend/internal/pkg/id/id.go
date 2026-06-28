// Package id generates 26-char ULID identifiers used as primary keys.
package id

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy = ulid.Monotonic(rand.Reader, 0)
	mu      sync.Mutex // protects entropy (ulid.MonotonicEntropy is not concurrency-safe)
)

// New returns a fresh ULID string in canonical 26-char Crockford base32 form.
func New() string {
	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}
