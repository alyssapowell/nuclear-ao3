package platform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/shared/models"
)

// A scope added to a denylisted route by mistake must still do nothing.
func TestDenylistBeatsAllowlist(t *testing.T) {
	for _, route := range []string{
		"GET /api/v1/auth/me", "POST /api/v1/auth/login", "GET /api/v1/admin/audit",
		"GET /api/v1/my/works", "GET /api/v1/apps", "POST /auth/token", "GET /ws",
		"POST /api/v1/pseuds", "GET /api/v1/notifications", "GET /api/v1/bookmarks",
		"GET /api/v1/blocks", "GET /api/v1/export/:id/download",
	} {
		appRoutes[route] = "read"
		if _, ok := AppRouteScope(route); ok {
			t.Errorf("%s reached an app although it is on the denylist", route)
		}
		delete(appRoutes, route)
	}
}

// Every scope the table names must exist and be one an app can hold.
func TestAllowlistScopesAreGrantable(t *testing.T) {
	for route, scope := range AppRoutes() {
		if !models.IsGrantableScope(scope) {
			t.Errorf("%s needs %q, which an app cannot be granted", route, scope)
		}
		if _, ok := AppRouteScope(route); !ok {
			t.Errorf("%s is in the allowlist but also on the denylist", route)
		}
	}
}

func gateRouter(d *Deps, hit *int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(d.AppAccess)
	ok := func(c *gin.Context) { *hit++; c.JSON(200, gin.H{"kind": c.GetString(CtxAuthKind)}) }
	r.GET("/api/v1/works", d.OptionalUser, ok)
	r.POST("/api/v1/works/:work_id/comments", d.OptionalUser, ok)
	r.GET("/api/v1/secret", d.RequireUser, ok) // not in the allowlist
	return r
}

func call(r *gin.Engine, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGateDecisions(t *testing.T) {
	user, client := uuid.New(), uuid.New()
	var seen string
	d := NewDeps(nil, func(tok string) (uuid.UUID, error) {
		if tok == "login" {
			return user, nil
		}
		return uuid.Nil, errors.New("no")
	}, WithAppTokens(func(_ context.Context, tok string) (*AppToken, error) {
		seen = tok
		switch tok {
		case "app-read":
			return &AppToken{UserID: user, ClientID: client, Scopes: []string{"read"}}, nil
		}
		return nil, errors.New("no")
	}, nil, 0))
	hit := 0
	r := gateRouter(d, &hit)

	if w := call(r, "GET", "/api/v1/secret", "login"); w.Code != 200 {
		t.Errorf("login token on an unlisted route: %d", w.Code)
	}
	if w := call(r, "GET", "/api/v1/works", "app-read"); w.Code != 200 {
		t.Errorf("app with the scope: %d", w.Code)
	}
	before := hit
	if w := call(r, "GET", "/api/v1/secret", "app-read"); w.Code != 403 || hit != before {
		t.Errorf("unlisted route must refuse an app before the handler: %d", w.Code)
	}
	w := call(r, "POST", "/api/v1/works/abc/comments", "app-read")
	if w.Code != 403 || w.Header().Get("WWW-Authenticate") != `Bearer error="insufficient_scope", scope="comments:write"` {
		t.Errorf("missing scope: %d %q", w.Code, w.Header().Get("WWW-Authenticate"))
	}
	if w := call(r, "GET", "/api/v1/secret", "junk"); w.Code != 401 {
		t.Errorf("unknown token on a login route: %d", w.Code)
	}
	if w := call(r, "GET", "/api/v1/nowhere", "app-read"); w.Code != 404 && w.Code != 403 {
		t.Errorf("unmatched route: %d", w.Code)
	}
	_ = seen
}

// Without the resolver, an app token is never believed.
func TestAppTokensIgnoredWithoutResolver(t *testing.T) {
	d := NewDeps(nil, func(string) (uuid.UUID, error) { return uuid.Nil, errors.New("no") })
	hit := 0
	r := gateRouter(d, &hit)
	if w := call(r, "GET", "/api/v1/secret", "app-read"); w.Code != 401 {
		t.Errorf("got %d", w.Code)
	}
}

// Without the gate, RequireUser still only accepts login tokens.
func TestNoGateMeansLoginOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := uuid.New()
	d := NewDeps(nil, func(tok string) (uuid.UUID, error) {
		if tok == "login" {
			return user, nil
		}
		return uuid.Nil, errors.New("no")
	}, WithAppTokens(func(context.Context, string) (*AppToken, error) {
		return &AppToken{UserID: user, Scopes: []string{"read"}}, nil
	}, nil, 0))
	r := gin.New()
	r.GET("/x", d.RequireUser, func(c *gin.Context) { c.Status(204) })
	if w := call(r, "GET", "/x", "app"); w.Code != 401 {
		t.Errorf("app token without the gate: %d", w.Code)
	}
	if w := call(r, "GET", "/x", "login"); w.Code != 204 {
		t.Errorf("login token: %d", w.Code)
	}
}

func TestGateRateLimit(t *testing.T) {
	user, client := uuid.New(), uuid.New()
	counts := map[string]int{}
	allow := func(key string, limit int, _ time.Duration) (bool, time.Duration) {
		counts[key]++
		return counts[key] <= limit, 7 * time.Second
	}
	d := NewDeps(nil, func(string) (uuid.UUID, error) { return uuid.Nil, errors.New("no") },
		WithAppTokens(func(_ context.Context, tok string) (*AppToken, error) {
			return &AppToken{UserID: user, ClientID: client, Scopes: []string{"read"}}, nil
		}, allow, 2))
	hit := 0
	r := gateRouter(d, &hit)
	var codes []int
	for i := 0; i < 3; i++ {
		w := call(r, "GET", "/api/v1/works", "x")
		codes = append(codes, w.Code)
		if w.Code == http.StatusTooManyRequests && w.Header().Get("Retry-After") != "8" {
			t.Errorf("Retry-After = %q", w.Header().Get("Retry-After"))
		}
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 {
		t.Errorf("codes %v", codes)
	}
	if len(counts) != 1 {
		t.Errorf("the key must be one (app, person) pair, got %v", counts)
	}
}
