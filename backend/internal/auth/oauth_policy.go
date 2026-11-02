package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"nuclear-ao3/shared/models"
)

// The rules the OAuth provider enforces in code. The database repeats the
// ones it can (migration 045), but nothing here relies on struct tags: the
// validate: tags in the models are never run.

const (
	// Access tokens live between 5 minutes and an hour.
	accessTTLMin     = 300
	accessTTLMax     = 3600
	accessTTLDefault = 3600

	// A refresh token lives up to 30 days.
	refreshTTLMin     = 3600
	refreshTTLMax     = 30 * 24 * 3600
	refreshTTLDefault = refreshTTLMax

	// However often a family of refresh tokens is used, it stops working 90
	// days after the sign-in that started it.
	refreshChainMax = 90 * 24 * time.Hour

	// An authorization code is good for one minute.
	authCodeTTL = 60 * time.Second

	// Longest values accepted from an app.
	maxStateLen       = 512
	maxNonceLen       = 512
	maxLoginHintLen   = 512
	maxRedirectURILen = 2048
	maxURLFieldLen    = 2048
	maxClientNameLen  = 100
	maxDescriptionLen = 500
	maxRedirectURIs   = 10
)

// Only these are ever accepted for a client.
var (
	allowedGrantTypes    = []string{"authorization_code", "refresh_token"}
	allowedResponseTypes = []string{"code"}
)

func clampAccessTTL(seconds int) int {
	if seconds <= 0 {
		return accessTTLDefault
	}
	return clampInt(seconds, accessTTLMin, accessTTLMax)
}

func clampRefreshTTL(seconds int) int {
	if seconds <= 0 {
		return refreshTTLDefault
	}
	return clampInt(seconds, refreshTTLMin, refreshTTLMax)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// hashSecret is how tokens and codes are stored: SHA-256, in hex. They are
// long random values, so a fast hash is enough; the database never holds
// anything an attacker could present to the server.
func hashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------
// PKCE (S256 only)
// ---------------------------------------------------------------------

var (
	// A challenge is the base64url of a SHA-256 digest: always 43 characters.
	pkceChallengeRE = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
	// A verifier is 43 to 128 characters from the unreserved set (RFC 7636).
	pkceVerifierRE = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)
)

func validPKCEChallenge(method, challenge string) bool {
	return method == "S256" && pkceChallengeRE.MatchString(challenge)
}

func validPKCEVerifier(verifier string) bool {
	return pkceVerifierRE.MatchString(verifier)
}

// validatePKCE checks a verifier against the stored challenge. Only S256 is
// supported, and the comparison takes the same time whatever matches.
func (as *AuthService) validatePKCE(codeChallenge, codeChallengeMethod, codeVerifier string) bool {
	if codeChallengeMethod != "S256" || !validPKCEVerifier(codeVerifier) {
		return false
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(codeChallenge)) == 1
}

// ---------------------------------------------------------------------
// Scopes, grant types and response types
// ---------------------------------------------------------------------

// parseScopes splits a space-separated scope string, dropping repeats.
func parseScopes(raw string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range strings.Fields(raw) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// scopesAllowed reports whether every scope can be granted to this client:
// it is a real scope, it is not admin-only, and the client was registered
// with it. An empty list is not allowed.
func scopesAllowed(requested, clientScopes []string) bool {
	if len(requested) == 0 {
		return false
	}
	for _, s := range requested {
		if !models.IsGrantableScope(s) || !contains(clientScopes, s) {
			return false
		}
	}
	return true
}

// cleanChoices checks a list against an allowlist and removes repeats. An
// empty list is an error; so is any value not on the list.
func cleanChoices(in, allowed []string) ([]string, bool) {
	if len(in) == 0 {
		return nil, false
	}
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !contains(allowed, v) {
			return nil, false
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out, true
}

// cleanScopes checks scopes named at registration or by an admin: each must
// be a real scope that apps may ask for.
func cleanScopes(in []string) ([]string, string) {
	if len(in) == 0 {
		return nil, "at least one scope is required"
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if models.IsAdminOnlyScope(s) {
			return nil, "scope " + s + " can never be granted to an app"
		}
		if !models.IsGrantableScope(s) {
			return nil, "unknown scope " + s
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out, ""
}

// ---------------------------------------------------------------------
// Redirect addresses and other URLs
// ---------------------------------------------------------------------

func inDevelopment() bool {
	return os.Getenv("ENVIRONMENT") == "development"
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// validRedirectURI says whether an address may be registered as a place to
// send people back to. Allowed:
//   - https addresses;
//   - http to localhost, 127.0.0.1 or [::1], only when ENVIRONMENT=development;
//   - for public (native) apps, a reverse-DNS scheme such as
//     com.example.app://callback. The scheme must contain a dot.
//
// Never allowed: javascript, data, file and vbscript schemes, a fragment, a
// username or password in the address, or an address over 2048 bytes.
func validRedirectURI(raw string, isPublic bool) bool {
	if raw == "" || len(raw) > maxRedirectURILen || strings.ContainsAny(raw, "\\ \t\r\n#") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" {
		return false
	}
	switch scheme := strings.ToLower(u.Scheme); scheme {
	case "", "javascript", "data", "file", "vbscript":
		return false
	case "https":
		return u.Hostname() != ""
	case "http":
		return inDevelopment() && isLoopbackHost(u.Hostname())
	default:
		return isPublic && strings.Contains(scheme, ".")
	}
}

// validOptionalURL checks a website or logo address: empty, or an https
// address (http to this machine in development).
func validOptionalURL(raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > maxURLFieldLen {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return true
	case "http":
		return inDevelopment() && isLoopbackHost(u.Hostname())
	}
	return false
}

// cleanRedirectURIs checks every address in a client's list.
func cleanRedirectURIs(in []string, isPublic bool) ([]string, string) {
	if len(in) == 0 {
		return nil, "at least one redirect URI is required"
	}
	if len(in) > maxRedirectURIs {
		return nil, "too many redirect URIs"
	}
	seen := map[string]bool{}
	var out []string
	for _, u := range in {
		if !validRedirectURI(u, isPublic) {
			return nil, "invalid redirect URI"
		}
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	return out, ""
}

// ---------------------------------------------------------------------
// Issuer
// ---------------------------------------------------------------------

// errIssuerNotSet is returned when SITE_URL is missing or still an example.
var errIssuerNotSet = errors.New("SITE_URL is not set to the archive's public address")

// oauthIssuer is the address this provider names itself with in discovery
// documents and ID tokens. It comes from SITE_URL, the same setting the
// startup check validates. It is refused if it is empty, still a
// placeholder, not https, or points at this machine (http to this machine
// is accepted when ENVIRONMENT=development).
func oauthIssuer() (string, error) {
	raw := strings.TrimRight(strings.TrimSpace(os.Getenv("SITE_URL")), "/")
	if raw == "" {
		return "", errIssuerNotSet
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errIssuerNotSet
	}
	host := strings.ToLower(u.Hostname())
	if isPlaceholderHost(host) {
		return "", errIssuerNotSet
	}
	if !inDevelopment() {
		if u.Scheme != "https" || isLocalHostName(host) {
			return "", errIssuerNotSet
		}
	} else if u.Scheme != "https" && u.Scheme != "http" {
		return "", errIssuerNotSet
	}
	return u.Scheme + "://" + u.Host + strings.TrimRight(u.Path, "/"), nil
}

// isPlaceholderHost spots the made-up addresses that ship in examples.
func isPlaceholderHost(host string) bool {
	if host == "example.com" || strings.HasSuffix(host, ".example.com") {
		return true
	}
	for _, marker := range []string{"yourdomain", "your-domain", "your-archive", "changeme"} {
		if strings.Contains(host, marker) {
			return true
		}
	}
	return false
}

func isLocalHostName(host string) bool {
	return isLoopbackHost(host) || host == "0.0.0.0" ||
		strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local")
}
