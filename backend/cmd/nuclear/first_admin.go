package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"

	"nuclear-ao3/internal/audit"
)

// firstAdminActor is how ADMIN_EMAIL appears in the audit log. Its one
// grant_role row there (the log can't be changed) is what marks the setting
// as used up.
const firstAdminActor = "setting:ADMIN_EMAIL"

// firstAdmin makes the account with the email address in ADMIN_EMAIL an
// admin, for hosting platforms where nobody can run "nuclear grant".
//
// It only ever makes the first admin of an archive, once. It does nothing if
// any account has ever had the admin role (even if it was taken away since),
// or if the setting has ever made an admin before, so it can't be used
// later to take over an archive. Further admins are made by an admin, as
// usual. The account must already exist and be usable: active, confirmed,
// and not banned or suspended. Sign up first, then set ADMIN_EMAIL and
// restart. It never takes a role away; removing the setting changes nothing.
//
// In trial mode (TRIAL_NO_EMAIL) nobody has proved they own an address, so
// whoever signed up with it first is the one made admin: sign up before
// setting it.
func firstAdmin(ctx context.Context, db *sql.DB, address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	if !strings.Contains(address, "@") {
		log.Printf("ADMIN_EMAIL: must be an email address")
		return
	}
	msg, err := grantFirstAdmin(ctx, db, address)
	if err != nil {
		log.Printf("ADMIN_EMAIL: %v", err)
		return
	}
	log.Printf("ADMIN_EMAIL: %s", msg)
}

func grantFirstAdmin(ctx context.Context, db *sql.DB, address string) (string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	// Several copies of the app may start at once; only one decides.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('nuclear:first-admin'))`); err != nil {
		return "", err
	}
	var used bool
	if err := tx.QueryRowContext(ctx, `SELECT
		EXISTS (SELECT 1 FROM user_roles WHERE role = 'admin')
		OR EXISTS (SELECT 1 FROM admin_actions WHERE action = 'grant_role' AND actor_label = $1)`,
		firstAdminActor).Scan(&used); err != nil {
		return "", err
	}
	if used {
		return "this archive has had an admin already, so the setting does nothing (it only makes the first one); you can remove it", nil
	}

	var userID uuid.UUID
	var email string
	var usable bool
	err = tx.QueryRowContext(ctx, `SELECT id, email,
		is_active AND is_verified AND NOT banned
		AND NOT (suspended AND (suspended_until IS NULL OR suspended_until > now()))
		FROM users WHERE email = $1 FOR UPDATE`, address).Scan(&userID, &email, &usable)
	if errors.Is(err, sql.ErrNoRows) {
		return "no account has that email address yet; sign up with it, then restart", nil
	}
	if err != nil {
		return "", err
	}
	if !usable {
		return "that account isn't confirmed, or is banned or suspended, so it wasn't made an admin", nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1, 'admin')`, userID); err != nil {
		return "", err
	}
	if err := audit.Record(ctx, tx, audit.Actor{Label: firstAdminActor}, "grant_role", "user", userID.String(), "",
		map[string]any{"email": email, "role": "admin"}); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return "that account is now an admin. You can remove the setting; it only ever makes the first admin", nil
}
