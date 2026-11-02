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
// Every endpoint has a limit per client address, and a second one per
// client_id when the request names one. The per-client limit is higher: a
// real app's servers share one id, and anyone can type any id into a request,
// so a low limit would let a stranger lock a real app out.

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

		deny := func(retry time.Duration) {
			c.Header("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate_limit_exceeded",
				"error_description": "Too many requests. Please try again later.",
			})
		}

		if ok, retry := l.Allow("oauth:"+endpoint+":ip:"+clientNetwork(c.ClientIP()), rate.PerIP, oauthLimitWindow); !ok {
			deny(retry)
			return
		}
		if id := presentedClientID(c, endpoint == "authorize"); id != "" {
			if len(id) > 64 {
				id = id[:64]
			}
			if ok, retry := l.Allow("oauth:"+endpoint+":client:"+id, rate.PerClient, oauthLimitWindow); !ok {
				deny(retry)
				return
			}
		}
		c.Next()
	}
}
