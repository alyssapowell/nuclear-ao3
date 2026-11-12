package platform

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// The two Cache-Control values the API hands out. A CDN in front of the site
// may keep the first one for a minute (and serve it a little stale while it
// fetches a fresh copy). The second tells every cache, shared or in the
// browser, to keep nothing.
const (
	publicCacheControl  = "public, max-age=60, stale-while-revalidate=300"
	privateCacheControl = "private, no-store"
	publicMaxAge        = 60
)

// publicRoutes are the only routes whose logged-out answer may be cached.
// Each is matched by its route pattern (what gin reports as FullPath), not by
// the address the visitor typed.
//
// A route belongs here only if, for a request with no Authorization header,
// the answer depends on nothing but the address and query string: the same
// for every logged-out visitor, with no per-IP or per-session content and no
// side effects. Left out on purpose: everything under /auth, /oauth, /admin,
// notifications, subscriptions and /my; work kudos (it says whether
// this visitor's IP already gave kudos); work stats; and a user's works and
// series lists.
//
// Reading a chapter counts a view, so a cached answer is not counted. See
// docs/HOSTING.md.
var publicRoutes = buildPublicRoutes()

func buildPublicRoutes() map[string]bool {
	m := map[string]bool{}
	add := func(paths ...string) {
		for _, p := range paths {
			m[p] = true
		}
	}
	// Works: the same pages are reachable under /works (numeric or UUID ids)
	// and /work (UUID only).
	for _, base := range []string{"/api/v1/works", "/api/v1/work"} {
		// Views are counted by a separate POST .../hit from the work page,
		// so the chapters themselves can be cached.
		add(base+"/:work_id",
			base+"/:work_id/chapters",
			base+"/:work_id/chapters/:chapter_id",
			base+"/:work_id/comments",
			// A download is made from the same rows a logged-out reader of
			// the page sees, and nothing else (no view is counted).
			base+"/:work_id/download.:format")
	}
	add("/api/v1/works") // the browse list
	add("/api/v1/series", "/api/v1/series/:series_id", "/api/v1/series/:series_id/works")
	add("/api/v1/collections", "/api/v1/collections/:collection_id", "/api/v1/collections/:collection_id/works")
	add("/api/v1/users/:user_id/bookmarks") // public bookmarks only when logged out
	add("/api/v1/search/works", "/api/v1/search/suggestions")
	// Tag pages and tag lists.
	add("/api/v1/tags", "/api/v1/tags/search", "/api/v1/tags/autocomplete",
		"/api/v1/tags/:tag_id", "/api/v1/tags/:tag_id/related", "/api/v1/tags/:tag_id/works",
		"/api/v1/fandoms", "/api/v1/fandoms/:fandom_id", "/api/v1/fandoms/:fandom_id/tags",
		"/api/v1/fandoms/:fandom_id/characters", "/api/v1/fandoms/:fandom_id/relationships",
		"/api/v1/characters", "/api/v1/characters/:character_id", "/api/v1/characters/:character_id/relationships",
		"/api/v1/relationships", "/api/v1/relationships/:relationship_id",
		"/api/v1/hierarchy/fandoms", "/api/v1/hierarchy/characters", "/api/v1/hierarchy/relationships",
		"/api/v1/stats/popular", "/api/v1/stats/trending")
	return m
}

// PublicCache sets Cache-Control and Vary on every response so a CDN can keep
// logged-out reads and nothing else.
//
// A response is marked public (max-age 60, stale-while-revalidate 300) only
// when all of these hold: the method is GET or HEAD, the request carries no
// Authorization header, the matched route is in publicRoutes, the status is
// 200, and the handler did not set a Set-Cookie header. Everything else gets
// "private, no-store". The API uses no auth cookies, so only the
// Authorization header is checked. Every response says "Vary: Authorization".
//
// The decision is made at the moment the status line is about to be sent,
// because only then are the final status and the headers set by the handler
// known.
// WebsiteFileKey marks a response as one of the website's built files (set
// by the file server), whose Cache-Control the file server decides.
const WebsiteFileKey = "platform.websiteFile"

func PublicCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &cacheWriter{ResponseWriter: c.Writer, c: c}
		c.Writer = w
		c.Next()
		// A handler that only set a status and wrote no body has sent
		// nothing yet; decide before gin sends the bare status line.
		w.decide()
	}
}

// cacheWriter wraps gin's writer and fixes the cache headers just before the
// first byte of the response is sent. Gin only records the status when a
// handler calls WriteHeader; the real send happens in WriteHeaderNow, Write,
// WriteString or Flush, so those all run the decision first.
type cacheWriter struct {
	gin.ResponseWriter
	c       *gin.Context
	decided bool
}

func (w *cacheWriter) decide() {
	if w.decided {
		return
	}
	w.decided = true
	h := w.Header()

	addVary(h, "Authorization")

	canCache := (w.c.Request.Method == http.MethodGet || w.c.Request.Method == http.MethodHead) &&
		w.c.Request.Header.Get("Authorization") == "" &&
		publicRoutes[w.c.FullPath()] &&
		w.Status() == http.StatusOK &&
		len(h.Values("Set-Cookie")) == 0
	// The website's own files carry their rules from serveFrontend: they
	// are the same for everyone and hold no account data. They still go
	// through the cleanup below, like any shared answer.
	websiteFile := w.c.GetBool(WebsiteFileKey) && h.Get("Cache-Control") != "" &&
		w.Status() == http.StatusOK && len(h.Values("Set-Cookie")) == 0
	if websiteFile {
		stripPerVisitorHeaders(h)
		return
	}
	if !canCache {
		h.Set("Cache-Control", privateCacheControl)
		return
	}

	// A handler may ask for less caching than we would give, never more.
	if own := h.Get("Cache-Control"); own != "" && strictAtLeast(own) {
		return
	}
	h.Set("Cache-Control", publicCacheControl)
	stripPerVisitorHeaders(h)
}

// stripPerVisitorHeaders removes headers that describe one visitor from an
// answer that may be shared. Rate-limit headers describe one visitor's IP.
func stripPerVisitorHeaders(h http.Header) {
	for name := range h {
		if strings.HasPrefix(strings.ToLower(name), "x-ratelimit-") {
			h.Del(name)
		}
	}
}

func (w *cacheWriter) WriteHeaderNow() {
	w.decide()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *cacheWriter) Write(b []byte) (int, error) {
	w.decide()
	return w.ResponseWriter.Write(b)
}

func (w *cacheWriter) WriteString(s string) (int, error) {
	w.decide()
	return w.ResponseWriter.WriteString(s)
}

func (w *cacheWriter) Flush() {
	w.decide()
	w.ResponseWriter.Flush()
}

// addVary adds token to the Vary header unless it is already listed, keeping
// what other middleware (CORS adds Origin) put there.
func addVary(h http.Header, token string) {
	for _, v := range h.Values("Vary") {
		for _, t := range strings.Split(v, ",") {
			if t = strings.TrimSpace(t); t == "*" || strings.EqualFold(t, token) {
				return
			}
		}
	}
	h.Add("Vary", token)
}

// strictAtLeast reports whether a handler's own Cache-Control keeps a cache
// from holding the response longer than publicCacheControl would: it forbids
// storing or reuse, is private, or has a max-age under our own.
func strictAtLeast(value string) bool {
	// A long shared-cache lifetime outweighs a short max-age beside it.
	for _, part := range strings.Split(value, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if n, err := strconv.Atoi(strings.TrimPrefix(part, "s-maxage=")); strings.HasPrefix(part, "s-maxage=") && (err != nil || n >= publicMaxAge) {
			return false
		}
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		switch {
		case part == "no-store", part == "no-cache", part == "private":
			return true
		case strings.HasPrefix(part, "s-maxage="), strings.HasPrefix(part, "max-age="):
			n, err := strconv.Atoi(part[strings.Index(part, "=")+1:])
			if err != nil || n < publicMaxAge {
				return true
			}
		}
	}
	return false
}
