package auth

import (
	"context"
	"time"
)

// pendingTTL is how long a sign-in request or consent screen stays valid.
const pendingTTL = 10 * time.Minute

// putPending saves short-lived OAuth state under key, replacing any earlier
// value. Expired rows are cleaned up along the way.
func (as *AuthService) putPending(ctx context.Context, key string, value []byte) error {
	as.db.ExecContext(ctx, `DELETE FROM oauth_pending WHERE expires_at < now()`)
	_, err := as.db.ExecContext(ctx, `
		INSERT INTO oauth_pending (key, value, expires_at)
		VALUES ($1, $2, now() + $3 * interval '1 second')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at`,
		key, value, int(pendingTTL.Seconds()))
	return err
}

// getPending reads state without using it up. It returns sql.ErrNoRows when
// the key is missing or expired.
func (as *AuthService) getPending(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := as.db.QueryRowContext(ctx,
		`SELECT value FROM oauth_pending WHERE key = $1 AND expires_at > now()`, key).Scan(&value)
	return value, err
}

// takePending reads state and removes it in one step, so only one caller
// can use it. It returns sql.ErrNoRows when the key is missing or expired.
func (as *AuthService) takePending(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := as.db.QueryRowContext(ctx,
		`DELETE FROM oauth_pending WHERE key = $1 AND expires_at > now() RETURNING value`, key).Scan(&value)
	return value, err
}
