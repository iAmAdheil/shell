// Package account stores the persistent record of a User, keyed by the OAuth
// identity they logged in with. Accounts outlive Sessions: see CONTEXT.md.
package account

import (
	"context"
	"time"
)

// Identity is who an OAuth provider says a User is. It is the only thing the
// login flow learns about a User, and the key an Account is found by.
type Identity struct {
	Provider       string // "google", later "github"
	ProviderUserID string // the provider's own stable ID for this person
	Name           string
	AvatarURL      string
}

// Account is one User's persistent record.
type Account struct {
	ID        string
	Identity  Identity
	CreatedAt time.Time
}

// Store holds Accounts. FindOrCreate returns the existing Account for an
// Identity, or creates one on first sight. It never creates a second Account
// for an Identity it has already seen.
type Store interface {
	FindOrCreate(ctx context.Context, id Identity) (Account, error)
	ByID(ctx context.Context, id string) (Account, error)
}
