package works_test

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

// Ordinary requests write nothing to the log: no SQL text, no arguments, no
// "about to" and "DEBUG" chatter. Real errors still do (with context).
func TestWorksServiceLogsNothingForOrdinaryRequests(t *testing.T) {
	e := newFixEnv(t)
	writer, tok := e.user("writer")
	work := e.work(writer, "Quiet", "published")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	for _, path := range []string{
		"/api/v1/works", "/api/v1/works?q=Quiet", "/api/v1/works/" + work.String(),
		"/api/v1/work/" + work.String(), "/api/v1/work/" + work.String() + "/chapters",
		"/api/v1/work/" + work.String() + "/kudos", "/api/v1/work/" + work.String() + "/comments",
		e.pseudPage(writer, "works"), e.pseudPage(writer, "series"), e.pseudPage(writer, "bookmarks"),
		"/api/v1/series", "/api/v1/subscriptions",
	} {
		e.do("GET", path, tok, nil)
	}
	if out := strings.TrimSpace(buf.String()); out != "" {
		t.Errorf("ordinary requests logged:\n%s", out)
	}
}
