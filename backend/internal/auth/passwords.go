package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"nuclear-ao3/internal/jobs"
	"nuclear-ao3/shared/email"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	// activationTTL is how long an activation link works.
	activationTTL = 14 * 24 * time.Hour
	// resetTTL is how long a password reset link works.
	resetTTL = 6 * time.Hour
	// resendInterval is the least time between two emails of one kind to
	// one account.
	resendInterval = 5 * time.Minute

	// AO3's password rule: 6 to 40 characters.
	minPasswordLen = 6
	maxPasswordLen = 40
	// bcrypt ignores (or, in newer versions, refuses) bytes past 72.
	maxPasswordBytes = 72
)

// passwordOK reports whether p is an allowed password.
func passwordOK(p string) bool {
	n := utf8.RuneCountInString(p)
	return n >= minPasswordLen && n <= maxPasswordLen && len(p) <= maxPasswordBytes
}

var weakPasswordMessage = "Password must be 6 to 40 characters long."

// newToken returns a random token for a link, and the hash that is stored
// in its place.
func newToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

// hashToken is the SHA-256 of a token, in hex. Tokens are long random
// secrets, so a plain hash is enough to keep them safe if the table leaks.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// tokenKind describes one of the two kinds of emailed link.
type tokenKind struct {
	table string
	ttl   time.Duration
	// eligible is the condition on the users table (as u) for an account
	// to get this email.
	eligible string
	// withIP says the table records the requester's address.
	withIP bool
	build  func(*email.Config, string, string, string) (email.Message, error)
}

var (
	activationKind = tokenKind{
		table:    "email_verification_tokens",
		ttl:      activationTTL,
		eligible: "u.is_active AND NOT u.is_verified",
		build:    (*email.Config).Activation,
	}
	resetKind = tokenKind{
		table:    "password_reset_tokens",
		ttl:      resetTTL,
		eligible: "u.is_active",
		withIP:   true,
		build:    (*email.Config).PasswordReset,
	}
)

// issue stores a new token for the user and queues the email carrying it.
// With email turned off the token is still stored but nothing is sent.
func (as *AuthService) issue(c *gin.Context, tx *sql.Tx, k tokenKind, userID uuid.UUID, to, username string) error {
	token, hash, err := newToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(k.ttl)
	if k.withIP {
		ip := c.ClientIP()
		if net.ParseIP(ip) == nil {
			ip = "0.0.0.0"
		}
		_, err = tx.ExecContext(c, `INSERT INTO `+k.table+` (user_id, token_hash, expires_at, ip_address)
			VALUES ($1, $2, $3, $4)`, userID, hash, expires, ip)
	} else {
		_, err = tx.ExecContext(c, `INSERT INTO `+k.table+` (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)`, userID, hash, expires)
	}
	if err != nil {
		return err
	}
	if as.mail == nil {
		return nil
	}
	msg, err := k.build(as.mail, to, username, token)
	if err != nil {
		return err
	}
	return jobs.Enqueue(c, tx, "email", msg)
}

// sendLink emails a link of the given kind to the account with this email
// address, if there is one that should get it and it hasn't had one in the
// last few minutes. It says nothing about any of that to the caller.
func (as *AuthService) sendLink(c *gin.Context, k tokenKind, address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	tx, err := as.db.BeginTx(c, nil)
	if err != nil {
		log.Printf("send %s link: %v", k.table, err)
		return
	}
	defer tx.Rollback()

	var userID uuid.UUID
	var to, username string
	// The row lock makes two requests at once send one email, not two.
	err = tx.QueryRowContext(c, `
		SELECT u.id, u.email, u.username FROM users u
		WHERE lower(u.email) = lower($1) AND `+k.eligible+`
		LIMIT 1 FOR UPDATE`, address).Scan(&userID, &to, &username)
	if err == sql.ErrNoRows {
		return
	}
	if err != nil {
		log.Printf("send %s link: %v", k.table, err)
		return
	}
	var recent bool
	err = tx.QueryRowContext(c, `
		SELECT EXISTS (SELECT 1 FROM `+k.table+` WHERE user_id = $1 AND created_at > now() - $2 * interval '1 second')`,
		userID, resendInterval.Seconds()).Scan(&recent)
	if err != nil {
		log.Printf("send %s link: %v", k.table, err)
		return
	}
	if recent {
		return
	}
	if err := as.issue(c, tx, k, userID, to, username); err != nil {
		log.Printf("send %s link: %v", k.table, err)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("send %s link: %v", k.table, err)
	}
}

// useToken marks an unused, unexpired token as used and returns whose it
// was. ok is false if there is no such token.
func useToken(c *gin.Context, tx *sql.Tx, k tokenKind, token string) (userID uuid.UUID, ok bool, err error) {
	if token == "" {
		return uuid.Nil, false, nil
	}
	err = tx.QueryRowContext(c, `
		UPDATE `+k.table+` SET used_at = now()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`, hashToken(token)).Scan(&userID)
	if err == sql.ErrNoRows {
		return uuid.Nil, false, nil
	}
	return userID, err == nil, err
}

// Register creates an account that can't log in until it is activated from
// the email sent to it.
func (as *AuthService) Register(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if _, err := mail.ParseAddress(req.Email); req.Username == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "A username and a valid email address are required."})
		return
	}
	if !passwordOK(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weak_password", "message": weakPasswordMessage})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("register: hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// The account, its activation token and the email are saved together:
	// the email is sent if and only if the account exists.
	tx, err := as.db.BeginTx(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	defer tx.Rollback()
	userID := uuid.New()
	_, err = tx.ExecContext(c, `
		INSERT INTO users (id, username, email, password_hash, display_name, is_active, is_verified)
		VALUES ($1, $2, $3, $4, $5, true, false)`,
		userID, req.Username, req.Email, string(hash), req.DisplayName)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		c.JSON(http.StatusConflict, gin.H{"error": "user_exists", "message": "That username or email address is already taken."})
		return
	}
	if err != nil {
		log.Printf("register: create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if as.mail == nil {
		log.Printf("register: activation emails are off (no mail settings), so %q can't be activated by email", req.Username)
	}
	if err := as.issue(c, tx, activationKind, userID, req.Email, req.Username); err != nil {
		log.Printf("register: activation email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Account created. Check your email for a link to activate it."})
}

// VerifyEmail activates the account an activation link was sent to.
func (as *AuthService) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	tx, err := as.db.BeginTx(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	defer tx.Rollback()
	userID, ok, err := useToken(c, tx, activationKind, req.Token)
	if err == nil && ok {
		_, err = tx.ExecContext(c, `UPDATE users SET is_verified = true, updated_at = now() WHERE id = $1`, userID)
	}
	if err == nil && ok {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("verify email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token", "message": "This activation link is invalid, already used, or has expired."})
		return
	}
	c.Status(http.StatusNoContent)
}

// ResendVerification sends the activation email again. It answers the same
// way whether or not the address belongs to an account.
func (as *AuthService) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	as.sendLink(c, activationKind, req.Email)
	c.JSON(http.StatusAccepted, gin.H{"message": "If that address belongs to an account waiting to be activated, a new link is on its way."})
}

// RequestPasswordReset emails a reset link. It answers the same way whether
// or not the address belongs to an account.
func (as *AuthService) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	as.sendLink(c, resetKind, req.Email)
	c.JSON(http.StatusAccepted, gin.H{"message": "If that address belongs to an account, a link to reset the password is on its way."})
}

// ConfirmPasswordReset sets a new password from a reset link and logs the
// account out everywhere. Following the link also proves the person owns the
// address, so the account counts as activated.
func (as *AuthService) ConfirmPasswordReset(c *gin.Context) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if !passwordOK(req.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weak_password", "message": weakPasswordMessage})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("reset password: hash: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	tx, err := as.db.BeginTx(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	defer tx.Rollback()
	userID, ok, err := useToken(c, tx, resetKind, req.Token)
	if err == nil && ok {
		var res sql.Result
		res, err = tx.ExecContext(c, `
			UPDATE users SET password_hash = $2, is_verified = true, updated_at = now()
			WHERE id = $1 AND is_active`, userID, string(hash))
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				ok = false
			}
		}
	}
	if err == nil && ok {
		// Any other reset links sent earlier are no good now.
		_, err = tx.ExecContext(c, `UPDATE password_reset_tokens SET used_at = now() WHERE user_id = $1 AND used_at IS NULL`, userID)
	}
	if err == nil && ok {
		err = as.revokeSessions(c, tx, userID, uuid.Nil)
	}
	if err == nil && ok {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("reset password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_token", "message": "This reset link is invalid, already used, or has expired."})
		return
	}
	c.Status(http.StatusNoContent)
}

// ChangePassword sets a new password for the logged-in user and logs out
// every other session.
func (as *AuthService) ChangePassword(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	var current string
	err := as.db.QueryRowContext(c, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&current)
	if err != nil {
		log.Printf("change password: find user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(current), []byte(req.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong_password", "message": "The current password is not correct."})
		return
	}
	if !passwordOK(req.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "weak_password", "message": weakPasswordMessage})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("change password: hash: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	tx, err := as.db.BeginTx(c, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(c, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, userID, string(hash))
	if err == nil {
		err = as.revokeSessions(c, tx, userID, currentSessionID(c))
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("change password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.Status(http.StatusNoContent)
}
