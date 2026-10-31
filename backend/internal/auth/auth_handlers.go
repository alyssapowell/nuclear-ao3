package auth

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Basic auth handlers for testing OAuth2/OIDC functionality

// dummyHash is compared against when no user matches, so that an unknown
// login takes about as long as a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not a real password"), bcrypt.DefaultCost)

// Login handles user authentication. The login may be a username or an
// email address, in either case.
func (as *AuthService) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	login := strings.TrimSpace(req.Login)
	if login == "" {
		login = strings.TrimSpace(req.Email)
	}
	if login == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	var user models.User
	var passwordHash string
	err := as.db.QueryRowContext(c, `
		SELECT id, username, email, password_hash, display_name, is_active, is_verified, created_at, updated_at
		FROM users WHERE CASE WHEN strpos($1, '@') > 0 THEN lower(email) = lower($1) ELSE lower(username) = lower($1) END
		LIMIT 1`, login).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash, &user.DisplayName,
		&user.IsActive, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}
	if err != nil {
		log.Printf("login: find user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "not_activated", "message": "This account has not been activated yet. Use the link in the activation email."})
		return
	}

	accessToken, expiresAt, err := as.startSession(c, user.ID)
	if err != nil {
		log.Printf("login: start session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		User:        &user,
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	})
}

// notImplemented answers endpoints that exist in the API but are not built
// yet, rather than pretending to succeed.
func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

// GetProfile returns the logged-in user's own account details.
func (as *AuthService) GetProfile(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var u struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		Email       string    `json:"email"`
		DisplayName string    `json:"display_name"`
		IsVerified  bool      `json:"is_verified"`
		CreatedAt   time.Time `json:"created_at"`
	}
	err := as.db.QueryRowContext(c, `
		SELECT id, username, email, COALESCE(display_name, ''), is_verified, created_at
		FROM users WHERE id = $1`, userID).Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.IsVerified, &u.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	if err != nil {
		log.Printf("get profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, u)
}

func (as *AuthService) UpdateProfile(c *gin.Context) { notImplemented(c) }

func (as *AuthService) ListUsers(c *gin.Context)  { notImplemented(c) }
func (as *AuthService) GetUser(c *gin.Context)    { notImplemented(c) }
func (as *AuthService) UpdateUser(c *gin.Context) { notImplemented(c) }
func (as *AuthService) GrantRole(c *gin.Context)  { notImplemented(c) }
func (as *AuthService) RevokeRole(c *gin.Context) { notImplemented(c) }
