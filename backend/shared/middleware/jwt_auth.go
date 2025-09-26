package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"nuclear-ao3/shared/models"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
}

// NewJWTManager creates a new JWT manager with RSA keys
func NewJWTManager(secretKey, issuer string) (*JWTManager, error) {
	// In production, you should load these from secure storage
	// For now, we'll generate them or use the secret as fallback
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &JWTManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		issuer:     issuer,
	}, nil
}

// GenerateTokenPair creates both access and refresh tokens
func (jm *JWTManager) GenerateTokenPair(user *models.User, roles []string) (string, string, error) {
	// Access token (short-lived: 15 minutes)
	accessToken, err := jm.GenerateAccessToken(user, roles)
	if err != nil {
		return "", "", err
	}

	// Refresh token (long-lived: 30 days) 
	refreshToken, err := jm.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateAccessToken creates a short-lived access token
func (jm *JWTManager) GenerateAccessToken(user *models.User, roles []string) (string, error) {
	now := time.Now()
	claims := models.AuthClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   user.ID.String(),
			Audience:  []string{"nuclear-ao3-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(jm.privateKey)
}

// GenerateRefreshToken creates a long-lived refresh token
func (jm *JWTManager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    jm.issuer,
		Subject:   userID.String(),
		Audience:  []string{"nuclear-ao3-refresh"},
		ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)), // 30 days
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(jm.privateKey)
}

// ValidateAccessToken validates and parses an access token
func (jm *JWTManager) ValidateAccessToken(tokenString string) (*models.AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.AuthClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken validates a refresh token
func (jm *JWTManager) ValidateRefreshToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		// Verify audience for refresh tokens
		if !claims.VerifyAudience("nuclear-ao3-refresh", true) {
			return nil, errors.New("invalid audience")
		}
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// GetPublicKeyPEM returns the public key in PEM format (for service-to-service validation)
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

// JWTAuthMiddleware validates JWT tokens and adds user context
func JWTAuthMiddleware(authService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Validate token (this would use the auth service's JWT manager)
		// For now, we'll simulate this - in real implementation, you'd inject the JWT manager
		claims, err := validateTokenFromService(authService, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Add user context to request
		c.Set("user_id", claims.UserID.String())
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// RequireRoleMiddleware checks if user has required role
func RequireRoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No roles found in token"})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid roles format"})
			c.Abort()
			return
		}

		// Check if user has required role
		hasRole := false
		for _, role := range userRoles {
			if role == requiredRole || role == "admin" { // Admins have access to everything
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Role '%s' required", requiredRole)})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissionMiddleware checks specific permissions
func RequirePermissionMiddleware(resource, action, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No roles found in token"})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid roles format"})
			c.Abort()
			return
		}

		// Check if user has required permission
		if !models.HasPermission(userRoles, resource, action, scope) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Permission required: %s:%s:%s", resource, action, scope),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuthMiddleware adds user context if token is present, but doesn't require it
func OptionalAuthMiddleware(authService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := tokenParts[1]
		claims, err := validateTokenFromService(authService, tokenString)
		if err != nil {
			// Don't abort, just continue without auth context
			c.Next()
			return
		}

		// Add user context
		c.Set("user_id", claims.UserID.String())
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("authenticated", true)

		c.Next()
	}
}

// Helper function to validate token from service (would be properly implemented)
func validateTokenFromService(authService interface{}, tokenString string) (*models.AuthClaims, error) {
	// This is a placeholder - in real implementation, you'd extract the JWT manager
	// from the auth service and call its ValidateAccessToken method
	// For now, return a mock error to show the structure
	return nil, errors.New("token validation not implemented in middleware")
}

// RateLimitMiddleware provides rate limiting per user/IP
func RateLimitMiddleware(redis interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID if authenticated
		userID, _ := c.Get("user_id")
		
		// Use user ID if available, otherwise use IP
		key := c.ClientIP()
		if userID != nil {
			key = fmt.Sprintf("user:%s", userID)
		} else {
			key = fmt.Sprintf("ip:%s", key)
		}

		// Different limits for different endpoints
		var limit int
		var window time.Duration

		endpoint := c.Request.URL.Path
		switch {
		case strings.Contains(endpoint, "/login"):
			limit = 5 // 5 login attempts per 15 minutes
			window = 15 * time.Minute
		case strings.Contains(endpoint, "/register"):
			limit = 3 // 3 registrations per hour
			window = time.Hour
		case strings.Contains(endpoint, "/reset-password"):
			limit = 3 // 3 reset requests per hour
			window = time.Hour
		default:
			limit = 100 // 100 requests per minute for other endpoints
			window = time.Minute
		}

		// Check rate limit (would use Redis in real implementation)
		// For now, just continue - you'd implement Redis-based rate limiting here
		
		c.Next()
	}
}