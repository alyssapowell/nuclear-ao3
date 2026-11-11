package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"nuclear-ao3/internal/audit"
	"nuclear-ao3/internal/moderation"
)

// firstAdmin makes the account with the email address in ADMIN_EMAIL an
// admin, for hosting platforms where nobody can run "nuclear grant". It only
// ever makes the first admin: once any account is an admin, the setting does
// nothing, so it can't be used later to take over an archive that already
// has one. Further admins are made by an admin, as usual. The account must
// already exist: sign up, then set ADMIN_EMAIL and restart. It never takes a
// role away; removing the setting changes nothing.
func firstAdmin(ctx context.Context, db *sql.DB, address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	var haveAdmin bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles
		WHERE role = 'admin' AND revoked_at IS NULL)`).Scan(&haveAdmin); err != nil {
		log.Printf("ADMIN_EMAIL: %v", err)
		return
	}
	if haveAdmin {
		log.Printf("ADMIN_EMAIL: this archive already has an admin, so the setting does nothing (it only makes the first one); you can remove it")
		return
	}
	if !strings.Contains(address, "@") {
		log.Printf("ADMIN_EMAIL: must be an email address")
		return
	}
	mod := moderation.New(db)
	userID, err := mod.UserByRef(ctx, address)
	if errors.Is(err, moderation.ErrNotFound) {
		log.Printf("ADMIN_EMAIL: no account has that email address yet; sign up with it, then restart")
		return
	}
	if err != nil {
		log.Printf("ADMIN_EMAIL: %v", err)
		return
	}
	actor := audit.Actor{Label: "setting:ADMIN_EMAIL"}
	if err := mod.GrantRole(ctx, actor, userID, "admin"); err != nil && !errors.Is(err, moderation.ErrNoChange) {
		log.Printf("ADMIN_EMAIL: %v", err)
		return
	}
	log.Printf("ADMIN_EMAIL: that account is now an admin. You can remove the setting; it only ever makes the first admin")
}
