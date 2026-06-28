// Package pagination provides cursor-based pagination helpers.
package pagination

import (
	"fmt"
	"strconv"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Params holds the parsed pagination parameters from a request.
type Params struct {
	Cursor string
	Limit  int
}

// Parse extracts cursor and limit from query string values.
// Returns defaults if values are missing or invalid.
func Parse(cursor string, limitStr string) (Params, error) {
	p := Params{Cursor: cursor, Limit: DefaultLimit}
	if limitStr != "" {
		n, err := strconv.Atoi(limitStr)
		if err != nil || n < 1 {
			return p, fmt.Errorf("invalid limit %q: must be >= 1", limitStr)
		}
		if n > MaxLimit {
			return p, fmt.Errorf("limit %d exceeds max %d", n, MaxLimit)
		}
		p.Limit = n
	}
	return p, nil
}

// NextCursor returns the key to use as the next cursor for a list.
func NextCursor(lastID string) string { return lastID }
