package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"nuclear-ao3/internal/jobs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

// Protection against password guessing.
//
// Two separate limits apply:
//
//  1. Per account, kept in the database (users.failed_login_attempts and
//     users.locked_until), so every copy of the app and every restart sees
//     the same count. After maxFailedLogins wrong passwords the account
//     refuses logins for lockDuration. These are AO3's numbers (Devise
//     lockable in config/initializers/devise.rb: maximum_attempts 50,
//     unlock_in 5 minutes, unlock_strategy time). As in Devise, the count has
//     no time window: it only goes back to zero after a good login, a
//     password reset, or a lock running out.
//
//  2. Per IP address, kept in memory (so each app copy counts on its own, as
//     with the browsing limit in ratelimit.go). It covers login and the
//     other endpoints where a secret can be guessed: password reset,
//     activation and their resend requests.
//
// A locked account answers exactly like a wrong password, and still spends
// the time to check the password, so someone guessing can't tell a locked
// account from an unlocked one, or either from a name that doesn't exist.
const (
	// maxFailedLogins is how many wrong passwords lock an account.
	maxFailedLogins = 50
	// lockDuration is how long a locked account refuses logins.
	lockDuration = 5 * time.Minute

	// attemptsPerIP is how many requests one address (an IPv6 /64 counts as
	// one address) may make to the guessable endpoints in attemptWindow.
	attemptsPerIP = 20
	attemptWindow = 5 * time.Minute

	// attemptsPerName and nameWindow limit tries at one login name from any
	// number of addresses, as AO3 does (config/config.yml, rack-attack: 5
	// per 20 seconds). It applies whether or not the name exists.
	attemptsPerName = 5
	nameWindow      = 20 * time.Second

	// maxUnknownNames caps how many unknown login names are remembered.
	maxUnknownNames = 10000
	// maxNameKeyLen cuts very long login names down before they are used as
	// a key, so one request can't use up a lot of memory.
	maxNameKeyLen = 100
)

var (
	loginFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuclear_login_failures_total",
		Help: "Login attempts refused as invalid credentials: wrong password, unknown name, or locked account.",
	})
	loginLockouts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "nuclear_login_lockouts_total",
		Help: "Times a login name reached the failed-login limit and was locked.",
	})
)

func init() {
	prometheus.MustRegister(loginFailures, loginLockouts)
}

// loginGuard holds the in-memory counters. Its zero value is ready to use.
type loginGuard struct {
	once sync.Once
	// byIP counts attempts per address. byNetwork is used instead when that
	// table is full, counting by the wider network.
	byIP      *windowCounter
	byNetwork *windowCounter
	// unknown counts failed logins for names that match no account.
	unknown *windowCounter
	// byName counts login attempts per login name.
	byName *windowCounter
}

func (g *loginGuard) init() {
	g.once.Do(func() {
		g.byIP = newWindowCounter(maxRateLimitKeys)
		g.byNetwork = newWindowCounter(maxRateLimitKeys / 10)
		g.unknown = newWindowCounter(maxUnknownNames)
		g.byName = newWindowCounter(maxRateLimitKeys)
	})
}

// allowNameAttempt reports whether another login try at this name is
// allowed now, and if not, how many seconds until it is.
func (as *AuthService) allowNameAttempt(login string) (bool, int) {
	g := &as.guard
	g.init()
	key := strings.ToLower(login)
	if len(key) > maxNameKeyLen {
		key = key[:maxNameKeyLen]
	}
	_, windowEnd, allowed := g.byName.hit("name:"+key, attemptsPerName, nameWindow)
	return allowed, int(time.Until(windowEnd).Seconds()) + 1
}

// attemptLimit is the per-IP limit for login, password reset and activation.
// All of those endpoints share one count per address.
func (as *AuthService) attemptLimit() gin.HandlerFunc {
	g := &as.guard
	g.init()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "attempt:" + clientNetwork(ip)
		counter := g.byIP
		if !g.byIP.tracking(key) {
			key, counter = "attempt:"+wideNetwork(ip), g.byNetwork
		}
		_, windowEnd, allowed := counter.hit(key, attemptsPerIP, attemptWindow)
		if allowed {
			c.Next()
			return
		}
		wait := int(time.Until(windowEnd).Seconds()) + 1
		c.Header("Retry-After", strconv.Itoa(wait))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":             "rate_limit_exceeded",
			"error_description": "Too many attempts. Please try again later.",
		})
	}
}

// noteUnknownFailure counts a failed login for a name that matches no
// account. Nothing is stored in the database for it. The count lives in
// memory with a cap on the number of names, and goes through the same
// numbers as a real account (maxFailedLogins in lockDuration), so the
// metrics look the same whether or not the name exists.
func (as *AuthService) noteUnknownFailure(login string) {
	loginFailures.Inc()
	g := &as.guard
	g.init()
	key := strings.ToLower(login)
	if len(key) > maxNameKeyLen {
		key = key[:maxNameKeyLen]
	}
	if used, _, allowed := g.unknown.hit(key, maxFailedLogins, lockDuration); allowed && used+1 == maxFailedLogins {
		loginLockouts.Inc()
	}
}

// recordFailedLogin counts a wrong password against an account. If it is
// the one that reaches the limit, the account is locked and the owner is
// emailed, in the same transaction. The row lock means that when several
// wrong guesses arrive at once, exactly one of them locks the account, so
// the owner gets one email per lock. Errors are logged and otherwise
// ignored: the caller refuses the login either way.
func (as *AuthService) recordFailedLogin(c *gin.Context, userID uuid.UUID, username, address string) {
	loginFailures.Inc()
	if err := as.countFailure(c.Request.Context(), userID, username, address); err != nil {
		log.Printf("login: count failed attempt: %v", err)
	}
}

func (as *AuthService) countFailure(ctx context.Context, userID uuid.UUID, username, address string) error {
	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// A lock that has run out starts the count again from zero.
	var attempts int
	var locked bool
	err = tx.QueryRowContext(ctx, `
		SELECT CASE WHEN locked_until IS NOT NULL AND locked_until <= now() THEN 0 ELSE failed_login_attempts END,
		       locked_until IS NOT NULL AND locked_until > now()
		FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&attempts, &locked)
	if err != nil {
		return err
	}
	if locked {
		// Locked by another request a moment ago. Nothing more to record.
		return nil
	}
	attempts++
	if attempts < maxFailedLogins {
		if _, err = tx.ExecContext(ctx, `UPDATE users SET failed_login_attempts = $2, locked_until = NULL WHERE id = $1`,
			userID, attempts); err != nil {
			return err
		}
		return tx.Commit()
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE users SET failed_login_attempts = $2, locked_until = now() + $3 * interval '1 second' WHERE id = $1`,
		userID, attempts, lockDuration.Seconds()); err != nil {
		return err
	}
	if as.mail != nil {
		msg, err := as.mail.AccountLocked(address, username)
		if err != nil {
			return fmt.Errorf("build lock email: %w", err)
		}
		if err := jobs.Enqueue(ctx, tx, "email", msg); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	loginLockouts.Inc()
	return nil
}

// clearFailedLogins forgets an account's failed attempts after a good login.
// It leaves a lock that is still in force alone.
func (as *AuthService) clearFailedLogins(c *gin.Context, userID uuid.UUID) {
	_, err := as.db.ExecContext(c.Request.Context(), `
		UPDATE users SET failed_login_attempts = 0, locked_until = NULL
		WHERE id = $1 AND (failed_login_attempts <> 0 OR locked_until IS NOT NULL)
		  AND (locked_until IS NULL OR locked_until <= now())`, userID)
	if err != nil {
		log.Printf("login: clear failed attempts: %v", err)
	}
}
