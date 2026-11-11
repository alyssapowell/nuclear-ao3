package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/notify"
	"nuclear-ao3/internal/platform"
	"nuclear-ao3/internal/search"
	"nuclear-ao3/internal/tags"
	"nuclear-ao3/internal/works"
)

// TestRoutesDoNotCollide mounts every area on one router. gin panics if two
// areas register the same path, so this catches collisions without a database.
func TestRoutesDoNotCollide(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deps := platform.NewDeps(nil, func(string) (uuid.UUID, error) { return uuid.Nil, nil })
	r := gin.New()
	mountRoutes(r, deps, services{
		auth:   &auth.AuthService{},
		works:  &works.WorkService{},
		tags:   &tags.TagService{},
		notify: &notify.NotificationService{},
		search: &search.Service{},
	})
	t.Logf("%d routes mounted", len(r.Routes()))
	mounted := map[string]bool{}
	for _, rt := range r.Routes() {
		mounted[rt.Method+" "+rt.Path] = true
	}
	// One or two routes from each area, so a forgotten Mount call shows up.
	for _, want := range []string{
		"GET /health",
		"POST /api/v1/auth/login",
		"POST /api/v1/works",
		"GET /api/v1/works/:work_id",
		"GET /api/v1/tags/autocomplete",
		"GET /api/v1/notifications",
		"GET /api/v1/search/works",
		"GET /api/v1/search/suggestions",
	} {
		if !mounted[want] {
			t.Errorf("route %s is not mounted", want)
		}
	}
}
