package platform

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// This file decides what an app (an OAuth client acting for a person) may
// reach. See docs/API_SCOPES.md for the table in plain words.
//
// The rule is "default deny". A login token (the person's own browser) gets
// everything the person can do. An app token gets a route only when the route
// is listed in appRoutes with a scope the person approved. A route that is
// new, forgotten, or not listed says no to apps. Nothing has to be added to
// keep a new route safe; something has to be added to open it.

// Keys under which the request's identity is kept in the gin context.
const (
	// CtxAuthKind is "login" or "app".
	CtxAuthKind = "auth_kind"
	// CtxClientID is the OAuth client id, for app requests only.
	CtxClientID = "oauth_client_id"
	// CtxScopes is the []string of scopes the person approved, for app
	// requests only.
	CtxScopes = "oauth_scopes"
	// CtxAppPseud is the uuid.UUID of the pseud the person shared with the
	// app, for app requests that have one.
	CtxAppPseud = "oauth_app_pseud"

	ctxIdentity = "auth_identity"
)

// Default per-app limit. It counts requests per (app, person), so one busy
// person cannot use up another person's share, and an app token cannot be
// used to get round the limits on logged-in browsing, which are separate.
const (
	DefaultAppRateLimit  = 120
	DefaultAppRateWindow = time.Minute
)

// AppToken is what an opaque OAuth access token stands for.
type AppToken struct {
	UserID   uuid.UUID
	ClientID uuid.UUID
	Scopes   []string
	// PseudID is the pseud the person chose to share with this app, if they
	// chose one. Anything the app posts is posted under it.
	PseudID *uuid.UUID
}

// AppTokenResolver turns an opaque access token into the person and app it
// was issued to. It must refuse expired and revoked tokens, apps that are not
// approved, and people who are not in good standing. The auth package
// provides the real one.
type AppTokenResolver func(ctx context.Context, token string) (*AppToken, error)

// AppLimiter counts one use of key and says whether it is allowed under
// limit uses per window. auth.Limiter.Allow has this shape.
type AppLimiter func(key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration)

// WithAppTokens lets a request's Bearer token be tried as an app token when
// it is not a login token, and sets the per-(app, person) rate limit.
// A limit of 0 or less uses DefaultAppRateLimit.
func WithAppTokens(resolve AppTokenResolver, allow AppLimiter, limit int) Option {
	return func(d *Deps) {
		d.app = resolve
		d.appAllow = allow
		if limit > 0 {
			d.appLimit = limit
		}
	}
}

type identity struct {
	user uuid.UUID
	app  *AppToken // nil for a login token
}

// appRoutes is the allowlist: "METHOD /full/route/pattern" to the scope an
// app needs. The pattern is the one the route was registered with, so
// /api/v1/works/:work_id and not a real id. It is the only way a route
// becomes reachable by an app.
//
// What is deliberately left out, and why, is in docs/API_SCOPES.md.
var appRoutes = map[string]string{
	// Works, chapters and their comments.
	"GET /api/v1/works":                                  "read",
	"GET /api/v1/works/:work_id":                         "read",
	"GET /api/v1/works/:work_id/chapters":                "read",
	"GET /api/v1/works/:work_id/chapters/:chapter_id":    "read",
	"GET /api/v1/works/:work_id/comments":                "read",
	"GET /api/v1/work/:work_id":                          "read",
	"GET /api/v1/work/:work_id/chapters":                 "read",
	"GET /api/v1/work/:work_id/chapters/:chapter_id":     "read",
	"GET /api/v1/work/:work_id/comments":                 "read",
	"POST /api/v1/works/:work_id/comments":               "comments:write",
	"POST /api/v1/work/:work_id/comments":                "comments:write",
	"POST /api/v1/works/:work_id/bookmark":               "bookmarks:manage",
	"GET /api/v1/series":                                 "read",
	"GET /api/v1/series/:series_id":                      "read",
	"GET /api/v1/series/:series_id/works":                "read",
	"GET /api/v1/collections":                            "read",
	"GET /api/v1/collections/:collection_id":             "read",
	"GET /api/v1/pseuds/:pseud_id/works":                 "read",
	"GET /api/v1/pseuds/:pseud_id/series":                "read",
	"GET /api/v1/pseuds/:pseud_id/bookmarks":             "read",
	"GET /api/v1/search/works":                           "read",
	"GET /api/v1/search/suggestions":                     "read",
	"GET /api/v1/tags":                                   "read",
	"GET /api/v1/tags/search":                            "read",
	"GET /api/v1/tags/autocomplete":                      "read",
	"GET /api/v1/tags/:tag_id":                           "read",
	"GET /api/v1/tags/:tag_id/related":                   "read",
	"GET /api/v1/tags/:tag_id/works":                     "read",
	"GET /api/v1/fandoms":                                "read",
	"GET /api/v1/fandoms/:fandom_id":                     "read",
	"GET /api/v1/fandoms/:fandom_id/tags":                "read",
	"GET /api/v1/fandoms/:fandom_id/characters":          "read",
	"GET /api/v1/fandoms/:fandom_id/relationships":       "read",
	"GET /api/v1/characters":                             "read",
	"GET /api/v1/characters/:character_id":               "read",
	"GET /api/v1/characters/:character_id/relationships": "read",
	"GET /api/v1/relationships":                          "read",
	"GET /api/v1/relationships/:relationship_id":         "read",
	"GET /api/v1/hierarchy/fandoms":                      "read",
	"GET /api/v1/hierarchy/characters":                   "read",
	"GET /api/v1/hierarchy/relationships":                "read",
	"GET /api/v1/stats/popular":                          "read",
	"GET /api/v1/stats/trending":                         "read",
	"GET /api/v1/stats/usage":                            "read",

	// What the person follows. These hold no pseud of the person's own.
	"GET /api/v1/subscriptions":               "read",
	"GET /api/v1/subscription-status":         "read",
	"GET /api/v1/user/tags/followed":          "read",
	"POST /api/v1/subscriptions":              "subscriptions:manage",
	"PUT /api/v1/subscriptions/:id":           "subscriptions:manage",
	"DELETE /api/v1/subscriptions/:id":        "subscriptions:manage",
	"POST /api/v1/user/tags/follow":           "subscriptions:manage",
	"DELETE /api/v1/user/tags/follow/:tag_id": "subscriptions:manage",
}

// appNeverPrefixes and appNeverRoutes are the hard denylist. They are checked
// before appRoutes, so a scope added to one of these routes by mistake still
// does nothing for an app. They cover everything that changes who someone is
// or how they sign in, every moderation and admin route, and the pages that
// list or change things across all of a person's pseuds.
var appNeverPrefixes = []string{
	"/auth/",             // login pages and the OAuth endpoints themselves
	"/api/v1/auth/",      // login, logout, register, password, sessions, email, me
	"/api/v1/oauth/",     // consent and the connected-apps page
	"/api/v1/apps",       // managing apps
	"/api/v1/admin/",     // every admin route
	"/api/v1/wrangling/", // tag wrangling
	"/api/v1/blocks",     // blocks and mutes: they name pseuds the person wants to avoid
	"/api/v1/mutes",
	"/api/v1/my/",          // dashboards that join all of a person's pseuds
	"/api/v1/export",       // personal downloads
	"/api/v1/notification", // notifications, preferences and rules
	"/api/v1/bookmarks",    // "my bookmarks" joins all pseuds
	"/ws",                  // live notifications
}

var appNeverRoutes = map[string]bool{
	"POST /api/v1/pseuds":                true, // pseud management
	"PUT /api/v1/pseuds/:pseud_id":       true,
	"DELETE /api/v1/pseuds/:pseud_id":    true,
	"GET /api/v1/works/:work_id/kudos":   true, // answers "have I given kudos", which is the person's own state
	"GET /api/v1/work/:work_id/kudos":    true,
	"GET /api/v1/works/:work_id/stats":   true, // owners get extra analytics
	"GET /api/v1/work/:work_id/stats":    true,
	"GET /api/v1/works/:work_id/authors": true,
	"GET /api/v1/works/:work_id/gifts":   true,
}

// appSkips are routes that never read who is logged in and do their own
// checks, so the gate leaves them alone: public discovery data, the health
// check, and /auth/userinfo, which is where an app presents its own token.
var appSkips = map[string]bool{
	"GET /health":                                 true,
	"GET /.well-known/openid-configuration":       true,
	"GET /.well-known/oauth-authorization-server": true,
	"GET /auth/jwks":                              true,
	"GET /auth/userinfo":                          true,
	"POST /auth/userinfo":                         true,
}

// AppRouteScope reports the scope an app needs for a route, and whether an
// app may use the route at all. route is "METHOD /pattern".
func AppRouteScope(route string) (scope string, ok bool) {
	if appNeverRoutes[route] {
		return "", false
	}
	path := route[strings.IndexByte(route, ' ')+1:]
	for _, p := range appNeverPrefixes {
		if strings.HasPrefix(path, p) {
			return "", false
		}
	}
	scope, ok = appRoutes[route]
	return scope, ok
}

// AppRoutes returns a copy of the allowlist, for tests and documentation.
func AppRoutes() map[string]string {
	out := make(map[string]string, len(appRoutes))
	for k, v := range appRoutes {
		out[k] = v
	}
	return out
}

// AppSkipped reports whether the gate ignores a route.
func AppSkipped(route string) bool { return appSkips[route] }

// IsApp reports whether the request came from an app token.
func IsApp(c *gin.Context) bool {
	k, _ := c.Get(CtxAuthKind)
	return k == "app"
}

// ReaderViewer is the one place a read handler learns who is looking and in
// what role. Every handler an app can reach uses it.
//
//   - A guest: (nil, false).
//   - A person at their own browser (login token): (their id, false). They get
//     the creator, owner and staff views the site gives them.
//   - An app acting for a person: (their id, true). The id is there so
//     members-only works show and the person's own blocks and mutes apply,
//     but asReader tells the handler to use the "as a reader" form of the
//     visibility functions (migration 050), and to pass nobody to anything
//     that only a creator or owner may see (see CreatorOnly). Otherwise the
//     difference would tell the app which hidden works the person wrote.
func ReaderViewer(c *gin.Context) (viewer *uuid.UUID, asReader bool) {
	if v, ok := c.Get("user_uuid"); ok {
		if id, ok := v.(uuid.UUID); ok {
			viewer = &id
		}
	}
	return viewer, viewer != nil && IsApp(c)
}

// CreatorOnly is the viewer to hand to a creator-only or owner-only check
// (is_work_creator, own_shadowed_kudos, blocked_silently and the like): the
// person themself for the site, nobody for an app.
func CreatorOnly(viewer *uuid.UUID, asReader bool) *uuid.UUID {
	if asReader {
		return nil
	}
	return viewer
}

// OwnerArg is CreatorOnly for use inside a SQL text: it gives the SQL for a
// value that is NULL when the query's as-reader parameter is true, and the
// viewer parameter otherwise. viewerN and readerN are the numbers of the $N
// placeholders that hold the viewer (uuid) and asReader (boolean). Use it
// wherever a query passes the viewer to a creator-only or owner-only
// function and also needs the viewer for the reader's own checks.
func OwnerArg(viewerN, readerN int) string {
	return fmt.Sprintf("(CASE WHEN $%d::boolean THEN NULL ELSE $%d::uuid END)", readerN, viewerN)
}

// AppPseud returns the pseud the person shared with the calling app, if any.
func AppPseud(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(CtxAppPseud)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// AppAccess is the gate. Install it once, for the whole router, with r.Use,
// after the middleware that should also see refused requests (logging,
// metrics). It works out who a Bearer token belongs to, and for an app token
// it applies the rate limit, the denylist, the allowlist and the scope check
// before any handler runs.
func (d *Deps) AppAccess(c *gin.Context) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		c.Next()
		return
	}
	route := c.Request.Method + " " + c.FullPath()
	if appSkips[route] {
		c.Next()
		return
	}
	token := strings.TrimPrefix(h, "Bearer ")

	// A login token comes first; it is the person at their own browser.
	if id, err := d.validate(token); err == nil {
		c.Set(ctxIdentity, identity{user: id})
		c.Next()
		return
	}
	if d.app == nil {
		c.Set(ctxIdentity, identity{})
		c.Next()
		return
	}
	at, err := d.app(c.Request.Context(), token)
	if err != nil || at == nil {
		c.Set(ctxIdentity, identity{})
		c.Next()
		return
	}

	if d.appAllow != nil {
		if ok, wait := d.appAllow("app:"+at.ClientID.String()+":"+at.UserID.String(), d.appLimit, d.appEvery); !ok {
			secs := int(wait.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited",
				"message": fmt.Sprintf("This app has made too many requests for this person. Try again in %d seconds.", secs)})
			return
		}
	}

	scope, ok := AppRouteScope(route)
	if !ok {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not_available_to_apps",
			"message": "Apps cannot use this part of the archive."})
		return
	}
	if !hasScope(at.Scopes, scope) {
		c.Header("WWW-Authenticate", fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q`, scope))
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient_scope",
			"message": "The person did not approve this app for the \"" + scope + "\" scope.", "scope": scope})
		return
	}
	c.Set(ctxIdentity, identity{user: at.UserID, app: at})
	if at.PseudID != nil {
		c.Set(CtxAppPseud, *at.PseudID)
	}
	c.Next()
}

func hasScope(have []string, want string) bool {
	for _, s := range have {
		if s == want {
			return true
		}
	}
	return false
}
