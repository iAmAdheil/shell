// Package login tracks who is currently logged in. A Login is the server-side
// half of an auth cookie: the cookie carries only an opaque token, and this
// store maps that token to an Account.
//
// Deliberately not called a "Session": CONTEXT.md reserves that word for the
// shared shell process a User types into.
package login

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// TTL is how long a Login survives with no activity. Ticket 14 asks for a
// long-lived cookie, refreshed on activity, so a User stays logged in
// across visits.
const TTL = 30 * 24 * time.Hour

// Login links one auth cookie to one Account.
type Login struct {
	Token     string
	AccountID string
	ExpiresAt time.Time
}

// Store holds Logins in server memory. Per ticket 04 the server is a single
// instance, so there is nothing to share with another process.
type Store struct {
	mu      sync.Mutex
	byToken map[string]Login

	// now is the clock. Tests replace it to move time forward.
	now func() time.Time
}

func NewStore() *Store {
	return &Store{byToken: map[string]Login{}, now: time.Now}
}

// Create starts a Login for an Account and returns its token.
func (s *Store) Create(accountID string) Login {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := Login{
		Token:     randomToken(),
		AccountID: accountID,
		ExpiresAt: s.now().Add(TTL),
	}
	s.byToken[l.Token] = l
	return l
}

// Get returns the Login for a token and extends its life, because a request
// carrying the token is the activity the refresh is based on. An expired
// Login is dropped and reported as absent.
func (s *Store) Get(token string) (Login, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.byToken[token]
	if !ok {
		return Login{}, false
	}
	if !s.now().Before(l.ExpiresAt) {
		delete(s.byToken, token)
		return Login{}, false
	}

	l.ExpiresAt = s.now().Add(TTL)
	s.byToken[token] = l
	return l, true
}

// Delete ends a Login. A token it has never seen is not an error.
func (s *Store) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byToken, token)
}

func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
