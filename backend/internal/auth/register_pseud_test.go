package auth

import "testing"

// Registering makes the account's default pseud, named after the username,
// in the same transaction as the account (the database does it, see
// migration 037), and a refused registration leaves no pseud behind.
func TestRegisterCreatesDefaultPseud(t *testing.T) {
	e := newPasswordEnv(t)
	if c := e.register("Newcomer", "secret1"); c != 201 {
		t.Fatalf("register: %d", c)
	}
	rows, err := e.as.db.Query(`SELECT p.name, p.is_default FROM pseuds p JOIN users u ON u.id = p.user_id
		WHERE u.username = 'Newcomer'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var name string
		var def bool
		rows.Scan(&name, &def)
		n++
		if name != "Newcomer" || !def {
			t.Errorf("pseud = %q default=%v, want Newcomer and default", name, def)
		}
	}
	if n != 1 {
		t.Fatalf("%d pseuds, want 1", n)
	}

	// The same name again is refused, and changes nothing.
	var before, after int
	e.as.db.QueryRow(`SELECT count(*) FROM pseuds`).Scan(&before)
	if c := e.register("newcomer", "secret1"); c != 409 {
		t.Fatalf("second registration: %d, want 409", c)
	}
	e.as.db.QueryRow(`SELECT count(*) FROM pseuds`).Scan(&after)
	if before != after {
		t.Errorf("a refused registration left %d new pseuds", after-before)
	}
}
