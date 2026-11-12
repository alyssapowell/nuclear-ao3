package platform

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func cacheRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(), PublicCache())
	ok := func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }
	r.GET("/api/v1/works/:work_id", func(c *gin.Context) {
		switch c.Param("work_id") {
		case "missing":
			c.JSON(404, gin.H{"error": "not found"})
		case "hidden":
			c.JSON(403, gin.H{"error": "no"})
		case "moved":
			c.Redirect(301, "/api/v1/work/x")
		case "ratelimited":
			c.Header("X-RateLimit-Remaining", "5")
			c.JSON(200, gin.H{})
		case "cookie":
			c.SetCookie("s", "1", 60, "/", "", false, true)
			c.JSON(200, gin.H{})
		case "strict":
			c.Header("Cache-Control", "no-store")
			c.JSON(200, gin.H{})
		case "short":
			c.Header("Cache-Control", "public, max-age=10")
			c.JSON(200, gin.H{})
		case "long":
			c.Header("Cache-Control", "public, max-age=86400")
			c.JSON(200, gin.H{})
		case "string":
			c.String(200, "hi")
		case "empty":
			c.Status(200)
		default:
			ok(c)
		}
	})
	// Gin does not answer HEAD from GET routes on its own; a route that is
	// registered for HEAD must get the same treatment.
	r.HEAD("/api/v1/works/:work_id", ok)
	r.GET("/api/v1/auth/me", ok)
	r.GET("/api/v1/notifications", ok)
	r.GET("/api/v1/admin/works", ok)
	r.GET("/api/v1/works/:work_id/kudos", ok)
	r.POST("/api/v1/works/:work_id/comments", ok)
	r.GET("/unlisted", ok)
	return r
}

func do(r *gin.Engine, method, path string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPublicCache(t *testing.T) {
	bearer := map[string]string{"Authorization": "Bearer abc"}
	cases := []struct {
		name   string
		method string
		path   string
		hdr    map[string]string
		status int
		cc     string
	}{
		{"logged-out 200 is public", "GET", "/api/v1/works/1", nil, 200, publicCacheControl},
		{"HEAD is like GET", "HEAD", "/api/v1/works/1", nil, 200, publicCacheControl},
		{"logged in is private", "GET", "/api/v1/works/1", bearer, 200, privateCacheControl},
		{"invalid token still private", "GET", "/api/v1/works/1", map[string]string{"Authorization": "x"}, 200, privateCacheControl},
		{"404", "GET", "/api/v1/works/missing", nil, 404, privateCacheControl},
		{"403", "GET", "/api/v1/works/hidden", nil, 403, privateCacheControl},
		{"redirect", "GET", "/api/v1/works/moved", nil, 301, privateCacheControl},
		{"unmatched route", "GET", "/nope", nil, 404, privateCacheControl},
		{"auth/me logged out", "GET", "/api/v1/auth/me", nil, 200, privateCacheControl},
		{"auth/me with token", "GET", "/api/v1/auth/me", bearer, 200, privateCacheControl},
		{"notifications", "GET", "/api/v1/notifications", nil, 200, privateCacheControl},
		{"admin", "GET", "/api/v1/admin/works", bearer, 200, privateCacheControl},
		{"kudos varies by IP", "GET", "/api/v1/works/1/kudos", nil, 200, privateCacheControl},
		{"not allowlisted", "GET", "/unlisted", nil, 200, privateCacheControl},
		{"POST", "POST", "/api/v1/works/1/comments", nil, 200, privateCacheControl},
		{"set-cookie", "GET", "/api/v1/works/cookie", nil, 200, privateCacheControl},
		{"handler stricter", "GET", "/api/v1/works/strict", nil, 200, "no-store"},
		{"handler shorter", "GET", "/api/v1/works/short", nil, 200, "public, max-age=10"},
		{"handler longer is capped", "GET", "/api/v1/works/long", nil, 200, publicCacheControl},
		{"c.String", "GET", "/api/v1/works/string", nil, 200, publicCacheControl},
		{"status only", "GET", "/api/v1/works/empty", nil, 200, publicCacheControl},
	}
	r := cacheRouter()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(r, tc.method, tc.path, tc.hdr)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			if got := w.Header().Get("Cache-Control"); got != tc.cc {
				t.Errorf("Cache-Control = %q, want %q", got, tc.cc)
			}
			if !strings.Contains(w.Header().Get("Vary"), "Authorization") {
				t.Errorf("Vary = %q, want Authorization", w.Header().Get("Vary"))
			}
		})
	}
}

func TestPublicCacheKeepsOtherVaryAndDropsRateLimit(t *testing.T) {
	r := cacheRouter()
	w := do(r, "GET", "/api/v1/works/1", map[string]string{"Origin": "http://localhost:3000"})
	vary := strings.Join(w.Header().Values("Vary"), ",")
	if !strings.Contains(vary, "Origin") || !strings.Contains(vary, "Authorization") {
		t.Errorf("Vary = %q, want Origin and Authorization", vary)
	}
	w = do(r, "GET", "/api/v1/works/ratelimited", nil)
	if w.Header().Get("X-RateLimit-Remaining") != "" || w.Header().Get("Cache-Control") != publicCacheControl {
		t.Errorf("rate-limit header kept on public response: %v", w.Header())
	}
}

func TestPublicRoutesHoldNothingPersonal(t *testing.T) {
	for p := range publicRoutes {
		for _, bad := range []string{"/auth", "/admin", "/my/", "notification", "subscription", "export", "kudos", "/oauth", "/user/", "/hit"} {
			if strings.Contains(p, bad) {
				t.Errorf("%s must not be cacheable", p)
			}
		}
		if !strings.HasPrefix(p, "/api/v1/") {
			t.Errorf("%s is outside /api/v1", p)
		}
	}
}
