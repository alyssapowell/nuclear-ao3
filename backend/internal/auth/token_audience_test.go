package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Only login tokens for this archive may be used to log in. ID tokens issued
// to third-party apps are signed with the same key, so the audience and
// issuer checks are what keep them out.
func TestValidateTokenAcceptsOnlyLoginTokens(t *testing.T) {
	jm, err := newTestJWTManager("nuclear-ao3")
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.New()

	sign := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		s, err := tok.SignedString(jm.privateKey)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	base := func() jwt.MapClaims {
		return jwt.MapClaims{"iss": "nuclear-ao3", "sub": user.String(), "aud": LoginAudience,
			"exp": time.Now().Add(time.Hour).Unix()}
	}

	login, err := jm.GenerateToken(user, LoginAudience, []string{"user"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jm.ValidateToken(login); err != nil {
		t.Errorf("login token rejected: %v", err)
	}

	idToken := base()
	idToken["aud"] = uuid.New().String() // a third-party client ID
	noExp := base()
	delete(noExp, "exp")
	otherIssuer := base()
	otherIssuer["iss"] = "someone-else"

	for name, claims := range map[string]jwt.MapClaims{
		"third-party audience": idToken,
		"no expiry":            noExp,
		"other issuer":         otherIssuer,
	} {
		if _, err := jm.ValidateToken(sign(claims)); err == nil {
			t.Errorf("%s: token was accepted, want rejected", name)
		}
	}
}
