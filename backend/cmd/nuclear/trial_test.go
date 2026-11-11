package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/email"

	"github.com/gin-gonic/gin"
)

func TestEmailJobIsDroppedInTrialMode(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(nil)
	payload, _ := json.Marshal(email.Message{To: "who@example.org", Subject: "Reset your password", Text: "https://x/reset?token=SECRET"})

	// A nil Config would panic if the handler tried to send.
	if err := emailHandler(nil, true)(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	out := logs.String()
	if !strings.Contains(out, "trial mode: email not sent: Reset your password") {
		t.Errorf("log: %q", out)
	}
	for _, secret := range []string{"who@example.org", "SECRET"} {
		if strings.Contains(out, secret) {
			t.Errorf("log leaks %q: %q", secret, out)
		}
	}
	if err := emailHandler(nil, true)(context.Background(), []byte("not json")); err == nil {
		t.Error("a broken payload should still be an error")
	}
}

func TestSiteEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, on := range []bool{true, false} {
		r := gin.New()
		r.Use(platform.PublicCache())
		r.GET("/api/v1/site", siteInfo(on))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/site", nil))
		want := `{"trial_no_email":false}`
		if on {
			want = `{"trial_no_email":true}`
		}
		if w.Code != 200 || w.Body.String() != want {
			t.Errorf("trial=%v: %d %s", on, w.Code, w.Body)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "private, no-store" {
			t.Errorf("Cache-Control %q", cc)
		}
	}
}
