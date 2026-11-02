package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"nuclear-ao3/shared/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Client authentication for the token, introspection and revocation
// endpoints (RFC 6749 section 2.3).

type clientAuthMode int

const (
	// authAllowPublic is for the token and revocation endpoints: a public app
	// may identify itself without a secret (its proof is PKCE or the token
	// itself, and it can only reach its own tokens).
	authAllowPublic clientAuthMode = iota
	// authConfidentialOnly is for introspection: the app must prove it holds
	// its secret.
	authConfidentialOnly
)

var (
	errInvalidClient  = errors.New("client authentication failed")
	errInvalidRequest = errors.New("invalid request")
)

// sensitiveQueryParams must travel in the POST body. A request that puts one
// in the address is refused, because addresses end up in logs and in
// browser history.
var sensitiveQueryParams = []string{
	"client_secret", "code", "code_verifier", "refresh_token", "token", "password",
}

// rejectSecretsInQuery answers 400 and returns true if the request carries a
// secret in its query string.
func rejectSecretsInQuery(c *gin.Context) bool {
	q := c.Request.URL.Query()
	for _, name := range sensitiveQueryParams {
		if q.Has(name) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "Send parameters in the POST body, not the address",
			})
			return true
		}
	}
	return false
}

// hasBasicHeader says whether the request tries to use HTTP Basic.
func hasBasicHeader(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	return len(h) >= 6 && strings.EqualFold(h[:6], "basic ")
}

// basicCredentials reads client_id and secret from an HTTP Basic header. Both
// are form-url-decoded afterwards, as RFC 6749 2.3.1 says.
func basicCredentials(r *http.Request) (id, secret string, err error) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return "", "", errInvalidClient
	}
	if id, err = url.QueryUnescape(user); err != nil {
		return "", "", errInvalidClient
	}
	if secret, err = url.QueryUnescape(pass); err != nil {
		return "", "", errInvalidClient
	}
	return id, secret, nil
}

// requestCredentials takes the client's id and secret from HTTP Basic if it
// is there, and only otherwise from the POST body. The query string is never
// read.
func requestCredentials(c *gin.Context) (id, secret string, err error) {
	if hasBasicHeader(c.Request) {
		id, secret, err = basicCredentials(c.Request)
		if err != nil {
			return "", "", err
		}
		// Two ways of authenticating at once is a mistake, or an attack.
		if c.PostForm("client_secret") != "" {
			return "", "", errInvalidRequest
		}
		if formID := c.PostForm("client_id"); formID != "" && formID != id {
			return "", "", errInvalidClient
		}
		return id, secret, nil
	}
	return c.PostForm("client_id"), c.PostForm("client_secret"), nil
}

var clientDummyHash struct {
	once sync.Once
	hash []byte
}

// spendBcrypt does the same work as checking a real secret. It runs when the
// client is unknown, so the time a refusal takes doesn't show which clients
// exist.
func spendBcrypt(secret string) {
	clientDummyHash.once.Do(func() {
		clientDummyHash.hash, _ = bcrypt.GenerateFromPassword([]byte("no such client"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(clientDummyHash.hash, []byte(secret))
}

// authenticateClient finds the calling app and checks its secret. It
// returns errInvalidClient for anything wrong with the credentials, and
// takes about as long whether or not the client exists.
func (as *AuthService) authenticateClient(c *gin.Context, mode clientAuthMode) (*models.OAuthClient, error) {
	id, secret, err := requestCredentials(c)
	if err != nil {
		spendBcrypt(secret)
		return nil, err
	}
	client, err := as.getClientByID(id)
	if err != nil {
		spendBcrypt(secret)
		return nil, errInvalidClient
	}
	if client.IsPublic {
		if mode != authAllowPublic || secret != "" {
			return nil, errInvalidClient
		}
		return client, nil
	}
	if bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(secret)) != nil {
		return nil, errInvalidClient
	}
	if !as.allowAuthenticatedClient(c, client.ID.String()) {
		return nil, errRateLimited
	}
	return client, nil
}

// errRateLimited means the 429 answer has already been written.
var errRateLimited = errors.New("rate limited")

// clientAuthFailed writes the answer for a failed authenticateClient.
func clientAuthFailed(c *gin.Context, err error) {
	if errors.Is(err, errRateLimited) {
		return
	}
	if errors.Is(err, errInvalidRequest) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Use one way to authenticate the client",
		})
		return
	}
	if hasBasicHeader(c.Request) {
		c.Header("WWW-Authenticate", `Basic realm="oauth"`)
	}
	c.JSON(http.StatusUnauthorized, gin.H{
		"error":             "invalid_client",
		"error_description": "Client authentication failed",
	})
}
