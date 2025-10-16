package main

import (
	"database/sql"
	"fmt"

	"nuclear-ao3/shared/models"

	"github.com/google/uuid"
)

// getUserByID retrieves a user by their ID
func (as *AuthService) getUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, display_name, bio, location, website, 
			   is_active, is_verified, last_login_at, created_at, updated_at
		FROM users 
		WHERE id = $1`

	err := as.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
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

// UserStats represents user statistics
type UserStats struct {
	WorkCount     int `json:"work_count"`
	BookmarkCount int `json:"bookmark_count"`
}
