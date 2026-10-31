package auth

import (
	"net/http"
	"testing"
)

// Registering an OAuth app is for admins only: an open endpoint would let
// anyone create a client with any scope (admin included) or grant type.
func TestOnlyAdminsCanRegisterOAuthClients(t *testing.T) {
	e := newSessionEnv(t)
	e.addUser("reader", "reader@example.org", "reader-pass")
	adminID := e.addUser("boss", "boss@example.org", "boss-pass")
	e.mustExec(`INSERT INTO user_roles (user_id, role) VALUES ($1, 'admin')`, adminID)

	client := map[string]any{
		"name":          "Some app",
		"redirect_uris": []string{"https://app.example.org/callback"},
		"scopes":        []string{"read"},
		"grant_types":   []string{"authorization_code"},
	}
	const path = "/auth/register-client"
	if c := e.status("POST", path, "", client); c != http.StatusUnauthorized {
		t.Errorf("logged out: %d, want 401", c)
	}
	if c := e.status("POST", path, e.login("reader", "reader-pass"), client); c != http.StatusForbidden {
		t.Errorf("ordinary user: %d, want 403", c)
	}
	var n int
	e.as.db.QueryRow(`SELECT count(*) FROM oauth_clients`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d clients created by people who aren't admins", n)
	}
	if c := e.status("POST", path, e.login("boss", "boss-pass"), client); c != http.StatusCreated && c != http.StatusOK {
		t.Errorf("admin: %d, want success", c)
	}
	var owner string
	e.as.db.QueryRow(`SELECT COALESCE(owner_id::text, '') FROM oauth_clients`).Scan(&owner)
	if owner != adminID.String() {
		t.Errorf("client owner %q, want the admin", owner)
	}
}
