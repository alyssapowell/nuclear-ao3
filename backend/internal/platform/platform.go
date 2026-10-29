// Package platform holds what every area of the app shares: configuration,
// the database and Redis connections, and the login and role checks.
//
// Each area (auth, works, tags, ...) gets a Deps value and mounts its routes
// with it. There is exactly one place that decides who a request belongs to,
// and it is this file.
package platform

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Env returns the environment variable key, or def when it is unset.
func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// OpenDB connects to Postgres using DATABASE_URL and checks the connection.
func OpenDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", Env("DATABASE_URL", "postgres://ao3_user:ao3_password@localhost/ao3_nuclear?sslmode=disable"))
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

	validate  TokenValidator
	redisAddr string
	redisPass string
	mu        sync.Mutex
	redis     map[int]*redis.Client
}

// NewDeps builds the shared dependencies.
func NewDeps(db *sql.DB, validate TokenValidator) *Deps {
	return &Deps{
		DB:        db,
		validate:  validate,
		redisAddr: Env("REDIS_URL", "localhost:6379"),
		redisPass: Env("REDIS_PASSWORD", ""),
		redis:     map[int]*redis.Client{},
	}
}

// Redis returns a client for the given Redis database number. Areas keep
// separate database numbers so their cache keys can't collide.
func (d *Deps) Redis(n int) *redis.Client {
	d.mu.Lock()
	defer d.mu.Unlock()
	if c, ok := d.redis[n]; ok {
		return c
	}
	c := redis.NewClient(&redis.Options{
		Addr:         d.redisAddr,
		Password:     d.redisPass,
		DB:           n,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
	})
	d.redis[n] = c
	return c
}

// Close shuts down every connection Deps opened.
func (d *Deps) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.redis {
		c.Close()
	}
	d.DB.Close()
}

// userFromRequest validates the Authorization: Bearer token, if any.
// Identity comes only from a verified token, never from headers like
// X-User-ID that a client can set.
func (d *Deps) userFromRequest(c *gin.Context) (uuid.UUID, bool) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return uuid.Nil, false
	}
	id, err := d.validate(strings.TrimPrefix(h, "Bearer "))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func setUser(c *gin.Context, id uuid.UUID) {
	c.Set("user_id", id.String())
	c.Set("user_uuid", id)
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
	setUser(c, id)
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

// Health reports whether the database and Redis are reachable.
func (d *Deps) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	checks := gin.H{"database": "ok", "redis": "ok"}
	status := http.StatusOK
	if err := d.DB.PingContext(ctx); err != nil {
		checks["database"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	if err := d.Redis(0).Ping(ctx).Err(); err != nil {
		checks["redis"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": http.StatusText(status), "checks": checks})
}
