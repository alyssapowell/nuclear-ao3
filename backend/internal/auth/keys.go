package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

// The login-token signing key is a PEM file the admin generates once with
// "nuclear keys generate", keeps a copy of somewhere safe, and points the
// server at with SIGNING_KEY_FILE. It is never stored in the database.

const (
	minKeyBits       = 2048
	generatedKeyBits = 3072
)

// KeyID is the RFC 7638 thumbprint of a public key: the base64url SHA-256 of
// its minimal JSON form. The same key always gets the same ID.
func KeyID(pub *rsa.PublicKey) string {
	// RFC 7638 asks for exactly these members, in this order, no spaces.
	enc := func(n *big.Int) string { return base64.RawURLEncoding.EncodeToString(n.Bytes()) }
	e, _ := json.Marshal(enc(big.NewInt(int64(pub.E))))
	n, _ := json.Marshal(enc(pub.N))
	sum := sha256.Sum256([]byte(`{"e":` + string(e) + `,"kty":"RSA","n":` + string(n) + `}`))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func checkKeySize(pub *rsa.PublicKey) error {
	if pub == nil || pub.N == nil {
		return errors.New("no key")
	}
	if pub.N.BitLen() < minKeyBits {
		return fmt.Errorf("RSA key is %d bits; it must be at least %d", pub.N.BitLen(), minKeyBits)
	}
	return nil
}

// parseRSAKey reads a PEM RSA key. It accepts private keys (PKCS#1 or PKCS#8)
// and public keys (PKIX or PKCS#1). priv is nil when the file holds only a
// public key.
func parseRSAKey(data []byte) (priv *rsa.PrivateKey, pub *rsa.PublicKey, err error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, nil, errors.New("not a PEM file")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		var k interface{}
		if k, err = x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			var ok bool
			if priv, ok = k.(*rsa.PrivateKey); !ok {
				return nil, nil, errors.New("not an RSA key")
			}
		}
	case "PUBLIC KEY":
		var k interface{}
		if k, err = x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			var ok bool
			if pub, ok = k.(*rsa.PublicKey); !ok {
				return nil, nil, errors.New("not an RSA key")
			}
		}
	case "RSA PUBLIC KEY":
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, nil, fmt.Errorf("unsupported PEM type %q", block.Type)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("not a valid RSA key: %w", err)
	}
	if priv != nil {
		pub = &priv.PublicKey
	}
	if err := checkKeySize(pub); err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// LoadPrivateKeyFile reads the signing key from path.
func LoadPrivateKeyFile(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	priv, _, err := parseRSAKey(data)
	if err != nil {
		return nil, err
	}
	if priv == nil {
		return nil, errors.New("this is a public key; the signing key must be a private key")
	}
	return priv, nil
}

// LoadPublicKeyFile reads a key kept only to verify old tokens. The file may
// hold a private or a public key; only the public half is used.
func LoadPublicKeyFile(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_, pub, err := parseRSAKey(data)
	return pub, err
}

// SplitPaths splits a comma-separated list of file paths, ignoring blanks.
func SplitPaths(list string) []string {
	var out []string
	for _, p := range strings.Split(list, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// LoadKeyFiles reads the current signing key and the previous keys.
func LoadKeyFiles(currentFile, previousFiles string) (*rsa.PrivateKey, []*rsa.PublicKey, error) {
	if strings.TrimSpace(currentFile) == "" {
		return nil, nil, errors.New("SIGNING_KEY_FILE is not set")
	}
	priv, err := LoadPrivateKeyFile(currentFile)
	if err != nil {
		return nil, nil, fmt.Errorf("SIGNING_KEY_FILE (%s): %w", currentFile, err)
	}
	var prev []*rsa.PublicKey
	for _, p := range SplitPaths(previousFiles) {
		pub, err := LoadPublicKeyFile(p)
		if err != nil {
			return nil, nil, fmt.Errorf("SIGNING_KEY_PREVIOUS_FILES (%s): %w", p, err)
		}
		prev = append(prev, pub)
	}
	return priv, prev, nil
}

// LoadJWTManager builds the manager from the key files named in the
// environment. If the key can't be loaded the error says how to fix it.
func LoadJWTManager(issuer, currentFile, previousFiles string) (*JWTManager, error) {
	priv, prev, err := LoadKeyFiles(currentFile, previousFiles)
	if err != nil {
		return nil, fmt.Errorf("%w\nThe server needs a signing key for logins. Create one with: nuclear keys generate\n"+
			"then set SIGNING_KEY_FILE to it (see docs/RUNBOOK.md, \"Set up the signing key\")", err)
	}
	return NewJWTManager(issuer, priv, prev...)
}

// GenerateKeyFile creates a new 3072-bit RSA signing key at path (PKCS#8 PEM,
// mode 0600, directory 0700) and returns its key ID. It never overwrites a file.
func GenerateKeyFile(path string) (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, generatedKeyBits)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("%s already exists; not overwriting it", path)
		}
		return "", err
	}
	_, err = f.Write(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return KeyID(&key.PublicKey), nil
}
