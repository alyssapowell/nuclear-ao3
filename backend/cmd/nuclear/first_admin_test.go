package main

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"nuclear-ao3/internal/testdb"
)

func TestAdminEmailMakesOnlyTheFirstAdmin(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	user := func(name string) uuid.UUID {
		id := uuid.New()
		if _, err := db.Exec(`INSERT INTO users (id, email, password_hash, is_active, is_verified)
			VALUES ($1, $2, 'x', true, true)`, id, name+"@example.org"); err != nil {
			t.Fatal(err)
		}
		return id
	}
	isAdmin := func(id uuid.UUID) bool {
		var ok bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM user_roles
			WHERE user_id = $1 AND role = 'admin' AND revoked_at IS NULL)`, id).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		return ok
	}

	// No account with the address yet: nothing happens.
	firstAdmin(ctx, db, "owner@example.org")
	owner := user("owner")
	if isAdmin(owner) {
		t.Fatal("made an admin of an account that signed up afterwards")
	}
	// An account id isn't accepted, only an email address.
	firstAdmin(ctx, db, owner.String())
	if isAdmin(owner) {
		t.Fatal("accepted an account id")
	}

	firstAdmin(ctx, db, " Owner@Example.org ")
	if !isAdmin(owner) {
		t.Fatal("the account wasn't made an admin")
	}
	var logged int
	db.QueryRow(`SELECT count(*) FROM admin_actions WHERE action = 'grant_role' AND actor_label = 'setting:ADMIN_EMAIL'`).Scan(&logged)
	if logged != 1 {
		t.Errorf("%d audit rows, want 1", logged)
	}

	// Once there is an admin, pointing the setting at someone else does nothing.
	other := user("other")
	firstAdmin(ctx, db, "other@example.org")
	if isAdmin(other) {
		t.Error("made a second admin; the setting must only make the first")
	}
	// Restarting with the same setting changes nothing either.
	firstAdmin(ctx, db, "owner@example.org")
	db.QueryRow(`SELECT count(*) FROM admin_actions WHERE action = 'grant_role'`).Scan(&logged)
	if logged != 1 {
		t.Errorf("%d grant rows after a restart, want 1", logged)
	}
}
