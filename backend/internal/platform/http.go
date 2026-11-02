package platform

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CORS allows cross-origin requests only from CORS_ORIGINS, a comma-separated
// list. In production the frontend is served from the same origin as the API,
// so no extra origins are needed. Defaults cover the local dev servers.
func CORS() gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range strings.Split(Env("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = true
		}
	}
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-CSRF-Token")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SecurityHeaders adds the standard browser security headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// Logger writes one line per request. It logs the path without the query
// string: queries can carry secrets (the live-notifications connection sends
// its login token as ?token=, and OAuth requests carry codes and state), and
// logs are kept, copied and read by more people than should see those.
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		path := p.Path
		if p.Request != nil && p.Request.URL != nil {
			path = p.Request.URL.Path
		}
		return fmt.Sprintf("%s %s %s %d %s %s\n",
			p.TimeStamp.Format(time.RFC3339), p.Method, path, p.StatusCode, p.Latency, p.ErrorMessage)
	})
}
