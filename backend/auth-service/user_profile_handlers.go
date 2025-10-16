package main

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
)

// GetUserProfile retrieves a user's public profile with AO3-style features
func (s *AuthService) GetUserProfile(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	// Get current user ID for permission checks
	var viewerID *uuid.UUID
	if userIDValue := c.GetString("user_id"); userIDValue != "" {
		if parsedID, err := uuid.Parse(userIDValue); err == nil {
			viewerID = &parsedID
		}
	}

	// Get user basic info and profile settings
	query := `
		SELECT 
			u.id, u.username, u.display_name, u.bio, u.location, u.website,
			u.is_verified, u.created_at,
			up.profile_visibility, up.work_visibility, up.comment_permissions,
			us.works_count, us.series_count, us.bookmarks_count, us.comments_count,
			us.kudos_given_count, us.kudos_received_count, us.words_written,
			us.last_work_date, us.join_date
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		LEFT JOIN user_statistics us ON u.id = us.user_id
		WHERE u.username = $1 AND u.is_active = true
	`

	var profile models.UserProfile
	var displayName, bio, location, website sql.NullString
	var profileVisibility, workVisibility, commentPermissions sql.NullString
	var lastWorkDate sql.NullTime

	err := s.db.QueryRow(query, username).Scan(
		&profile.ID, &profile.Username, &displayName, &bio, &location, &website,
		&profile.IsVerified, &profile.CreatedAt,
		&profileVisibility, &workVisibility, &commentPermissions,
		&profile.WorksCount, &profile.SeriesCount, &profile.BookmarksCount, &profile.CommentsCount,
		&profile.KudosGivenCount, &profile.KudosReceivedCount, &profile.WordsWritten,
		&lastWorkDate, &profile.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	// Handle nullable fields
	if displayName.Valid {
		profile.DisplayName = displayName.String
	}
	if bio.Valid {
		profile.Bio = bio.String
	}
	if location.Valid {
		profile.Location = location.String
	}
	if website.Valid {
		profile.Website = website.String
	}
	if lastWorkDate.Valid {
		profile.LastWorkDate = &lastWorkDate.Time
	}

	// Set default visibility if not set
	visibility := "public"
	if profileVisibility.Valid {
		visibility = profileVisibility.String
	}

	// Check if viewer can see this profile
	areFriends := false
	if viewerID != nil {
		// Check friendship status
		var friendshipStatus string
		err = s.db.QueryRow(`
			SELECT status FROM user_relationships 
			WHERE ((requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1))
			AND status = 'accepted'
		`, *viewerID, profile.ID).Scan(&friendshipStatus)
		if err == nil {
			areFriends = true
		}
	}

	if !models.CanViewProfile(viewerID, profile.ID, visibility, areFriends) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This profile is private"})
		return
	}

	// Get user's pseudonyms
	pseudQuery := `SELECT name FROM user_pseudonyms WHERE user_id = $1 ORDER BY is_default DESC, created_at ASC`
	rows, err := s.db.Query(pseudQuery, profile.ID)
	if err == nil {
		defer rows.Close()
		var pseudonyms []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				pseudonyms = append(pseudonyms, name)
			}
		}
		profile.Pseudonyms = pseudonyms
	}

	// Get friends count
	var friendsCount int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM user_relationships 
		WHERE (requester_id = $1 OR addressee_id = $1) AND status = 'accepted'
	`, profile.ID).Scan(&friendsCount)
	profile.FriendsCount = friendsCount

	c.JSON(http.StatusOK, profile)
}

// UpdateUserProfile updates the current user's profile
func (s *AuthService) UpdateUserProfile(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.UserProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	if req.DisplayName != nil {
		setParts = append(setParts, "display_name = $"+strconv.Itoa(argCount))
		args = append(args, *req.DisplayName)
		argCount++
	}
	if req.Bio != nil {
		setParts = append(setParts, "bio = $"+strconv.Itoa(argCount))
		args = append(args, *req.Bio)
		argCount++
	}
	if req.Location != nil {
		setParts = append(setParts, "location = $"+strconv.Itoa(argCount))
		args = append(args, *req.Location)
		argCount++
	}
	if req.Website != nil {
		setParts = append(setParts, "website = $"+strconv.Itoa(argCount))
		args = append(args, *req.Website)
		argCount++
	}

	if len(setParts) > 0 {
		// Update users table
		setParts = append(setParts, "updated_at = NOW()")
		args = append(args, userID)

		query := "UPDATE users SET " + joinStrings(setParts, ", ") + " WHERE id = $" + strconv.Itoa(argCount)
		_, err = s.db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	// Update user preferences if provided
	if req.ProfileVisibility != nil || req.WorkVisibility != nil || req.CommentPermissions != nil || req.SkinTheme != nil {
		prefsParts := []string{}
		prefsArgs := []interface{}{}
		prefsArgCount := 1

		if req.ProfileVisibility != nil {
			prefsParts = append(prefsParts, "profile_visibility = $"+strconv.Itoa(prefsArgCount))
			prefsArgs = append(prefsArgs, *req.ProfileVisibility)
			prefsArgCount++
		}
		if req.WorkVisibility != nil {
			prefsParts = append(prefsParts, "work_visibility = $"+strconv.Itoa(prefsArgCount))
			prefsArgs = append(prefsArgs, *req.WorkVisibility)
			prefsArgCount++
		}
		if req.CommentPermissions != nil {
			prefsParts = append(prefsParts, "comment_permissions = $"+strconv.Itoa(prefsArgCount))
			prefsArgs = append(prefsArgs, *req.CommentPermissions)
			prefsArgCount++
		}
		if req.SkinTheme != nil {
			prefsParts = append(prefsParts, "skin_theme = $"+strconv.Itoa(prefsArgCount))
			prefsArgs = append(prefsArgs, *req.SkinTheme)
			prefsArgCount++
		}

		prefsParts = append(prefsParts, "updated_at = NOW()")
		prefsArgs = append(prefsArgs, userID)

		prefsQuery := `
			INSERT INTO user_preferences (user_id, ` + joinStrings(getPrefsColumns(&req), ", ") + `, created_at, updated_at) 
			VALUES ($` + strconv.Itoa(prefsArgCount) + `, ` + joinStrings(getPrefsPlaceholders(&req, prefsArgCount), ", ") + `, NOW(), NOW())
			ON CONFLICT (user_id) DO UPDATE SET ` + joinStrings(prefsParts, ", ")

		_, err = s.db.Exec(prefsQuery, prefsArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// CreateUserPseudonym creates a new pseudonym for the user
func (s *AuthService) CreateUserPseudonym(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.UserPseudonymRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pseudonym data"})
		return
	}

	// Check if pseudonym name is already taken
	var exists bool
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM user_pseudonyms WHERE name = $1)", req.Name).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check pseudonym availability"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Pseudonym name is already taken"})
		return
	}

	// If this is the user's first pseudonym or they want it as default, make it default
	var pseudCount int
	s.db.QueryRow("SELECT COUNT(*) FROM user_pseudonyms WHERE user_id = $1", userID).Scan(&pseudCount)

	isDefault := req.IsDefault || pseudCount == 0

	// If making this default, unset other defaults
	if isDefault {
		s.db.Exec("UPDATE user_pseudonyms SET is_default = false WHERE user_id = $1", userID)
	}

	// Create the pseudonym
	pseudonymID := uuid.New()
	query := `
		INSERT INTO user_pseudonyms (id, user_id, name, is_default, description, icon_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err = s.db.Exec(query, pseudonymID, userID, req.Name, isDefault, req.Description, req.IconURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pseudonym"})
		return
	}

	// Return the created pseudonym
	var pseudonym models.UserPseudonym
	err = s.db.QueryRow(`
		SELECT id, user_id, name, is_default, description, icon_url, created_at
		FROM user_pseudonyms WHERE id = $1
	`, pseudonymID).Scan(
		&pseudonym.ID, &pseudonym.UserID, &pseudonym.Name, &pseudonym.IsDefault,
		&pseudonym.Description, &pseudonym.IconURL, &pseudonym.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pseudonym created but failed to retrieve details"})
		return
	}

	c.JSON(http.StatusCreated, pseudonym)
}

// GetUserPseudonyms retrieves all pseudonyms for the current user
func (s *AuthService) GetUserPseudonyms(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	query := `
		SELECT id, user_id, name, is_default, description, icon_url, created_at
		FROM user_pseudonyms 
		WHERE user_id = $1 
		ORDER BY is_default DESC, created_at ASC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pseudonyms"})
		return
	}
	defer rows.Close()

	var pseudonyms []models.UserPseudonym
	for rows.Next() {
		var pseudonym models.UserPseudonym
		err := rows.Scan(
			&pseudonym.ID, &pseudonym.UserID, &pseudonym.Name, &pseudonym.IsDefault,
			&pseudonym.Description, &pseudonym.IconURL, &pseudonym.CreatedAt,
		)
		if err != nil {
			continue
		}
		pseudonyms = append(pseudonyms, pseudonym)
	}

	c.JSON(http.StatusOK, gin.H{"pseudonyms": pseudonyms})
}

// SendFriendRequest sends a friend request to another user
func (s *AuthService) SendFriendRequest(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	targetUsername := c.Param("username")
	if targetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target username is required"})
		return
	}

	// Get target user ID
	var targetUserID uuid.UUID
	err = s.db.QueryRow("SELECT id FROM users WHERE username = $1 AND is_active = true", targetUsername).Scan(&targetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// Can't send friend request to yourself
	if userID == targetUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send friend request to yourself"})
		return
	}

	// Check if relationship already exists
	var existingStatus string
	err = s.db.QueryRow(`
		SELECT status FROM user_relationships 
		WHERE (requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1)
	`, userID, targetUserID).Scan(&existingStatus)

	if err == nil {
		switch existingStatus {
		case "pending":
			c.JSON(http.StatusConflict, gin.H{"error": "Friend request already pending"})
			return
		case "accepted":
			c.JSON(http.StatusConflict, gin.H{"error": "Already friends"})
			return
		case "blocked":
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot send friend request"})
			return
		}
	}

	// Create friend request
	relationshipID := uuid.New()
	query := `
		INSERT INTO user_relationships (id, requester_id, addressee_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW())
	`

	_, err = s.db.Exec(query, relationshipID, userID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send friend request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Friend request sent",
		"relationship_id": relationshipID,
	})
}

// RespondToFriendRequest accepts or rejects a friend request
func (s *AuthService) RespondToFriendRequest(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	relationshipIDStr := c.Param("relationshipId")
	relationshipID, err := uuid.Parse(relationshipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship ID"})
		return
	}

	var req struct {
		Response string `json:"response" validate:"required,oneof=accept reject"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Verify this is a pending request to this user
	var requesterID uuid.UUID
	var status string
	err = s.db.QueryRow(`
		SELECT requester_id, status FROM user_relationships 
		WHERE id = $1 AND addressee_id = $2
	`, relationshipID, userID).Scan(&requesterID, &status)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Friend request not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify friend request"})
		return
	}

	if status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Friend request is not pending"})
		return
	}

	// Update the relationship status
	newStatus := "rejected"
	if req.Response == "accept" {
		newStatus = "accepted"
	}

	_, err = s.db.Exec(`
		UPDATE user_relationships 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2
	`, newStatus, relationshipID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update friend request"})
		return
	}

	message := "Friend request rejected"
	if newStatus == "accepted" {
		message = "Friend request accepted"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// BlockUser blocks another user
func (s *AuthService) BlockUser(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	targetUsername := c.Param("username")
	if targetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target username is required"})
		return
	}

	var req struct {
		BlockType string `json:"block_type" validate:"required,oneof=full comments works"`
		Reason    string `json:"reason" validate:"max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get target user ID
	var targetUserID uuid.UUID
	err = s.db.QueryRow("SELECT id FROM users WHERE username = $1 AND is_active = true", targetUsername).Scan(&targetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// Can't block yourself
	if userID == targetUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot block yourself"})
		return
	}

	// Remove any existing friendship
	s.db.Exec(`
		DELETE FROM user_relationships 
		WHERE (requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1)
	`, userID, targetUserID)

	// Create or update block
	blockID := uuid.New()
	query := `
		INSERT INTO user_blocks (id, blocker_id, blocked_id, block_type, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (blocker_id, blocked_id) DO UPDATE SET 
			block_type = EXCLUDED.block_type,
			reason = EXCLUDED.reason,
			created_at = NOW()
	`

	_, err = s.db.Exec(query, blockID, userID, targetUserID, req.BlockType, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User blocked successfully"})
}

// UnblockUser removes a user block
func (s *AuthService) UnblockUser(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	targetUsername := c.Param("username")
	if targetUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target username is required"})
		return
	}

	// Get target user ID
	var targetUserID uuid.UUID
	err = s.db.QueryRow("SELECT id FROM users WHERE username = $1 AND is_active = true", targetUsername).Scan(&targetUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	// Remove the block
	result, err := s.db.Exec("DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2", userID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not blocked"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked successfully"})
}

// GetUserDashboard retrieves dashboard information for the current user
func (s *AuthService) GetUserDashboard(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get dashboard data
	query := `
		SELECT 
			u.id, u.username, u.display_name,
			COUNT(DISTINCT n.id) FILTER (WHERE n.is_read = false) as unread_notifications,
			COUNT(DISTINCT w.id) FILTER (WHERE w.status = 'posted') as published_works,
			COUNT(DISTINCT w2.id) FILTER (WHERE w2.status = 'draft') as draft_works,
			COUNT(DISTINCT b.id) as bookmarks,
			COUNT(DISTINCT s.id) as series,
			COALESCE(SUM(ws.hits), 0) as total_hits,
			COALESCE(SUM(ws.kudos), 0) as total_kudos,
			COALESCE(SUM(ws.comments), 0) as total_comments,
			MAX(w.published_at) as last_published
		FROM users u
		LEFT JOIN notifications n ON u.id = n.user_id
		LEFT JOIN works w ON u.id = w.user_id
		LEFT JOIN works w2 ON u.id = w2.user_id
		LEFT JOIN bookmarks b ON u.id = b.user_id
		LEFT JOIN series s ON u.id = s.user_id
		LEFT JOIN work_statistics ws ON w.id = ws.work_id
		WHERE u.id = $1
		GROUP BY u.id, u.username, u.display_name
	`

	var dashboard models.UserDashboard
	var displayName sql.NullString
	var lastPublished sql.NullTime

	err = s.db.QueryRow(query, userID).Scan(
		&dashboard.ID, &dashboard.Username, &displayName,
		&dashboard.UnreadNotifications, &dashboard.PublishedWorks, &dashboard.DraftWorks,
		&dashboard.Bookmarks, &dashboard.Series, &dashboard.TotalHits,
		&dashboard.TotalKudos, &dashboard.TotalComments, &lastPublished,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve dashboard"})
		return
	}

	if displayName.Valid {
		dashboard.DisplayName = &displayName.String
	}
	if lastPublished.Valid {
		dashboard.LastPublished = &lastPublished.Time
	}

	c.JSON(http.StatusOK, dashboard)
}

// Helper functions

func joinStrings(slice []string, separator string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += separator + slice[i]
	}
	return result
}

func getPrefsColumns(req *models.UserProfileUpdateRequest) []string {
	columns := []string{}
	if req.ProfileVisibility != nil {
		columns = append(columns, "profile_visibility")
	}
	if req.WorkVisibility != nil {
		columns = append(columns, "work_visibility")
	}
	if req.CommentPermissions != nil {
		columns = append(columns, "comment_permissions")
	}
	if req.SkinTheme != nil {
		columns = append(columns, "skin_theme")
	}
	return columns
}

func getPrefsPlaceholders(req *models.UserProfileUpdateRequest, startIndex int) []string {
	placeholders := []string{}
	index := startIndex + 1
	if req.ProfileVisibility != nil {
		placeholders = append(placeholders, "$"+strconv.Itoa(index))
		index++
	}
	if req.WorkVisibility != nil {
		placeholders = append(placeholders, "$"+strconv.Itoa(index))
		index++
	}
	if req.CommentPermissions != nil {
		placeholders = append(placeholders, "$"+strconv.Itoa(index))
		index++
	}
	if req.SkinTheme != nil {
		placeholders = append(placeholders, "$"+strconv.Itoa(index))
		index++
	}
	return placeholders
}
