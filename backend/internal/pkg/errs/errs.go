// Package errs defines sentinel domain errors for the Sentinel API.
//
// Service-layer code returns these directly; handlers (api/*) translate them
// into HTTP status codes via errors.Is. Wrap at the boundary with
// fmt.Errorf("...: %w", err) to preserve the chain (per CLAUDE.md).
package errs

import "errors"

// Auth-related sentinels.
var (
	// ErrEmailTaken is returned by Register when the email already exists.
	ErrEmailTaken = errors.New("email already in use")

	// ErrInvalidCredentials is returned by Login when email is unknown or password mismatches.
	// We deliberately do NOT distinguish "user not found" from "bad password"
	// to avoid leaking which emails are registered.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrUserNotFound is returned by LookupByID when no row matches.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserInactive is returned when an account is disabled by an admin.
	ErrUserInactive = errors.New("user account is disabled")

	// ErrInvalidToken indicates the JWT failed to parse or signature is bad.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired indicates the JWT exp claim is in the past.
	ErrTokenExpired = errors.New("token expired")
)

// Validation sentinels (returned by handlers when binding/validation fails).
var (
	// ErrBadRequest is a generic 400 cause. Handlers attach their own message.
	ErrBadRequest = errors.New("bad request")
)