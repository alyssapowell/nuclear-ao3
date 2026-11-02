package platform

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The access log must never contain query strings: they can carry login
// tokens (?token= on the live-notifications connection) and OAuth codes.
func TestLoggerLeavesOutQueryStrings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	old := gin.DefaultWriter
	gin.DefaultWriter = &buf
	defer func() { gin.DefaultWriter = old }()

	r := gin.New()
	r.Use(Logger())
	r.GET("/ws", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/ws?token=secret-login-token&code=abc", nil))

	out := buf.String()
	if !strings.Contains(out, " /ws ") {
		t.Fatalf("path missing from log line: %q", out)
	}
	if strings.Contains(out, "secret-login-token") || strings.Contains(out, "code=abc") || strings.Contains(out, "?") {
		t.Errorf("query string written to the log: %q", out)
	}
}
