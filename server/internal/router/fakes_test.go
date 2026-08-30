package router_test

import (
	"context"
	"fmt"
	"sync"

	"backend/internal/account"
)

// fakeAccounts is an in-memory account.Store. The real Postgres store is
// covered separately, in internal/account.
type fakeAccounts struct {
	mu    sync.Mutex
	byKey map[string]account.Account
	byID  map[string]account.Account
	next  int
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{
		byKey: map[string]account.Account{},
		byID:  map[string]account.Account{},
	}
}

func (f *fakeAccounts) FindOrCreate(_ context.Context, id account.Identity) (account.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := id.Provider + "|" + id.ProviderUserID
	if a, ok := f.byKey[key]; ok {
		return a, nil
	}

	f.next++
	a := account.Account{ID: fmt.Sprintf("acct-%d", f.next), Identity: id}
	f.byKey[key] = a
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeAccounts) ByID(_ context.Context, id string) (account.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	a, ok := f.byID[id]
	if !ok {
		return account.Account{}, fmt.Errorf("no account %q", id)
	}
	return a, nil
}

// fakeProvider stands in for Google. It answers with a canned Identity, so the
// login flow is testable without a browser or a network call.
type fakeProvider struct {
	identity account.Identity
	err      error
}

func (f *fakeProvider) AuthCodeURL(state string) string {
	return "https://accounts.example.test/authorize?state=" + state
}

func (f *fakeProvider) Exchange(_ context.Context, _ string) (account.Identity, error) {
	if f.err != nil {
		return account.Identity{}, f.err
	}
	return f.identity, nil
}
