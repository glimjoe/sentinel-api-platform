// Package id generates 26-char ULID identifiers used as primary keys.
//
// ULIDs are 128-bit, lexicographically sortable by time, and URL-safe — ideal
// for primary keys that appear in API responses without forcing UUID dashes.
package id

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// entropy is a package-level reader for ulid.Make. Safe for concurrent use.
var entropy = ulid.Monotonic(rand.Reader, 0)

// New returns a fresh ULID string in canonical 26-char Crockford base32 form.
// e.g. "01HXYZAB3CDEF4GHIJ5KLMNOPQ"
func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}