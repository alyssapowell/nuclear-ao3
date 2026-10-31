package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// loginTokenTTL is how long a login lasts.
const loginTokenTTL = 14 * 24 * time.Hour

// lastSeenInterval is the most often a session's last_seen is written.
const lastSeenInterval = time.Minute

// startSession records a new login session for the request's client and
// returns its token and the token's expiry in unix seconds.
func (as *AuthService) startSession(c *gin.Context, userID uuid.UUID) (string, int64, error) {
	ip := c.ClientIP()
	if net.ParseIP(ip) == nil {
		ip = "0.0.0.0"
	}
	var sessionID uuid.UUID
	err := as.db.QueryRowContext(c, `
		INSERT INTO user_sessions (user_id, ip_address, user_agent)
		VALUES ($1, $2, $3) RETURNING id`, userID, ip, c.Request.UserAgent()).Scan(&sessionID)
	if err != nil {
		return "", 0, err
	}
	token, err := as.jwt.GenerateLoginToken(userID, sessionID, loginTokenTTL)
	if err != nil {
		return "", 0, err
	}
	return token, time.Now().Add(loginTokenTTL).Unix(), nil
}

var errSessionInactive = errors.New("session is not active")

// validateSession checks a login token and that its session and user are
// still active, returning the user and session IDs. This is the one check
// every authenticated request goes through. OAuth access tokens are opaque
// and OIDC ID tokens have a different audience, so neither reaches it.
func (as *AuthService) validateSession(ctx context.Context, token string) (userID, sessionID uuid.UUID, err error) {
	claims, err := as.jwt.parseLoginToken(token)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if userID, err = uuid.Parse(claims.Subject); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if sessionID, err = uuid.Parse(claims.SessionID); err != nil {
		return uuid.Nil, uuid.Nil, errors.New("token has no session")
	}
	var lastSeen time.Time
	err = as.db.QueryRowContext(ctx, `
		SELECT s.last_seen FROM user_sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.user_id = $2 AND s.is_active AND u.is_active`,
		sessionID, userID).Scan(&lastSeen)
	if err == sql.ErrNoRows {
		return uuid.Nil, uuid.Nil, errSessionInactive
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if time.Since(lastSeen) > lastSeenInterval {
		if _, err := as.db.ExecContext(ctx, `
			UPDATE user_sessions SET last_seen = now()
			WHERE id = $1 AND last_seen < now() - interval '1 minute'`, sessionID); err != nil {
			log.Printf("update session last_seen: %v", err)
		}
	}
	return userID, sessionID, nil
}

// revokeSessions logs the user out of every session except the one given
// (uuid.Nil means all of them).
func (as *AuthService) revokeSessions(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, userID, except uuid.UUID) error {
	_, err := db.ExecContext(ctx, `
		UPDATE user_sessions SET is_active = false
		WHERE user_id = $1 AND is_active AND id <> $2`, userID, except)
	return err
}

// currentUserID returns the user the auth middleware identified.
func currentUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// currentSessionID returns the session of the caller's token, or uuid.Nil.
func currentSessionID(c *gin.Context) uuid.UUID {
	id, _ := c.Get("session_id")
	sid, _ := id.(uuid.UUID)
	return sid
}

// Logout ends the session the request's token belongs to.
func (as *AuthService) Logout(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if sid := currentSessionID(c); sid != uuid.Nil {
		if _, err := as.db.ExecContext(c,
			`UPDATE user_sessions SET is_active = false WHERE id = $1 AND user_id = $2`, sid, userID); err != nil {
			log.Printf("logout: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
			return
		}
	}
	c.Status(http.StatusNoContent)
}

type sessionInfo struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Current   bool      `json:"current"`
}

// GetSessions lists the caller's active sessions, newest first.
func (as *AuthService) GetSessions(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	rows, err := as.db.QueryContext(c, `
		SELECT id, created_at, last_seen, host(ip_address), COALESCE(user_agent, '')
		FROM user_sessions WHERE user_id = $1 AND is_active
		ORDER BY created_at DESC`, userID)
	if err != nil {
		log.Printf("list sessions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	defer rows.Close()
	current := currentSessionID(c)
	list := []sessionInfo{}
	for rows.Next() {
		var s sessionInfo
		if err := rows.Scan(&s.ID, &s.CreatedAt, &s.LastSeen, &s.IPAddress, &s.UserAgent); err != nil {
			log.Printf("list sessions: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
			return
		}
		s.Current = s.ID == current
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		log.Printf("list sessions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// RevokeSession logs out one of the caller's own sessions.
func (as *AuthService) RevokeSession(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	res, err := as.db.ExecContext(c,
		`UPDATE user_sessions SET is_active = false WHERE id = $1 AND user_id = $2 AND is_active`, id, userID)
	if err != nil {
		log.Printf("revoke session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.Status(http.StatusNoContent)
}
