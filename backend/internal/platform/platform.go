// Package platform holds what every area of the app shares: configuration,
// the database connection, and the login and role checks.
//
// Each area (auth, works, tags, ...) gets a Deps value and mounts its routes
// with it. There is exactly one place that decides who a request belongs to,
// and it is this file.
package platform

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Env returns the environment variable key, or def when it is unset.
func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// OpenDB connects to Postgres using DATABASE_URL and checks the connection.
// There is no built-in address: guessing one could connect a command, or a
// server that applies migrations, to the wrong database.
func OpenDB() (*sql.DB, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, errors.New("DATABASE_URL is not set; set it to the archive's Postgres address (see .env.example)")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database not reachable: %w", err)
	}
	return db, nil
}

// TokenValidator turns a bearer token into the user it was issued to.
// The auth package provides the real one.
type TokenValidator func(token string) (uuid.UUID, error)

// Deps is everything an area of the app needs to mount its routes.
type Deps struct {
	DB *sql.DB

	validate TokenValidator
	app      AppTokenResolver
	appAllow AppLimiter
	appLimit int
	appEvery time.Duration
}

// Option changes how NewDeps builds Deps.
type Option func(*Deps)

// NewDeps builds the shared dependencies. Without the WithAppTokens option,
// app (OAuth) tokens are not understood at all and only login tokens work.
func NewDeps(db *sql.DB, validate TokenValidator, opts ...Option) *Deps {
	d := &Deps{DB: db, validate: validate, appLimit: DefaultAppRateLimit, appEvery: DefaultAppRateWindow}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Close shuts down the database connection.
func (d *Deps) Close() {
	d.DB.Close()
}

// userFromRequest validates the Authorization: Bearer token, if any.
// Identity comes only from a verified token, never from headers like
// X-User-ID that a client can set.
//
// When the AppAccess gate ran first it has already decided who this is (a
// login, or an app that passed its scope check), and that answer is used.
// Without the gate only login tokens are accepted: an app token is never
// believed unless the gate has checked the route's scope.
func (d *Deps) userFromRequest(c *gin.Context) (identity, bool) {
	if v, ok := c.Get(ctxIdentity); ok {
		id, ok := v.(identity)
		return id, ok && id.user != uuid.Nil
	}
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return identity{}, false
	}
	id, err := d.validate(strings.TrimPrefix(h, "Bearer "))
	if err != nil {
		return identity{}, false
	}
	return identity{user: id}, true
}

func setUser(c *gin.Context, id identity) {
	c.Set("user_id", id.user.String())
	c.Set("user_uuid", id.user)
	if id.app != nil {
		c.Set(CtxAuthKind, "app")
		c.Set(CtxClientID, id.app.ClientID.String())
		c.Set(CtxScopes, id.app.Scopes)
	} else {
		c.Set(CtxAuthKind, "login")
	}
}

// RequireUser rejects requests without a valid login token.
func (d *Deps) RequireUser(c *gin.Context) {
	id, ok := d.userFromRequest(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	setUser(c, id)
	c.Next()
}

// RequireUserFromQuery is RequireUser for WebSockets, which can't send
// headers: the token comes in the ?token= query parameter.
func (d *Deps) RequireUserFromQuery(c *gin.Context) {
	id, err := d.validate(c.Query("token"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	setUser(c, identity{user: id})
	c.Next()
}

// OptionalUser attaches the user when a valid token is present and lets
// anonymous requests through.
func (d *Deps) OptionalUser(c *gin.Context) {
	if id, ok := d.userFromRequest(c); ok {
		setUser(c, id)
	}
	c.Next()
}

// RequireRole allows the request only if the logged-in user holds one of
// roles (admins always pass). Use it after RequireUser.
func (d *Deps) RequireRole(roles ...string) gin.HandlerFunc {
	allowed := append([]string{"admin"}, roles...)
	return func(c *gin.Context) {
		id, ok := c.Get("user_uuid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "login required"})
			return
		}
		// Roles are the person's own. An app never acts with them, even if
		// a route was opened to apps by mistake.
		if IsApp(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not_available_to_apps"})
			return
		}
		var has bool
		err := d.DB.QueryRowContext(c.Request.Context(),
			`SELECT EXISTS (SELECT 1 FROM user_roles
			  WHERE user_id = $1 AND role = ANY($2) AND revoked_at IS NULL)`,
			id, pq.Array(allowed)).Scan(&has)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not check permissions"})
			return
		}
		if !has {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "you don't have permission to do this"})
			return
		}
		c.Next()
	}
}

// Health reports whether the database is reachable.
func (d *Deps) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	checks := gin.H{"database": "ok"}
	status := http.StatusOK
	if err := d.DB.PingContext(ctx); err != nil {
		checks["database"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": http.StatusText(status), "checks": checks})
}
