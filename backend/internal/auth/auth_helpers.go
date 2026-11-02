package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"nuclear-ao3/shared/models"

	"github.com/google/uuid"
)

// getUserByID retrieves a user by their ID
func (as *AuthService) getUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, COALESCE(display_name, ''), COALESCE(bio, ''),
			   COALESCE(location, ''), COALESCE(website, ''),
			   is_active, is_verified, last_login_at, created_at, updated_at
		FROM users 
		WHERE id = $1`

	err := as.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.DisplayName,
		&user.Bio, &user.Location, &user.Website,
		&user.IsActive, &user.IsVerified, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// normalizeEmail is how an email address is compared and counted: trimmed
// and lowercased. Login, sign-up and the login guard all go through it.
func normalizeEmail(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

// queryRower is a *sql.DB or a *sql.Tx.
type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// defaultPseud returns the account's default pseud, or nil if it has none.
// It is how an account is shown to its owner: accounts have no username.
func defaultPseud(ctx context.Context, q queryRower, userID uuid.UUID) *models.PseudClaim {
	var p models.PseudClaim
	err := q.QueryRowContext(ctx, `SELECT id, name FROM pseuds WHERE user_id = $1
		ORDER BY is_default DESC, created_at LIMIT 1`, userID).Scan(&p.ID, &p.Name)
	if err != nil {
		return nil
	}
	return &p
}

// defaultPseudName is the default pseud's name, for greeting the owner in an
// email. It is empty when there is none.
func defaultPseudName(ctx context.Context, q queryRower, userID uuid.UUID) string {
	if p := defaultPseud(ctx, q, userID); p != nil {
		return p.Name
	}
	return ""
}
