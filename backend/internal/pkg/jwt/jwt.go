// Package jwt mints and validates HS256 access tokens for the Sentinel API.
//
// Phase 1 uses access-only JWTs (no refresh-token rotation). Phase 1.5+
// will reuse the same primitive for refresh tokens by passing a different
// secret and longer TTL.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/glimjoe/sentinel-api-platform/internal/pkg/errs"
)

// Claims is the JWT payload for Sentinel access tokens.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"em"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Issuer returns "sentinel".
const Issuer = "sentinel"

// Mint creates a signed access token for the given user. jti is a fresh ULID
// so that future revocation lists can target individual tokens.
func Mint(secret, userID, email, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// Parse validates the signature and lifetime of token. On success it returns
// the parsed Claims; on failure it returns one of errs.ErrInvalidToken or
// errs.ErrTokenExpired so callers can map to a single 401 response.
func Parse(secret, token string) (*Claims, error) {
	if token == "" {
		return nil, errs.ErrInvalidToken
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errs.ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", errs.ErrInvalidToken, err)
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errs.ErrInvalidToken
	}
	return claims, nil
}