package auth

import (
	"net/url"
	"os"
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// The tests need a public address for the provider (SITE_URL), as a real
// server does. The tests of what happens without one unset it for themselves.
func TestMain(m *testing.M) {
	os.Setenv("SITE_URL", "https://archive.example.org")
	os.Unsetenv("ENVIRONMENT")
	os.Exit(m.Run())
}

// The RFC 7636 example pair. Most tests only need some valid PKCE and use
// this one; the tests of PKCE itself make their own.
const (
	testVerifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	testChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	pkceParams    = "&code_challenge=" + testChallenge + "&code_challenge_method=S256"
)

// generousOAuthRates keeps the OAuth rate limits out of the way of tests that
// make many requests from one address. The rate limit tests set their own.
func generousOAuthRates() map[string]oauthRate {
	big := oauthRate{PerIP: 1 << 30, PerClient: 1 << 30}
	return map[string]oauthRate{"token": big, "introspect": big, "revoke": big, "authorize": big}
}

// formBody encodes a request struct as an application/x-www-form-urlencoded
// body, using its `form` tags, which is how apps really call the token,
// introspection and revocation endpoints.
func formBody(v any) []byte {
	vals := url.Values{}
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Tag.Get("form")
		if name == "" || rv.Field(i).Kind() != reflect.String || rv.Field(i).String() == "" {
			continue
		}
		vals.Set(name, rv.Field(i).String())
	}
	return []byte(vals.Encode())
}

func readFileForTest(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func bcryptHash(secret string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	return string(h), err
}
