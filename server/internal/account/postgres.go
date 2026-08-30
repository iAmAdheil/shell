package account

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// schema is applied at startup. The UNIQUE constraint is what makes one
// Identity mean one Account, whatever two concurrent logins try to do.
const schema = `
CREATE TABLE IF NOT EXISTS accounts (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider         text        NOT NULL,
    provider_user_id text        NOT NULL,
    name             text        NOT NULL,
    avatar_url       text        NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id)
);`

// PostgresStore keeps Accounts in Postgres, so they survive a restart.
// Sessions do not: see ticket 03.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// OpenPostgres connects, checks the connection, and creates the table if it
// is missing.
func OpenPostgres(ctx context.Context, url string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("create accounts table: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() { s.pool.Close() }

const columns = `id, provider, provider_user_id, name, avatar_url, created_at`

// FindOrCreate returns the Account for an Identity, creating it on first
// login. A repeat login updates the name and avatar, because the provider is
// the source of truth for both.
func (s *PostgresStore) FindOrCreate(ctx context.Context, id Identity) (Account, error) {
	const q = `
INSERT INTO accounts (provider, provider_user_id, name, avatar_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider, provider_user_id)
DO UPDATE SET name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
RETURNING ` + columns

	a, err := scan(s.pool.QueryRow(ctx, q, id.Provider, id.ProviderUserID, id.Name, id.AvatarURL))
	if err != nil {
		return Account{}, fmt.Errorf("find or create account: %w", err)
	}
	return a, nil
}

// ByID returns one Account.
func (s *PostgresStore) ByID(ctx context.Context, id string) (Account, error) {
	a, err := scan(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM accounts WHERE id = $1`, id))
	if err != nil {
		return Account{}, fmt.Errorf("read account %q: %w", id, err)
	}
	return a, nil
}

func scan(row pgx.Row) (Account, error) {
	var a Account
	err := row.Scan(
		&a.ID,
		&a.Identity.Provider,
		&a.Identity.ProviderUserID,
		&a.Identity.Name,
		&a.Identity.AvatarURL,
		&a.CreatedAt,
	)
	return a, err
}
