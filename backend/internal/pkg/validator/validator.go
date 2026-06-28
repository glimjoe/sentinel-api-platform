// Package validator provides common input validation helpers.
package validator

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// MinLen returns an error if s has fewer than n runes.
func MinLen(s string, n int) error {
	if utf8.RuneCountInString(s) < n {
		return fmt.Errorf("%w: must be at least %d characters", errs.ErrBadRequest, n)
	}
	return nil
}

// MaxLen returns an error if s has more than n runes.
func MaxLen(s string, n int) error {
	if utf8.RuneCountInString(s) > n {
		return fmt.Errorf("%w: must be at most %d characters", errs.ErrBadRequest, n)
	}
	return nil
}

// Email returns an error if s is not a valid email address.
func Email(s string) error {
	if _, err := mail.ParseAddress(strings.TrimSpace(s)); err != nil {
		return fmt.Errorf("%w: invalid email", errs.ErrBadRequest)
	}
	return nil
}

// NotBlank returns an error if s is empty or whitespace only.
func NotBlank(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%w: must not be blank", errs.ErrBadRequest)
	}
	return nil
}
