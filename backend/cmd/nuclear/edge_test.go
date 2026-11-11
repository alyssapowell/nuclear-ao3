package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"nuclear-ao3/internal/auth"
	"nuclear-ao3/internal/platform"
)

// edgeServer is the app's middleware and a few routes like the real ones,
// serving a small built website from a temporary folder.
func edgeServer(t *testing.T) *gin.Engine {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>website</html>"), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "app-1a2b.js"), []byte("js"), 0o644)
	os.WriteFile(filepath.Join(dir, "robots.txt"), []byte("robots"), 0o644)

	gin.SetMode(gin.TestMode)
	website := serveFrontend(dir)
	r := gin.New()
	r.Use(platform.PublicCache(), func(c *gin.Context) {
		c.Header("X-RateLimit-Remaining", "41") // as the rate limiter does
		c.Next()
	}, websitePaths(website))
	ok := func(c *gin.Context) { c.String(http.StatusOK, "app") }
	r.GET("/auth/authorize", ok)
	r.POST("/auth/token", ok)
	r.GET("/auth/consent/:id", ok)
	r.POST("/auth/register-client", ok)
	r.GET("/auth/consents", ok)
	r.GET("/api/v1/works", ok)
	r.NoRoute(website)
	return r
}

func get(r http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestAppAnswersOnlyTheAddressesCaddyWouldSendIt(t *testing.T) {
	r := edgeServer(t)
	for _, tc := range []struct {
		method, path, want string
	}{
		{"GET", "/auth/authorize", "app"},
		{"POST", "/auth/token", "app"},
		{"GET", "/auth/consent/abc", "app"},
		{"GET", "/api/v1/works", "app"},
		// Website pages, even where the app has a route of its own.
		{"POST", "/auth/register-client", "website"},
		{"GET", "/auth/consents", "website"},
		{"GET", "/auth/login", "website"},
		{"GET", "/works/123", "website"},
		{"GET", "/", "website"},
		{"GET", "/assets/", "website"}, // a folder: never a file listing
	} {
		w := get(r, tc.method, tc.path)
		if !strings.Contains(w.Body.String(), tc.want) {
			t.Errorf("%s %s: got %d %q, want %s", tc.method, tc.path, w.Code, w.Body.String(), tc.want)
		}
	}
	if w := get(r, "GET", "/api/v1/nope"); w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "website") {
		t.Errorf("unknown API address: got %d %q, want a JSON 404", w.Code, w.Body.String())
	}
}

func TestWebsiteFilesAreCachedLikeCaddyDoes(t *testing.T) {
	r := edgeServer(t)
	for path, want := range map[string]string{
		"/assets/app-1a2b.js": "public, max-age=31536000, immutable",
		"/robots.txt":         "no-cache",
		"/works/123":          "no-cache", // index.html
		"/":                   "no-cache",
	} {
		w := get(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status %d", path, w.Code)
		}
		if got := w.Header().Get("Cache-Control"); got != want {
			t.Errorf("%s: Cache-Control %q, want %q", path, got, want)
		}
		if got := w.Header().Get("X-RateLimit-Remaining"); got != "" {
			t.Errorf("%s: a shared file carries one visitor's rate limit (%q)", path, got)
		}
	}
	// API answers keep the API's own rules.
	if got := get(r, "GET", "/auth/authorize").Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("app answer: Cache-Control %q, want private, no-store", got)
	}
}

func pemKey(t *testing.T) string {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}))
}

func TestKeysCanBeGivenAsSettings(t *testing.T) {
	cur, old1, old2 := pemKey(t), pemKey(t), pemKey(t)
	t.Setenv("SIGNING_KEY_FILE", "")
	t.Setenv("SIGNING_KEY_PREVIOUS_FILES", "")
	t.Setenv("OIDC_SIGNING_KEY_FILE", "")
	// Platforms often store newlines as the two characters \n.
	t.Setenv("SIGNING_KEY", strings.ReplaceAll(cur, "\n", `\n`))
	t.Setenv("SIGNING_KEY_PREVIOUS", old1+old2)
	t.Setenv("OIDC_SIGNING_KEY", "")
	if err := keysFromEnv(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("SIGNING_KEY") != "" {
		t.Error("SIGNING_KEY is still in the environment")
	}
	f := os.Getenv("SIGNING_KEY_FILE")
	if info, err := os.Stat(f); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("key file %q: %v, mode %v; want readable only by its owner", f, err, info.Mode())
	}
	priv, prev, err := auth.LoadKeyFiles(f, os.Getenv("SIGNING_KEY_PREVIOUS_FILES"))
	if err != nil {
		t.Fatal(err)
	}
	want, _, _ := pemDecodeKey(cur)
	if !priv.Equal(want) || len(prev) != 2 {
		t.Errorf("loaded the wrong keys: same current key %v, %d previous", priv.Equal(want), len(prev))
	}
}

func pemDecodeKey(text string) (*rsa.PrivateKey, []byte, error) {
	b, rest := pem.Decode([]byte(text))
	k, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	return k, rest, err
}

func TestKeySettingsRefuseMistakes(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"key and file":   {"SIGNING_KEY": pemKey(t), "SIGNING_KEY_FILE": "/x.pem"},
		"not PEM":        {"SIGNING_KEY": "hunter2"},
		"two current":    {"SIGNING_KEY": pemKey(t) + pemKey(t)},
		"trailing words": {"OIDC_SIGNING_KEY": pemKey(t) + "oops"},
	} {
		for _, k := range []string{"SIGNING_KEY", "SIGNING_KEY_FILE", "SIGNING_KEY_PREVIOUS", "SIGNING_KEY_PREVIOUS_FILES", "OIDC_SIGNING_KEY", "OIDC_SIGNING_KEY_FILE"} {
			t.Setenv(k, env[k])
		}
		if err := keysFromEnv(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestKeySettingSurvivesWhatPlatformsDoToIt(t *testing.T) {
	key := pemKey(t)
	want, _, _ := pemDecodeKey(key)
	oneLine := func(s string) string { return strings.ReplaceAll(s, "\n", "") }
	for name, text := range map[string]string{
		"as pasted":           key,
		"newlines as \\n":     strings.ReplaceAll(key, "\n", `\n`),
		"newlines as spaces":  strings.ReplaceAll(key, "\n", " "),
		"newlines removed":    oneLine(key),
		"windows newlines":    strings.ReplaceAll(key, "\n", "\r\n"),
		"in quotes":           `"` + key + `"`,
		"one line of base64":  base64.StdEncoding.EncodeToString([]byte(key)),
		"base64, then quoted": `'` + base64.StdEncoding.EncodeToString([]byte(key)) + `'`,
	} {
		blocks, err := splitPEM(text)
		if err != nil || len(blocks) != 1 {
			t.Errorf("%s: %v (%d keys)", name, err, len(blocks))
			continue
		}
		got, _, err := pemDecodeKey(string(blocks[0]))
		if err != nil || !got.Equal(want) {
			t.Errorf("%s: a different key came out (%v)", name, err)
		}
	}
	// Two old keys run together on one line, as SIGNING_KEY_PREVIOUS.
	if blocks, err := splitPEM(oneLine(pemKey(t) + pemKey(t))); err != nil || len(blocks) != 2 {
		t.Errorf("two keys on one line: %v (%d keys)", err, len(blocks))
	}
	// A cut-off key, or a key with extra text, is refused, and the error
	// never repeats any of it.
	for name, text := range map[string]string{
		"cut off":    key[:len(key)/2],
		"extra text": key + "SIGNING_KEY=",
		"garbage":    "hunter2",
	} {
		_, err := splitPEM(text)
		if err == nil {
			t.Errorf("%s: accepted", name)
		} else if strings.Contains(err.Error(), "MII") || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("%s: the error repeats the secret: %v", name, err)
		}
	}
}
