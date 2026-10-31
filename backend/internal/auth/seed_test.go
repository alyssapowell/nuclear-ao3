package auth

import (
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"nuclear-ao3/shared/models"
)

// seedUser inserts an activated account straight into the database and gives
// it the extra roles asked for. Registering through the API now needs an
// emailed activation, which the OAuth suites have no use for.
func seedUser(db *sql.DB, username, email, password string, roles ...string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5, true, true)`, id, username, email, string(hash), username); err != nil {
		return nil, err
	}
	for _, role := range roles {
		if _, err := db.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, role); err != nil {
			return nil, err
		}
	}
	return &models.User{ID: id, Username: username, Email: email, IsVerified: true}, nil
}

// signIn gives a request a real login: a session row and a signed token for
// it, sent as the Authorization header (the way the browser logs in).
func signIn(as *AuthService, req *http.Request, userID uuid.UUID) error {
	var sessionID uuid.UUID
	if err := as.db.QueryRow(`INSERT INTO user_sessions (user_id, ip_address, user_agent)
		VALUES ($1, '127.0.0.1', 'oauth-test') RETURNING id`, userID).Scan(&sessionID); err != nil {
		return err
	}
	token, err := as.jwt.GenerateLoginToken(userID, sessionID, loginTokenTTL)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
