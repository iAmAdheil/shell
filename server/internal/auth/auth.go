// Package auth turns an OAuth login into an Account and an auth cookie.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"backend/internal/account"
)

// IdentityProvider is one OAuth provider, as this package needs it: send the
// User somewhere to approve, then turn the returned code into an Identity.
//
// Everything after Exchange is provider-independent, so a test can swap this
// out and drive the whole login flow with no browser and no network.
type IdentityProvider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (account.Identity, error)
}

// randomToken returns an unguessable, URL-safe token.
func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
