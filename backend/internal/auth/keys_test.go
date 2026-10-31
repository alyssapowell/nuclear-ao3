package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	sharedKeyOnce sync.Once
	sharedKey     *rsa.PrivateKey
)

// testKey returns a 2048-bit key shared by tests that don't care which key.
func testKey() *rsa.PrivateKey {
	sharedKeyOnce.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		sharedKey = k
	})
	return sharedKey
}

// newTestJWTManager is a manager with the shared test key and no files.
func newTestJWTManager(issuer string) (*JWTManager, error) {
	return NewJWTManager(issuer, testKey())
}

// newTestAuthService is what New does, minus the key files.
func newTestAuthService(db *sql.DB) (*AuthService, error) {
	jm, err := newTestJWTManager("nuclear-ao3")
	if err != nil {
		return nil, err
	}
	return &AuthService{db: db, jwt: jm}, nil
}

func newKey(t *testing.T, bits int) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func writePEM(t *testing.T, name, typ string, der []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func mustLogin(t *testing.T, jm *JWTManager) string {
	t.Helper()
	tok, err := jm.GenerateLoginToken(uuid.New(), uuid.New(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestKeyCurrentKeyTokenValidates(t *testing.T) {
	jm, err := NewJWTManager("nuclear-ao3", newKey(t, 2048))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jm.ValidateToken(mustLogin(t, jm)); err != nil {
		t.Fatalf("token from the current key rejected: %v", err)
	}
}

func TestKeyPlannedRotation(t *testing.T) {
	oldKey, newK := newKey(t, 2048), newKey(t, 2048)
	oldJM, _ := NewJWTManager("nuclear-ao3", oldKey)
	oldToken := mustLogin(t, oldJM)

	// New key signs; the old public key is listed as previous.
	rotated, err := NewJWTManager("nuclear-ao3", newK, &oldKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.ValidateToken(oldToken); err != nil {
		t.Fatalf("old token rejected while the old key is listed: %v", err)
	}
	if _, err := rotated.ValidateToken(mustLogin(t, rotated)); err != nil {
		t.Fatalf("new token rejected: %v", err)
	}

	// Old key removed from the list.
	after, _ := NewJWTManager("nuclear-ao3", newK)
	if _, err := after.ValidateToken(oldToken); err == nil {
		t.Fatal("old token accepted after the old key was removed")
	}
}

func TestKeyUnknownKidRejected(t *testing.T) {
	jm, _ := NewJWTManager("nuclear-ao3", newKey(t, 2048))
	other, _ := NewJWTManager("nuclear-ao3", newKey(t, 2048))
	if _, err := jm.ValidateToken(mustLogin(t, other)); err == nil {
		t.Fatal("token with an unknown kid was accepted")
	}
	// Even a token signed with our key is refused if its kid is wrong.
	imposter := *jm
	imposter.keyID = "something-else"
	if _, err := jm.ValidateToken(mustLogin(t, &imposter)); err == nil {
		t.Fatal("token with a wrong kid was accepted")
	}
}

func TestKeyJWKSListsCurrentAndPrevious(t *testing.T) {
	cur, prev := newKey(t, 2048), newKey(t, 2048)
	jm, _ := NewJWTManager("nuclear-ao3", cur, &prev.PublicKey)
	keys := jm.GetJWKS()["keys"].([]map[string]interface{})
	if len(keys) != 2 {
		t.Fatalf("want 2 keys in JWKS, got %d", len(keys))
	}
	if keys[0]["kid"] != KeyID(&cur.PublicKey) || keys[1]["kid"] != KeyID(&prev.PublicKey) {
		t.Fatal("JWKS should list the current key first, then the previous one")
	}
	for _, k := range keys {
		if _, leaked := k["d"]; leaked {
			t.Fatal("JWKS contains private key material")
		}
	}
}

func TestKeyLoadRejectsBadFiles(t *testing.T) {
	dir := t.TempDir()
	garbage := filepath.Join(dir, "garbage.pem")
	os.WriteFile(garbage, []byte("not a key"), 0o600)
	small := newKey(t, 1024)
	smallFile := writePEM(t, "small.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(small))

	for name, path := range map[string]string{
		"missing":  filepath.Join(dir, "nope.pem"),
		"garbage":  garbage,
		"1024 bit": smallFile,
		"unset":    "",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadJWTManager("nuclear-ao3", path, ""); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	// A bad previous key stops startup too.
	good := writePEM(t, "good.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(newKey(t, 2048)))
	if _, err := LoadJWTManager("nuclear-ao3", good, garbage); err == nil {
		t.Fatal("expected an error for a bad previous key")
	}
}

func TestKeyLoadFormats(t *testing.T) {
	k := newKey(t, 2048)
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(k)
	pub, _ := x509.MarshalPKIXPublicKey(&k.PublicKey)
	p1 := writePEM(t, "a.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(k))
	p8 := writePEM(t, "b.pem", "PRIVATE KEY", pkcs8)
	pp := writePEM(t, "c.pem", "PUBLIC KEY", pub)

	for _, p := range []string{p1, p8} {
		if _, err := LoadPrivateKeyFile(p); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
	if _, err := LoadPrivateKeyFile(pp); err == nil {
		t.Fatal("a public key must not be accepted as the signing key")
	}
	// Previous list accepts private or public keys, with blanks ignored.
	other := newKey(t, 2048)
	jm, err := LoadJWTManager("nuclear-ao3", writePEM(t, "d.pem", "PRIVATE KEY", mustPKCS8(t, other)), pp+" , ,"+p1)
	if err != nil {
		t.Fatal(err)
	}
	if len(jm.previous) != 1 { // pp and p1 are the same key
		t.Fatalf("want 1 previous key, got %d", len(jm.previous))
	}
}

func mustPKCS8(t *testing.T, k *rsa.PrivateKey) []byte {
	t.Helper()
	b, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestKeyGenerateRefusesOverwriteAndIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets", "k.pem")
	kid, err := GenerateKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode is %v, want 0600", st.Mode().Perm())
	}
	if d, _ := os.Stat(filepath.Dir(path)); d.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode is %v, want 0700", d.Mode().Perm())
	}
	priv, err := LoadPrivateKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if priv.N.BitLen() != 3072 || KeyID(&priv.PublicKey) != kid {
		t.Fatal("generated key is not 3072 bits or its kid doesn't match")
	}
	before, _ := os.ReadFile(path)
	if _, err := GenerateKeyFile(path); err == nil {
		t.Fatal("generate overwrote an existing file")
	}
	if after, _ := os.ReadFile(path); string(before) != string(after) {
		t.Fatal("existing key file was changed")
	}
}

func TestKeyIDIsStable(t *testing.T) {
	k := newKey(t, 2048)
	if KeyID(&k.PublicKey) != KeyID(&k.PublicKey) {
		t.Fatal("kid changed between calls")
	}
	// Same key read back from a file in another format has the same kid.
	p := writePEM(t, "x.pem", "PRIVATE KEY", mustPKCS8(t, k))
	loaded, _ := LoadPrivateKeyFile(p)
	if KeyID(&loaded.PublicKey) != KeyID(&k.PublicKey) {
		t.Fatal("kid differs after reloading the key")
	}
	if KeyID(&newKey(t, 2048).PublicKey) == KeyID(&k.PublicKey) {
		t.Fatal("different keys share a kid")
	}
	// Known vector from RFC 7638 section 3.1.
	n := "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
	nb, err := decodeB64Int(n)
	if err != nil {
		t.Fatal(err)
	}
	const want = "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"
	if got := KeyID(&rsa.PublicKey{N: nb, E: 65537}); got != want {
		t.Fatalf("kid %s, RFC 7638 says %s", got, want)
	}
}

func decodeB64Int(s string) (*big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}
