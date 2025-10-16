package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	keyID      string
}

// NewJWTManager creates a new JWT manager with persistent keys
func NewJWTManager(secret, issuer string, db *sql.DB) (*JWTManager, error) {
	var privateKey *rsa.PrivateKey
	var keyID string
	var err error

	// If database is available, try to load/save persistent keys
	if db != nil {
		// Try to load existing key from database
		privateKey, keyID, err = loadJWTKeyFromDB(db)
		if err != nil {
			log.Printf("No existing JWT key found, generating new one: %v", err)
			// Generate new key pair
			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return nil, fmt.Errorf("failed to generate private key: %v", err)
			}
			keyID = uuid.New().String()

			// Save to database for persistence
			if err := saveJWTKeyToDB(db, privateKey, keyID); err != nil {
				log.Printf("Warning: Failed to save JWT key to database: %v", err)
			} else {
				log.Printf("Generated and saved new JWT signing key with ID: %s", keyID)
			}
		} else {
			log.Printf("Loaded existing JWT signing key with ID: %s", keyID)
		}
	} else {
		// No database available (test mode) - generate ephemeral key
		log.Printf("No database available, generating ephemeral JWT key for testing")
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %v", err)
		}
		keyID = uuid.New().String()
	}

	return &JWTManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		issuer:     issuer,
		keyID:      keyID,
	}, nil
}

// loadJWTKeyFromDB loads the JWT signing key from the database
func loadJWTKeyFromDB(db *sql.DB) (*rsa.PrivateKey, string, error) {
	var keyData string
	var keyID string

	err := db.QueryRow("SELECT key_data, key_id FROM jwt_signing_keys WHERE is_active = true ORDER BY created_at DESC LIMIT 1").Scan(&keyData, &keyID)
	if err != nil {
		return nil, "", err
	}

	// Decode PEM data
	block, _ := pem.Decode([]byte(keyData))
	if block == nil {
		return nil, "", fmt.Errorf("failed to decode PEM block")
	}

	// Parse private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse private key: %v", err)
	}

	return privateKey, keyID, nil
}

// saveJWTKeyToDB saves the JWT signing key to the database
func saveJWTKeyToDB(db *sql.DB, privateKey *rsa.PrivateKey, keyID string) error {
	// Encode private key to PEM format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Deactivate old keys
	_, err := db.Exec("UPDATE jwt_signing_keys SET is_active = false WHERE is_active = true")
	if err != nil {
		return fmt.Errorf("failed to deactivate old keys: %v", err)
	}

	// Insert new key
	_, err = db.Exec(`
		INSERT INTO jwt_signing_keys (key_id, key_data, is_active, created_at) 
		VALUES ($1, $2, true, NOW())
	`, keyID, string(privateKeyPEM))

	return err
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

// ValidateToken validates and parses a JWT token
func (jm *JWTManager) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
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

// GetJWKS returns the JSON Web Key Set for the public key
func (jm *JWTManager) GetJWKS() map[string]interface{} {
	return map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": jm.keyID,
				"n":   encodeRSAPublicKeyComponent(jm.publicKey.N),
				"e":   encodeRSAPublicKeyComponent(big.NewInt(int64(jm.publicKey.E))),
			},
		},
	}
}

// Helper function to encode RSA key components for JWKS
func encodeRSAPublicKeyComponent(component *big.Int) string {
	return base64.RawURLEncoding.EncodeToString(component.Bytes())
}
