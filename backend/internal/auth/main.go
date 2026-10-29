package auth

import (
	"context"
	"database/sql"
	"fmt"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/notifications"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Mount registers the account, session, admin and OAuth2/OIDC routes.
func Mount(r *gin.Engine, authService *AuthService, d *platform.Deps) {

	// Auth endpoints
	api := r.Group("/api/v1/auth")
	{
		// Public endpoints (no authentication required)
		api.POST("/register", authService.Register)
		api.POST("/login", authService.Login)
		api.POST("/refresh", authService.RefreshToken)
		api.POST("/reset-password", authService.RequestPasswordReset)
		api.POST("/reset-password/confirm", authService.ConfirmPasswordReset)
		api.POST("/verify-email", authService.VerifyEmail)
		api.POST("/resend-verification", authService.ResendVerification)

		// Protected endpoints (require authentication)
		protected := api.Group("")
		protected.Use(JWTAuthMiddleware(authService))
		{
			protected.POST("/logout", authService.Logout)
			protected.GET("/me", authService.GetProfile)
			protected.PUT("/me", authService.UpdateProfile)
			protected.POST("/change-password", authService.ChangePassword)
			protected.GET("/sessions", authService.GetSessions)
			protected.DELETE("/sessions/:session_id", authService.RevokeSession)
			protected.GET("/security-events", authService.GetSecurityEvents)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(JWTAuthMiddleware(authService))
		admin.Use(d.RequireRole("admin"))

		{
			admin.GET("/users", authService.ListUsers)
			admin.GET("/users/:user_id", authService.GetUser)
			admin.PUT("/users/:user_id", authService.UpdateUser)
			admin.POST("/users/:user_id/roles", authService.GrantRole)
			admin.DELETE("/users/:user_id/roles/:role", authService.RevokeRole)
			admin.GET("/security-events", authService.GetAllSecurityEvents)
			admin.GET("/metrics", authService.GetAuthMetrics)

			// OAuth2 client management
			admin.GET("/oauth/clients", authService.AdminListClients)
			admin.GET("/oauth/clients/:client_id", authService.AdminGetClient)
			admin.PUT("/oauth/clients/:client_id", authService.AdminUpdateClient)
			admin.DELETE("/oauth/clients/:client_id", authService.AdminDeleteClient)
			admin.POST("/oauth/clients/:client_id/reset-secret", authService.AdminResetClientSecret)
			admin.GET("/oauth/tokens", authService.AdminListTokens)
			admin.DELETE("/oauth/tokens/:token_id", authService.AdminRevokeToken)
		}
	}

	// OAuth2/OIDC Discovery endpoints
	r.GET("/.well-known/openid-configuration", authService.WellKnownOIDC)
	r.GET("/.well-known/oauth-authorization-server", authService.WellKnownOAuth2)

	// OAuth2/OIDC endpoints
	oauth := r.Group("/auth")
	{
		// Authorization endpoint (GET and POST for different flows)
		oauth.GET("/authorize", authService.Authorize)
		oauth.POST("/authorize", authService.Authorize)

		// Token endpoint
		oauth.POST("/token", authService.Token)

		// User info endpoint (OIDC)
		oauth.GET("/userinfo", authService.UserInfo)
		oauth.POST("/userinfo", authService.UserInfo)

		// Token introspection (RFC 7662)
		oauth.POST("/introspect", authService.Introspect)

		// Token revocation (RFC 7009)
		oauth.POST("/revoke", authService.Revoke)

		// Client registration (Dynamic Client Registration)
		oauth.POST("/register-client", authService.RegisterClient)

		// JWKS endpoint for token verification
		oauth.GET("/jwks", authService.GetJWKS)

		// Consent handling
		oauth.GET("/consent/:consent_id", authService.ShowConsent)
		oauth.POST("/consent/:consent_id", authService.ProcessConsent)

		// User consent management
		protected := oauth.Group("")
		protected.Use(JWTAuthMiddleware(authService))
		{
			protected.GET("/consents", authService.GetUserConsents)
			protected.DELETE("/consents/:consent_id", authService.RevokeConsent)
			protected.GET("/authorized-applications", authService.GetAuthorizedApplications)
			protected.DELETE("/authorized-applications/:client_id", authService.RevokeApplication)
		}
	}
}

// AuthService holds all dependencies for authentication
type AuthService struct {
	db      *sql.DB
	redis   *redis.Client
	jwt     *JWTManager
	onEvent func(context.Context, *notifications.EventData) // nil until SetEventHandler
}

// New builds the auth service on top of the shared database and Redis.
func New(db *sql.DB, rdb *redis.Client) (*AuthService, error) {
	jwtManager, err := NewJWTManager(
		platform.Env("JWT_SECRET", ""),
		platform.Env("JWT_ISSUER", "nuclear-ao3"),
		db,
	)
	if err != nil {
		return nil, fmt.Errorf("create JWT manager: %w", err)
	}
	return &AuthService{db: db, redis: rdb, jwt: jwtManager}, nil
}

// SetEventHandler sets where account events (like registration) are sent.
func (as *AuthService) SetEventHandler(h func(context.Context, *notifications.EventData)) {
	as.onEvent = h
}

// ValidateUserToken checks a login token and returns the user it belongs to.
// Every other area of the app uses this through platform.Deps.

func (as *AuthService) ValidateUserToken(token string) (uuid.UUID, error) {
	claims, err := as.jwt.ValidateToken(token)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(claims.Subject)
}
