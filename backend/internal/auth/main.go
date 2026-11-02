package auth

import (
	"context"
	"database/sql"
	"os"

	"nuclear-ao3/internal/platform"
	"nuclear-ao3/shared/email"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Mount registers the account, session, admin and OAuth2/OIDC routes.
func Mount(r *gin.Engine, authService *AuthService, d *platform.Deps) {

	// Auth endpoints
	api := r.Group("/api/v1/auth")
	{
		// Login, reset and activation share a per-IP limit against guessing.
		limit := authService.attemptLimit()
		// Public endpoints (no authentication required)
		api.POST("/register", authService.Register)
		api.POST("/login", limit, authService.Login)
		api.POST("/reset-password", limit, authService.RequestPasswordReset)
		api.POST("/reset-password/confirm", limit, authService.ConfirmPasswordReset)
		api.POST("/verify-email", limit, authService.VerifyEmail)
		api.POST("/resend-verification", limit, authService.ResendVerification)

		// Protected endpoints (require authentication)
		protected := api.Group("")
		protected.Use(JWTAuthMiddleware(authService))
		{
			protected.POST("/logout", authService.Logout)
			protected.GET("/me", authService.GetProfile)
			protected.PUT("/me", d.ActiveAccount, authService.UpdateProfile)
			protected.POST("/change-password", authService.ChangePassword)
			protected.GET("/sessions", authService.GetSessions)
			protected.DELETE("/sessions/:session_id", authService.RevokeSession)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(JWTAuthMiddleware(authService))
		admin.Use(d.ActiveAccount)
		admin.Use(d.RequireRole("admin"))

		{
			admin.GET("/users", authService.ListUsers)
			admin.GET("/users/:user_id", authService.GetUser)
			admin.PUT("/users/:user_id", authService.UpdateUser)
			admin.POST("/users/:user_id/roles", authService.GrantRole)
			admin.DELETE("/users/:user_id/roles/:role", authService.RevokeRole)

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

	// Apps people register themselves. A new app is pending until an admin
	// approves it (see oauth_apps.go). Banned and suspended people can't use
	// any of these.
	apps := r.Group("/api/v1/apps", d.RequireUser, d.ActiveAccount)
	{
		apps.POST("", authService.RegisterApp)
		apps.GET("", authService.ListMyApps)
		apps.PATCH("/:id", authService.UpdateMyApp)
		apps.POST("/:id/reset-secret", authService.ResetMyAppSecret)
		apps.DELETE("/:id", authService.DeleteMyApp)
	}

	// The review queue and the four decisions, for admins.
	review := r.Group("/api/v1/admin/apps", d.RequireUser, d.ActiveAccount, d.RequireRole("admin"))
	{
		review.GET("", authService.AdminListApps)
		review.POST("/:id/approve", authService.adminReview("approve"))
		review.POST("/:id/reject", authService.adminReview("reject"))
		review.POST("/:id/suspend", authService.adminReview("suspend"))
		review.POST("/:id/reinstate", authService.adminReview("reinstate"))
	}

	// OAuth2/OIDC Discovery endpoints
	r.GET("/.well-known/openid-configuration", authService.WellKnownOIDC)
	r.GET("/.well-known/oauth-authorization-server", authService.WellKnownOAuth2)

	// OAuth2/OIDC endpoints
	oauth := r.Group("/auth")
	{
		// Authorization endpoint (GET and POST for different flows)
		oauth.GET("/authorize", authService.oauthLimit("authorize"), authService.Authorize)
		oauth.POST("/authorize", authService.oauthLimit("authorize"), authService.Authorize)

		// Token endpoint
		oauth.POST("/token", authService.oauthLimit("token"), authService.Token)

		// User info endpoint (OIDC)
		oauth.GET("/userinfo", authService.UserInfo)
		oauth.POST("/userinfo", authService.UserInfo)

		// Token introspection (RFC 7662)
		oauth.POST("/introspect", authService.oauthLimit("introspect"), authService.Introspect)

		// Token revocation (RFC 7009)
		oauth.POST("/revoke", authService.oauthLimit("revoke"), authService.Revoke)

		// Client registration. Only admins register apps (and discovery does
		// not advertise this address): an open registration would let anyone
		// create a client.
		oauth.POST("/register-client", d.RequireUser, d.ActiveAccount, d.RequireRole(), authService.RegisterClient)

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
	db  *sql.DB
	jwt *JWTManager
	// oidc signs ID tokens. nil means the login key does it (not
	// recommended); see OIDC_SIGNING_KEY_FILE.
	oidc *JWTManager
	mail *email.Config // nil until SetMail: no emails are sent
	// guard holds the in-memory counters against password guessing.
	guard loginGuard
	// oauth holds the counters for the OAuth endpoints (oauth_limits.go).
	oauth oauthGuard
	// oauthRates replaces the default OAuth limits per endpoint. Tests use it.
	oauthRates map[string]oauthRate
}

// New builds the auth service on top of the shared database. The
// login signing key is read from the files named by SIGNING_KEY_FILE and
// SIGNING_KEY_PREVIOUS_FILES; without a valid key it returns an error. The
// key for OIDC ID tokens comes from OIDC_SIGNING_KEY_FILE.
func New(db *sql.DB) (*AuthService, error) {
	jwtManager, err := LoadJWTManager(
		platform.Env("JWT_ISSUER", "nuclear-ao3"),
		os.Getenv("SIGNING_KEY_FILE"),
		os.Getenv("SIGNING_KEY_PREVIOUS_FILES"),
	)
	if err != nil {
		return nil, err
	}
	oidc, err := LoadOIDCManager(jwtManager, platform.Env("JWT_ISSUER", "nuclear-ao3"), os.Getenv("OIDC_SIGNING_KEY_FILE"))
	if err != nil {
		return nil, err
	}
	return &AuthService{db: db, jwt: jwtManager, oidc: oidc}, nil
}

// SetMail turns on account emails (activation and password reset), sent from
// and with links to the archive described by cfg.
func (as *AuthService) SetMail(cfg *email.Config) {
	as.mail = cfg
}

// ValidateUserToken checks a login token and returns the user it belongs to.
// Every other area of the app uses this through platform.Deps. The token must
// belong to a session that has not been logged out or revoked.
func (as *AuthService) ValidateUserToken(token string) (uuid.UUID, error) {
	userID, _, err := as.validateSession(context.Background(), token)
	return userID, err
}
