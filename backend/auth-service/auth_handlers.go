package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Basic auth handlers for testing OAuth2/OIDC functionality

// generateRefreshToken creates a cryptographically secure refresh token
func generateRefreshToken() (string, string, error) {
	// Generate 32 random bytes
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}

	// Create the token string (what we send to client)
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create hash (what we store in database)
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	return token, tokenHash, nil
}

// storeAuthRefreshToken stores a refresh token in the database
func (as *AuthService) storeAuthRefreshToken(userID uuid.UUID, tokenHash string, userAgent, ipAddress string) error {
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := as.db.Exec(query, userID, tokenHash, expiresAt, userAgent, ipAddress)
	return err
}

// validateAuthRefreshToken validates a refresh token and returns the user ID
func (as *AuthService) validateAuthRefreshToken(token string) (uuid.UUID, error) {
	// Hash the provided token
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	var userID uuid.UUID
	var revokedAt *time.Time

	query := `
		SELECT user_id, revoked_at 
		FROM refresh_tokens 
		WHERE token_hash = $1 AND expires_at > NOW()`

	err := as.db.QueryRow(query, tokenHash).Scan(&userID, &revokedAt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid refresh token")
	}

	if revokedAt != nil {
		return uuid.Nil, fmt.Errorf("refresh token has been revoked")
	}

	return userID, nil
}

// revokeAuthRefreshToken revokes a refresh token
func (as *AuthService) revokeAuthRefreshToken(token string) error {
	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1`
	_, err := as.db.Exec(query, tokenHash)
	return err
}

// isTokenBlacklisted checks if an access token has been blacklisted
func (as *AuthService) isTokenBlacklisted(tokenHash string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE token_hash = $1 AND expires_at > NOW())`
	as.db.QueryRow(query, tokenHash).Scan(&exists)
	return exists
}

// blacklistToken adds an access token to the blacklist
func (as *AuthService) blacklistToken(tokenString string, userID uuid.UUID, expiresAt time.Time) error {
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	query := `INSERT INTO token_blacklist (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`
	_, err := as.db.Exec(query, tokenHash, userID, expiresAt)
	return err
}

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

	// Generate refresh token
	refreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	// Store refresh token
	if err := as.storeAuthRefreshToken(userID, tokenHash, c.GetHeader("User-Agent"), c.ClientIP()); err != nil {
		log.Printf("Failed to store refresh token: %v", err)
		// Don't fail registration, just log
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

	// Send welcome email notification event (async, don't block registration)
	go func() {
		notificationURL := os.Getenv("NOTIFICATION_SERVICE_URL")
		if notificationURL == "" {
			notificationURL = "http://notification-service:8085"
		}

		eventData := map[string]interface{}{
			"type":      "user_registered",
			"user_id":   userID.String(),
			"username":  req.Username,
			"email":     req.Email,
			"timestamp": now.Format(time.RFC3339),
		}

		payload, _ := json.Marshal(eventData)
		resp, err := http.Post(
			notificationURL+"/api/events",
			"application/json",
			bytes.NewBuffer(payload),
		)

		if err != nil {
			log.Printf("Failed to send registration event to notification service: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Notification service returned error status: %d", resp.StatusCode)
		} else {
			log.Printf("Registration event sent for user: %s", req.Username)
		}
	}()

	c.JSON(http.StatusCreated, models.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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

	// Generate refresh token
	refreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	// Store refresh token
	if err := as.storeAuthRefreshToken(user.ID, tokenHash, c.GetHeader("User-Agent"), c.ClientIP()); err != nil {
		log.Printf("Failed to store refresh token: %v", err)
		// Don't fail login, just log
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days
	})
}

// RefreshToken handles token refresh for session extension
func (as *AuthService) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// Validate refresh token against database
	userID, err := as.validateAuthRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_refresh_token"})
		return
	}

	// Revoke old refresh token (rotation)
	if err := as.revokeAuthRefreshToken(req.RefreshToken); err != nil {
		log.Printf("Failed to revoke old refresh token: %v", err)
	}

	// Generate new tokens
	accessToken, err := as.jwt.GenerateToken(userID, "nuclear-ao3", []string{"user"}, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	newRefreshToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	if err := as.storeAuthRefreshToken(userID, tokenHash, c.GetHeader("User-Agent"), c.ClientIP()); err != nil {
		log.Printf("Failed to store new refresh token: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
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
	// Get token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_token_provided"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Get user ID from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDInterface.(uuid.UUID)

	// Parse token to get expiration
	claims, err := as.jwt.ValidateToken(tokenString)
	if err != nil {
		// Token already invalid, just return success
		c.JSON(http.StatusOK, gin.H{"message": "logged out"})
		return
	}

	// Blacklist the access token
	expiresAt := time.Unix(claims.ExpiresAt.Unix(), 0)
	if err := as.blacklistToken(tokenString, userID, expiresAt); err != nil {
		log.Printf("Failed to blacklist token: %v", err)
		// Don't fail logout, just log
	}

	// Revoke all refresh tokens for this user
	revokeQuery := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := as.db.Exec(revokeQuery, userID); err != nil {
		log.Printf("Failed to revoke refresh tokens: %v", err)
	}

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
