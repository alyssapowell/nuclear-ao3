package main

import (
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"nuclear-ao3/internal/platform"
)

// oauthPaths are the only addresses under /auth that belong to the app: the
// OAuth2 and OpenID Connect endpoints other apps call. Every other /auth
// address (login, register, activate, reset password) is a page of the
// website. This is the same list as "@oauth" in the Caddyfile; it is kept
// here too so the app is safe to put on the internet without Caddy in front,
// as on a hosting platform. /auth/register-client and the old /auth/consents
// and /auth/authorized-applications addresses are left out on purpose: the
// website does those through /api.
var oauthPaths = map[string]bool{
	"/.well-known/openid-configuration":       true,
	"/.well-known/oauth-authorization-server": true,
	"/auth/authorize":                         true,
	"/auth/token":                             true,
	"/auth/userinfo":                          true,
	"/auth/introspect":                        true,
	"/auth/revoke":                            true,
	"/auth/jwks":                              true,
}

// isOAuthPath reports whether path is one of the app's OAuth addresses.
func isOAuthPath(path string) bool {
	return oauthPaths[path] || strings.HasPrefix(path, "/auth/consent/")
}

// websitePaths hands every /auth address that isn't an OAuth endpoint to the
// website (or a 404 when the app isn't serving it), even if the app has a
// route there, so the app answers only the addresses Caddy would send it.
func websitePaths(website gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/auth/") && !isOAuthPath(p) && c.FullPath() != "" {
			website(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// notFound is the website handler when the app isn't serving the website.
func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

// serveFrontend serves the built single-page app from dir. Unknown paths get
// index.html so the browser-side router can handle them; unknown /api paths
// get a JSON 404. Files under /assets have a fingerprint in their name and
// never change, so they may be kept for a year; everything else, index.html
// above all, must be checked every time, or visitors keep running an old
// version after an upgrade. (These are the Caddyfile's rules too.)
func serveFrontend(dir string) gin.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.Set(platform.WebsiteFileKey, true)
		if info, err := os.Stat(filepath.Join(dir, filepath.Clean("/"+p))); err != nil || info.IsDir() {
			c.Header("Cache-Control", "no-cache")
			c.File(filepath.Join(dir, "index.html"))
			return
		}
		if strings.HasPrefix(p, "/assets/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		fs.ServeHTTP(c.Writer, c.Request)
	}
}

// keysFromEnv lets the signing keys be given as settings instead of files,
// for hosting platforms that have a place for secret settings but no way to
// put a file in the container:
//
//	SIGNING_KEY            the PEM text of the login key (instead of SIGNING_KEY_FILE)
//	SIGNING_KEY_PREVIOUS   the PEM text of old keys, one after another
//	                       (instead of SIGNING_KEY_PREVIOUS_FILES)
//	OIDC_SIGNING_KEY       the PEM text of the ID-token key (instead of
//	                       OIDC_SIGNING_KEY_FILE)
//
// Each is written to a file only its owner can read, in a new private
// folder, and the matching *_FILE setting is pointed at it, so everything
// after this reads keys the usual way. The key is still made with "nuclear
// keys generate" and saved somewhere safe by an admin; this only changes how
// it reaches the app. Giving both a key and its file is refused, so it is
// never unclear which one is in use.
func keysFromEnv() error {
	pairs := []struct{ text, file string }{
		{"SIGNING_KEY", "SIGNING_KEY_FILE"},
		{"SIGNING_KEY_PREVIOUS", "SIGNING_KEY_PREVIOUS_FILES"},
		{"OIDC_SIGNING_KEY", "OIDC_SIGNING_KEY_FILE"},
	}
	var dir string
	for _, p := range pairs {
		text := strings.TrimSpace(os.Getenv(p.text))
		if text == "" {
			continue
		}
		if strings.TrimSpace(os.Getenv(p.file)) != "" {
			return fmt.Errorf("both %s and %s are set; use one", p.text, p.file)
		}
		blocks, err := splitPEM(text)
		if err != nil {
			return fmt.Errorf("%s: %w", p.text, err)
		}
		if p.text != "SIGNING_KEY_PREVIOUS" && len(blocks) != 1 {
			return fmt.Errorf("%s must hold exactly one key, not %d", p.text, len(blocks))
		}
		if dir == "" {
			if dir, err = os.MkdirTemp("", "nuclear-keys-"); err != nil {
				return err
			}
		}
		var files []string
		for i, b := range blocks {
			f := filepath.Join(dir, fmt.Sprintf("%s-%d.pem", strings.ToLower(p.text), i))
			if err := os.WriteFile(f, b, 0o600); err != nil {
				return err
			}
			files = append(files, f)
		}
		os.Setenv(p.file, strings.Join(files, ","))
		// The key text is not needed again; keep it out of anything that
		// later reads or prints the environment.
		os.Unsetenv(p.text)
	}
	return nil
}

// splitPEM splits text into its PEM blocks, re-encoded one per item.
// Platforms often turn newlines in a setting into the two characters \n, so
// those are turned back first.
func splitPEM(text string) ([][]byte, error) {
	rest := []byte(strings.ReplaceAll(text, `\n`, "\n"))
	var out [][]byte
	for {
		var b *pem.Block
		b, rest = pem.Decode(rest)
		if b == nil {
			break
		}
		out = append(out, pem.EncodeToMemory(b))
	}
	if strings.TrimSpace(string(rest)) != "" || len(out) == 0 {
		return nil, fmt.Errorf("is not PEM text (it should start with -----BEGIN)")
	}
	return out, nil
}
