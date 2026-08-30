package account

import "context"

// CountFor reports how many Accounts exist for one Identity. It is here so a
// test can prove a repeat login adds no second row. Test builds only.
func (s *PostgresStore) CountFor(ctx context.Context, id Identity) (int, error) {
	const q = `SELECT count(*) FROM accounts WHERE provider = $1 AND provider_user_id = $2`

	var n int
	err := s.pool.QueryRow(ctx, q, id.Provider, id.ProviderUserID).Scan(&n)
	return n, err
}
