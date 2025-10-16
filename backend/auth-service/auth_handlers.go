package main

import (
	"net/http"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Basic auth handlers for testing OAuth2/OIDC functionality

// Register handles user registration
func (as *AuthService) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Create user
	userID := uuid.New()
	now := time.Now()

	// Insert user into database
	query := `
		INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, false, $6, $7)`

	_, err = as.db.Exec(query, userID, req.Username, req.Email, string(hashedPassword), req.DisplayName, now, now)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user_exists"})
		return
	}

	// Generate tokens
	accessToken, err := as.jwt.GenerateToken(userID, "nuclear-ao3", []string{"user"}, 30*24*time.Hour) // 30 days
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	// Return user and tokens
	user := &models.User{
		ID:          userID,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		IsActive:    true,
		IsVerified:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	c.JSON(http.StatusCreated, models.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: "dummy_refresh_token", // For testing
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
	})
}

// Login handles user authentication
func (as *AuthService) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// Find user
	var user models.User
	var passwordHash string
	query := `
		SELECT id, username, email, password_hash, display_name, is_active, is_verified, created_at, updated_at
		FROM users WHERE email = $1`

	err := as.db.QueryRow(query, req.Email).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash, &user.DisplayName,
		&user.IsActive, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// Generate access token
	accessToken, err := as.jwt.GenerateToken(user.ID, "nuclear-ao3", []string{"user"}, 30*24*time.Hour) // 30 days
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: "dummy_refresh_token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
	})
}

// RefreshToken handles token refresh for session extension
func (as *AuthService) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// For now, since we're using dummy refresh tokens, we'll implement a simple validation
	// In production, refresh tokens should be stored in database with expiration
	if req.RefreshToken != "dummy_refresh_token" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_refresh_token"})
		return
	}

	// Extract user ID from the current token context (simplified)
	// In production, you'd validate the refresh token against the database
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_user"})
		return
	}

	// Generate new access token (shorter TTL since it can be refreshed)
	accessToken, err := as.jwt.GenerateToken(userID, "nuclear-ao3", []string{"user"}, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": "dummy_refresh_token", // In production, generate new refresh token
		"token_type":    "Bearer",
		"expires_in":    900, // 15 minutes in seconds
	})
}

func (as *AuthService) RequestPasswordReset(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "password reset requested"})
}

func (as *AuthService) ConfirmPasswordReset(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "password reset confirmed"})
}

func (as *AuthService) VerifyEmail(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

func (as *AuthService) ResendVerification(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "verification resent"})
}

func (as *AuthService) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (as *AuthService) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	c.JSON(http.StatusOK, gin.H{"user_id": userID})
}

func (as *AuthService) UpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (as *AuthService) ChangePassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func (as *AuthService) GetSessions(c *gin.Context) {
	c.JSON(http.StatusOK, []models.UserSession{})
}

func (as *AuthService) RevokeSession(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "session revoked"})
}

func (as *AuthService) GetSecurityEvents(c *gin.Context) {
	c.JSON(http.StatusOK, []models.SecurityEvent{})
}

func (as *AuthService) ListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, []models.User{})
}

func (as *AuthService) GetUser(c *gin.Context) {
	c.JSON(http.StatusOK, models.User{})
}

func (as *AuthService) UpdateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func (as *AuthService) GrantRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "role granted"})
}

func (as *AuthService) RevokeRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "role revoked"})
}

func (as *AuthService) GetAllSecurityEvents(c *gin.Context) {
	c.JSON(http.StatusOK, []models.SecurityEvent{})
}

func (as *AuthService) GetAuthMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"metrics": "data"})
}
