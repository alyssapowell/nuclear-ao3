package auth

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Limits for the OAuth endpoints, counted in memory with the same window
// counter the login limits use. Each copy of the app counts on its own (see
// docs/DECISIONS.md, "Redis removed").
//
// Every endpoint has a limit per client address. The token, introspection and
// revocation endpoints also have a limit per app, counted only once the app
// has proved who it is with its secret: a client_id is public, so counting
// requests that merely name it would let a stranger lock a real app out. The
// authorize step has no per-app limit for the same reason; it's the page
// people are sent to.

const oauthLimitWindow = time.Minute

// oauthRate is how many requests a minute one address, and one client_id,
// may make to an endpoint.
type oauthRate struct {
	PerIP     int
	PerClient int
}

var defaultOAuthRates = map[string]oauthRate{
	"token":      {PerIP: 60, PerClient: 300},
	"introspect": {PerIP: 300, PerClient: 1200},
	"revoke":     {PerIP: 60, PerClient: 300},
	"authorize":  {PerIP: 120, PerClient: 600},
}

// oauthGuard holds the counters. Its zero value is ready to use.
type oauthGuard struct {
	once    sync.Once
	limiter *Limiter
}

func (g *oauthGuard) counter() *Limiter {
	g.once.Do(func() { g.limiter = NewLimiter() })
	return g.limiter
}

func (as *AuthService) oauthRateFor(endpoint string) oauthRate {
	if r, ok := as.oauthRates[endpoint]; ok {
		return r
	}
	return defaultOAuthRates[endpoint]
}

// oauthLimit is the rate limit for one OAuth endpoint ("token",
// "introspect", "revoke" or "authorize").
func (as *AuthService) oauthLimit(endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rate := as.oauthRateFor(endpoint)
		l := as.oauth.counter()

		if ok, retry := l.Allow("oauth:"+endpoint+":ip:"+clientNetwork(c.ClientIP()), rate.PerIP, oauthLimitWindow); !ok {
			denyOAuth(c, retry)
			return
		}
		c.Set(oauthEndpointKey, endpoint)
		c.Next()
	}
}

const oauthEndpointKey = "oauth_endpoint"

func denyOAuth(c *gin.Context, retry time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error":             "rate_limit_exceeded",
		"error_description": "Too many requests. Please try again later.",
	})
}

// allowAuthenticatedClient counts one request against an app that has just
// proved who it is with its secret. It reports false, having answered 429,
// when the app is over its limit for this endpoint.
func (as *AuthService) allowAuthenticatedClient(c *gin.Context, clientID string) bool {
	endpoint := c.GetString(oauthEndpointKey)
	if endpoint == "" || endpoint == "authorize" {
		return true
	}
	rate := as.oauthRateFor(endpoint)
	if ok, retry := as.oauth.counter().Allow("oauth:"+endpoint+":client:"+clientID, rate.PerClient, oauthLimitWindow); !ok {
		denyOAuth(c, retry)
		return false
	}
	return true
}
