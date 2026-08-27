package account_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"

	"backend/internal/account"
)

// openStore connects to the Postgres from docker compose. Run `make db-up`
// first. The test skips, loudly, if there is no database to talk to.
func openStore(t *testing.T) *account.PostgresStore {
	t.Helper()

	if os.Getenv("DATABASE_URL") == "" {
		_ = godotenv.Load(filepath.Join("..", "..", "..", ".env"))
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("no DATABASE_URL: run `make db-up` and set DATABASE_URL in .env")
	}

	store, err := account.OpenPostgres(context.Background(), url)
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

// newIdentity returns an Identity no other test run has used.
func newIdentity(t *testing.T) account.Identity {
	t.Helper()

	b := make([]byte, 8)
	rand.Read(b)
	return account.Identity{
		Provider:       "google",
		ProviderUserID: "test-" + hex.EncodeToString(b),
		Name:           "Ada Lovelace",
		AvatarURL:      "https://example.test/ada.png",
	}
}

func TestLoggingInTwiceCreatesOnlyOneAccount(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	id := newIdentity(t)

	first, err := store.FindOrCreate(ctx, id)
	if err != nil {
		t.Fatalf("first FindOrCreate: %v", err)
	}
	second, err := store.FindOrCreate(ctx, id)
	if err != nil {
		t.Fatalf("second FindOrCreate: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("second login made a new Account: %q then %q", first.ID, second.ID)
	}
	n, err := store.CountFor(ctx, id)
	if err != nil {
		t.Fatalf("CountFor: %v", err)
	}
	if n != 1 {
		t.Errorf("%d Accounts for one Identity, want 1", n)
	}
}

func TestTwoDifferentPeopleGetDifferentAccounts(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()

	ada, err := store.FindOrCreate(ctx, newIdentity(t))
	if err != nil {
		t.Fatalf("FindOrCreate ada: %v", err)
	}
	grace, err := store.FindOrCreate(ctx, newIdentity(t))
	if err != nil {
		t.Fatalf("FindOrCreate grace: %v", err)
	}

	if ada.ID == grace.ID {
		t.Errorf("two Identities share Account %q", ada.ID)
	}
}

func TestAnAccountCanBeReadBackByID(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	id := newIdentity(t)

	created, err := store.FindOrCreate(ctx, id)
	if err != nil {
		t.Fatalf("FindOrCreate: %v", err)
	}

	got, err := store.ByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Identity.Name != id.Name {
		t.Errorf("name = %q, want %q", got.Identity.Name, id.Name)
	}
	if got.Identity.ProviderUserID != id.ProviderUserID {
		t.Errorf("provider user ID = %q, want %q", got.Identity.ProviderUserID, id.ProviderUserID)
	}
}

func TestAChangedProfileNameIsPickedUpOnNextLogin(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	id := newIdentity(t)

	created, err := store.FindOrCreate(ctx, id)
	if err != nil {
		t.Fatalf("first FindOrCreate: %v", err)
	}

	renamed := id
	renamed.Name = "Ada King"
	if _, err := store.FindOrCreate(ctx, renamed); err != nil {
		t.Fatalf("second FindOrCreate: %v", err)
	}

	got, err := store.ByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Identity.Name != "Ada King" {
		t.Errorf("name = %q, want %q", got.Identity.Name, "Ada King")
	}
}
