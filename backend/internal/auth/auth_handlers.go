package auth

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nuclear-ao3/internal/moderation"
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
	if ok, wait := as.allowNameAttempt(login); !ok {
		c.Header("Retry-After", strconv.Itoa(wait))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":             "rate_limit_exceeded",
			"error_description": "Too many attempts. Please try again later.",
		})
		return
	}

	var user models.User
	var passwordHash string
	var locked bool
	err := as.db.QueryRowContext(c.Request.Context(), `
		SELECT id, username, email, password_hash, display_name, is_active, is_verified, created_at, updated_at,
		       locked_until IS NOT NULL AND locked_until > now()
		FROM users WHERE CASE WHEN strpos($1, '@') > 0 THEN lower(email) = lower($1) ELSE lower(username) = lower($1) END
		LIMIT 1`, login).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash, &user.DisplayName,
		&user.IsActive, &user.IsVerified, &user.CreatedAt, &user.UpdatedAt, &locked)
	if err == sql.ErrNoRows {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		as.noteUnknownFailure(login)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}
	if err != nil {
		log.Printf("login: find user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// The password is always checked, even for a locked account, so a locked
	// account takes as long to refuse as any other. A locked account is
	// refused even with the right password, and with the same answer as a
	// wrong one.
	passwordRight := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) == nil
	switch {
	case locked:
		loginFailures.Inc()
	case !passwordRight:
		as.recordFailedLogin(c, user.ID, user.Username, user.Email)
	case !user.IsActive:
		loginFailures.Inc()
	}
	if locked || !passwordRight || !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}
	as.clearFailedLogins(c, user.ID)

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

// GrantRole gives an account a role (admin, moderator or tag_wrangler). The
// change and its audit row are written together.
func (as *AuthService) GrantRole(c *gin.Context) {
	actor, ok := moderation.Actor(c, as.db)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid id"})
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON"})
		return
	}
	if err := moderation.New(as.db).GrantRole(c.Request.Context(), actor, userID, body.Role); err != nil {
		moderation.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": body.Role, "granted": true})
}

// RevokeRole takes a role away from an account.
func (as *AuthService) RevokeRole(c *gin.Context) {
	actor, ok := moderation.Actor(c, as.db)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid id"})
		return
	}
	role := c.Param("role")
	if err := moderation.New(as.db).RevokeRole(c.Request.Context(), actor, userID, role); err != nil {
		moderation.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": role, "revoked": true})
}
