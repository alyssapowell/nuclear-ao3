package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTManager signs login tokens with one key and checks them against that
// key plus any previous keys kept for a planned rotation. The keys come from
// files the admin holds (see keys.go); nothing is stored in the database.
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	keyID      string

	previous []*rsa.PublicKey          // old keys, verify only, in the order given
	verify   map[string]*rsa.PublicKey // kid -> public key, current and previous
}

// NewJWTManager builds a manager that signs with key and also accepts tokens
// signed by the previous public keys. The server gets its keys from files
// through LoadJWTManager; tests call this directly.
func NewJWTManager(issuer string, key *rsa.PrivateKey, previous ...*rsa.PublicKey) (*JWTManager, error) {
	if key == nil {
		return nil, fmt.Errorf("no signing key")
	}
	if err := checkKeySize(&key.PublicKey); err != nil {
		return nil, err
	}
	jm := &JWTManager{
		privateKey: key,
		publicKey:  &key.PublicKey,
		issuer:     issuer,
		keyID:      KeyID(&key.PublicKey),
		verify:     map[string]*rsa.PublicKey{},
	}
	jm.verify[jm.keyID] = jm.publicKey
	for _, pub := range previous {
		if err := checkKeySize(pub); err != nil {
			return nil, err
		}
		kid := KeyID(pub)
		if _, dup := jm.verify[kid]; dup {
			continue
		}
		jm.verify[kid] = pub
		jm.previous = append(jm.previous, pub)
	}
	return jm, nil
}

// verificationKey is the jwt key function: it picks the key named by the
// token's kid header and refuses a token whose kid isn't one we hold.
func (jm *JWTManager) verificationKey(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("token has no key id")
	}
	pub, ok := jm.verify[kid]
	if !ok {
		return nil, fmt.Errorf("token was signed with an unknown key")
	}
	return pub, nil
}

// GenerateToken creates a new JWT token
func (jm *JWTManager) GenerateToken(userID uuid.UUID, audience string, scopes []string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   jm.issuer,
		"sub":   userID.String(),
		"aud":   audience,
		"exp":   now.Add(expiresIn).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"jti":   uuid.New().String(),
		"scope": scopes,
		"typ":   "Bearer",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = jm.keyID

	return token.SignedString(jm.privateKey)
}

// LoginAudience is the audience of tokens issued when a user logs in to the
// archive itself. OAuth ID tokens carry the third-party app's ID instead.
const LoginAudience = "nuclear-ao3"

// loginClaims are the claims of a login token: the standard ones plus the
// ID of the session the token belongs to.
type loginClaims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
}

// GenerateLoginToken creates the token for one login session. Revoking the
// session makes the token stop working.
func (jm *JWTManager) GenerateLoginToken(userID, sessionID uuid.UUID, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   jm.issuer,
		"sub":   userID.String(),
		"aud":   LoginAudience,
		"exp":   now.Add(expiresIn).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"jti":   uuid.New().String(),
		"sid":   sessionID.String(),
		"scope": []string{"user"},
		"typ":   "Bearer",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = jm.keyID
	return token.SignedString(jm.privateKey)
}

// parseLoginToken checks the signature, issuer, audience and expiry of a
// login token. It does not check that the session is still active.
func (jm *JWTManager) parseLoginToken(tokenString string) (*loginClaims, error) {
	claims := &loginClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, jm.verificationKey, jwt.WithAudience(LoginAudience), jwt.WithIssuer(jm.issuer), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// ValidateToken checks that tokenString is a valid login token for this
// archive. Tokens issued to third-party apps (such as OIDC ID tokens, which
// are signed with the same key) are rejected because their audience differs.
// It does not check the session; use AuthService.ValidateUserToken for that.
func (jm *JWTManager) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	claims, err := jm.parseLoginToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &claims.RegisteredClaims, nil
}

// GetPublicKey returns the public key for token verification
func (jm *JWTManager) GetPublicKey() *rsa.PublicKey {
	return jm.publicKey
}

// GetPublicKeyPEM returns the public key in PEM format
func (jm *JWTManager) GetPublicKeyPEM() (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(jm.publicKey)
	if err != nil {
		return "", err
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

// GetJWKS returns the JSON Web Key Set: the current key first, then the
// previous public keys, so apps can still check tokens signed before a rotation.
func (jm *JWTManager) GetJWKS() map[string]interface{} {
	keys := []map[string]interface{}{jwk(jm.publicKey)}
	for _, pub := range jm.previous {
		keys = append(keys, jwk(pub))
	}
	return map[string]interface{}{"keys": keys}
}

func jwk(pub *rsa.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": KeyID(pub),
		"n":   encodeRSAPublicKeyComponent(pub.N),
		"e":   encodeRSAPublicKeyComponent(big.NewInt(int64(pub.E))),
	}
}

// Helper function to encode RSA key components for JWKS
func encodeRSAPublicKeyComponent(component *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(component.Bytes())
}
