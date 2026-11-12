package main

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
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

var paramSegment = regexp.MustCompile(`:[A-Za-z_]+`)

// TestFrontendCallsMountedRoutes fails when the frontend calls an address the
// app does not serve (a leftover from the old multi-service layout, say).
//
// The list of addresses is frontend_api_paths.txt, generated from the source
// by scripts/frontend-api-paths.mjs. After adding or removing a call in
// frontend/src, regenerate it:
//
//	node scripts/frontend-api-paths.mjs
//
// A path is accepted if any method is mounted there, with every :name and
// ${...} treated as one wildcard segment.
func TestFrontendCallsMountedRoutes(t *testing.T) {
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
	mounted := map[string]bool{}
	for _, rt := range r.Routes() {
		mounted[paramSegment.ReplaceAllString(rt.Path, ":p")] = true
	}

	raw, err := os.ReadFile("frontend_api_paths.txt")
	if err != nil {
		t.Fatalf("read the list (run node scripts/frontend-api-paths.mjs): %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) < 20 {
		t.Fatalf("only %d calls listed; the generator is probably broken", len(lines))
	}
	for _, line := range lines {
		path, file, _ := strings.Cut(line, "\t")
		if !mounted[paramSegment.ReplaceAllString(path, ":p")] {
			t.Errorf("%s calls %s, which the app does not serve", file, path)
		}
	}
}

// TestFrontendPathListIsCurrent reruns the generator in --check mode, so the
// checked-in list cannot go stale and let a dead call through. Skipped where
// node is not installed.
func TestFrontendPathListIsCurrent(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	out, err := exec.Command(node, "../../../scripts/frontend-api-paths.mjs", "--check").CombinedOutput()
	if err != nil {
		t.Fatalf("frontend_api_paths.txt is out of date; run: node scripts/frontend-api-paths.mjs\n%s", out)
	}
}
