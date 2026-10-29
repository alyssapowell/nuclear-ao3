package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/export"
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
		export: &export.ExportService{},
		search: &search.SearchService{},
	})
	t.Logf("%d routes mounted", len(r.Routes()))
	if n := len(r.Routes()); n < 200 {
		t.Fatalf("expected the full API to be mounted, got only %d routes", n)
	}
}
