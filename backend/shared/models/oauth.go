package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuthClient represents a registered OAuth2 client application
type OAuthClient struct {
	ID                uuid.UUID `json:"client_id" db:"client_id"`
	Secret            string    `json:"-" db:"client_secret"` // Never serialize
	Name              string    `json:"client_name" db:"client_name"`
	Description       string    `json:"description" db:"description"`
	Website           string    `json:"website" db:"website"`
	LogoURL           string    `json:"logo_url" db:"logo_url"`
	RedirectURIs      []string  `json:"redirect_uris" db:"redirect_uris"`
	Scopes            []string  `json:"scopes" db:"scopes"`
	GrantTypes        []string  `json:"grant_types" db:"grant_types"`
	ResponseTypes     []string  `json:"response_types" db:"response_types"`
	IsPublic          bool      `json:"is_public" db:"is_public"`     // PKCE required for public clients
	IsConfidential    bool      `json:"is_confidential" db:"is_confidential"`
	IsTrusted         bool      `json:"is_trusted" db:"is_trusted"`   // Skip consent for trusted clients
	IsFirstParty      bool      `json:"is_first_party" db:"is_first_party"` // AO3's own apps
	OwnerID           *uuid.UUID `json:"owner_id" db:"owner_id"`      // User who registered the client
	AccessTokenTTL    int       `json:"access_token_ttl" db:"access_token_ttl"` // Seconds
	RefreshTokenTTL   int       `json:"refresh_token_ttl" db:"refresh_token_ttl"` // Seconds
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// AuthorizationCode represents a short-lived authorization code
type AuthorizationCode struct {
	Code              string    `json:"code" db:"code"`
	ClientID          uuid.UUID `json:"client_id" db:"client_id"`
	UserID            uuid.UUID `json:"user_id" db:"user_id"`
	RedirectURI       string    `json:"redirect_uri" db:"redirect_uri"`
	Scopes            []string  `json:"scopes" db:"scopes"`
	State             string    `json:"state" db:"state"`
	Nonce             string    `json:"nonce,omitempty" db:"nonce"` // OIDC
	CodeChallenge     string    `json:"code_challenge,omitempty" db:"code_challenge"` // PKCE
	CodeChallengeMethod string  `json:"code_challenge_method,omitempty" db:"code_challenge_method"` // PKCE
	ExpiresAt         time.Time `json:"expires_at" db:"expires_at"`
	UsedAt            *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// OAuthAccessToken represents an OAuth2 access token
type OAuthAccessToken struct {
	ID           uuid.UUID `json:"jti" db:"id"`
	Token        string    `json:"access_token" db:"token"`
	UserID       *uuid.UUID `json:"sub" db:"user_id"`
	ClientID     uuid.UUID `json:"client_id" db:"client_id"`
	Scopes       []string  `json:"scope" db:"scopes"`
	TokenType    string    `json:"token_type" db:"token_type"` // Usually "Bearer"
	ExpiresAt    time.Time `json:"exp" db:"expires_at"`
	IsRevoked    bool      `json:"is_revoked" db:"is_revoked"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	LastUsed     *time.Time `json:"last_used,omitempty" db:"last_used"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time `json:"iat" db:"created_at"`
}

// OAuthRefreshToken represents an OAuth2 refresh token
type OAuthRefreshToken struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Token        string     `json:"refresh_token" db:"token"`
	AccessTokenID uuid.UUID `json:"access_token_id" db:"access_token_id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	ClientID     uuid.UUID  `json:"client_id" db:"client_id"`
	Scopes       []string   `json:"scopes" db:"scopes"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	IsRevoked    bool       `json:"is_revoked" db:"is_revoked"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	LastUsed     *time.Time `json:"last_used,omitempty" db:"last_used"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// UserConsent represents user consent for OAuth2 scopes
type UserConsent struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ClientID  uuid.UUID `json:"client_id" db:"client_id"`
	Scopes    []string  `json:"scopes" db:"scopes"`
	GrantedAt time.Time `json:"granted_at" db:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsRevoked bool      `json:"is_revoked" db:"is_revoked"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// OAuthScope represents available OAuth2 scopes
type OAuthScope struct {
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Category    string `json:"category" db:"category"` // "profile", "works", "admin", etc.
	IsDefault   bool   `json:"is_default" db:"is_default"`
	RequiresConsent bool `json:"requires_consent" db:"requires_consent"`
	IsAdminOnly bool   `json:"is_admin_only" db:"is_admin_only"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// OIDC Claims for ID tokens
type OIDCClaims struct {
	// Standard JWT claims
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	AuthTime  int64  `json:"auth_time,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	
	// Standard OIDC profile claims
	Name              string `json:"name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	MiddleName        string `json:"middle_name,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Profile           string `json:"profile,omitempty"`
	Picture           string `json:"picture,omitempty"`
	Website           string `json:"website,omitempty"`
	Gender            string `json:"gender,omitempty"`
	Birthdate         string `json:"birthdate,omitempty"`
	Zoneinfo          string `json:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`
	
	// Standard OIDC email claims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	
	// Standard OIDC phone claims
	PhoneNumber         string `json:"phone_number,omitempty"`
	PhoneNumberVerified bool   `json:"phone_number_verified,omitempty"`
	
	// Standard OIDC address claims
	Address *OIDCAddress `json:"address,omitempty"`
	
	// AO3-specific claims
	AO3Username     string   `json:"ao3_username,omitempty"`
	AO3DisplayName  string   `json:"ao3_display_name,omitempty"`
	AO3Roles        []string `json:"ao3_roles,omitempty"`
	AO3JoinDate     int64    `json:"ao3_join_date,omitempty"`
	AO3WorkCount    int      `json:"ao3_work_count,omitempty"`
	AO3BookmarkCount int     `json:"ao3_bookmark_count,omitempty"`
}

type OIDCAddress struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"street_address,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// Request/Response types

type ClientRegistrationRequest struct {
	Name              string   `json:"client_name" validate:"required,min=1,max=100"`
	Description       string   `json:"description" validate:"max=500"`
	Website           string   `json:"website" validate:"omitempty,url"`
	LogoURL           string   `json:"logo_url" validate:"omitempty,url"`
	RedirectURIs      []string `json:"redirect_uris" validate:"required,min=1,dive,required,uri"`
	Scopes            []string `json:"scopes" validate:"required,min=1"`
	GrantTypes        []string `json:"grant_types" validate:"required,min=1"`
	ResponseTypes     []string `json:"response_types" validate:"required,min=1"`
	IsPublic          bool     `json:"is_public"`
	AccessTokenTTL    int      `json:"access_token_ttl" validate:"min=300,max=86400"` // 5 min to 24 hours
	RefreshTokenTTL   int      `json:"refresh_token_ttl" validate:"min=3600,max=2592000"` // 1 hour to 30 days
}

type ClientRegistrationResponse struct {
	ClientID         string    `json:"client_id"`
	ClientSecret     string    `json:"client_secret,omitempty"` // Only for confidential clients
	Name             string    `json:"client_name"`
	Description      string    `json:"description"`
	Website          string    `json:"website"`
	LogoURL          string    `json:"logo_url"`
	RedirectURIs     []string  `json:"redirect_uris"`
	Scopes           []string  `json:"scopes"`
	GrantTypes       []string  `json:"grant_types"`
	ResponseTypes    []string  `json:"response_types"`
	IsPublic         bool      `json:"is_public"`
	AccessTokenTTL   int       `json:"access_token_ttl"`
	RefreshTokenTTL  int       `json:"refresh_token_ttl"`
	CreatedAt        time.Time `json:"created_at"`
}

type AuthorizeRequest struct {
	ClientID            string `form:"client_id" validate:"required"`
	RedirectURI         string `form:"redirect_uri" validate:"required,uri"`
	ResponseType        string `form:"response_type" validate:"required,oneof=code"`
	Scope               string `form:"scope" validate:"required"`
	State               string `form:"state"`
	Nonce               string `form:"nonce"` // OIDC
	CodeChallenge       string `form:"code_challenge"` // PKCE
	CodeChallengeMethod string `form:"code_challenge_method" validate:"omitempty,oneof=S256 plain"`
	Display             string `form:"display" validate:"omitempty,oneof=page popup touch wap"`
	Prompt              string `form:"prompt" validate:"omitempty,oneof=none login consent select_account"`
	MaxAge              int    `form:"max_age"`
	UILocales           string `form:"ui_locales"`
	LoginHint           string `form:"login_hint"`
}

type TokenRequest struct {
	GrantType    string `form:"grant_type" validate:"required,oneof=authorization_code refresh_token client_credentials"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	RefreshToken string `form:"refresh_token"`
	Scope        string `form:"scope"`
	CodeVerifier string `form:"code_verifier"` // PKCE
	Username     string `form:"username"` // Resource Owner Password Credentials (if enabled)
	Password     string `form:"password"` // Resource Owner Password Credentials (if enabled)
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"` // OIDC
}

type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

type IntrospectRequest struct {
	Token         string `form:"token" validate:"required"`
	TokenTypeHint string `form:"token_type_hint" validate:"omitempty,oneof=access_token refresh_token"`
	ClientID      string `form:"client_id"`
	ClientSecret  string `form:"client_secret"`
}

type IntrospectResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Username  string   `json:"username,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
	ExpiresAt int64    `json:"exp,omitempty"`
	IssuedAt  int64    `json:"iat,omitempty"`
	NotBefore int64    `json:"nbf,omitempty"`
	Subject   string   `json:"sub,omitempty"`
	Audience  string   `json:"aud,omitempty"`
	Issuer    string   `json:"iss,omitempty"`
	JWTID     string   `json:"jti,omitempty"`
}

type UserInfoResponse struct {
	Subject           string   `json:"sub"`
	Name              string   `json:"name,omitempty"`
	GivenName         string   `json:"given_name,omitempty"`
	FamilyName        string   `json:"family_name,omitempty"`
	MiddleName        string   `json:"middle_name,omitempty"`
	Nickname          string   `json:"nickname,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
	Profile           string   `json:"profile,omitempty"`
	Picture           string   `json:"picture,omitempty"`
	Website           string   `json:"website,omitempty"`
	Email             string   `json:"email,omitempty"`
	EmailVerified     bool     `json:"email_verified,omitempty"`
	Gender            string   `json:"gender,omitempty"`
	Birthdate         string   `json:"birthdate,omitempty"`
	Zoneinfo          string   `json:"zoneinfo,omitempty"`
	Locale            string   `json:"locale,omitempty"`
	PhoneNumber       string   `json:"phone_number,omitempty"`
	PhoneNumberVerified bool   `json:"phone_number_verified,omitempty"`
	Address           *OIDCAddress `json:"address,omitempty"`
	UpdatedAt         int64    `json:"updated_at,omitempty"`
	
	// AO3-specific claims
	AO3Username      string   `json:"ao3_username,omitempty"`
	AO3DisplayName   string   `json:"ao3_display_name,omitempty"`
	AO3Roles         []string `json:"ao3_roles,omitempty"`
	AO3JoinDate      int64    `json:"ao3_join_date,omitempty"`
	AO3WorkCount     int      `json:"ao3_work_count,omitempty"`
	AO3BookmarkCount int      `json:"ao3_bookmark_count,omitempty"`
}

// OIDC Discovery document
type OIDCDiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSUri                          string   `json:"jwks_uri"`
	RegistrationEndpoint             string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	ResponseModesSupported           []string `json:"response_modes_supported,omitempty"`
	GrantTypesSupported              []string `json:"grant_types_supported"`
	ACRValuesSupported               []string `json:"acr_values_supported,omitempty"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	IDTokenEncryptionAlgValuesSupported []string `json:"id_token_encryption_alg_values_supported,omitempty"`
	IDTokenEncryptionEncValuesSupported []string `json:"id_token_encryption_enc_values_supported,omitempty"`
	UserinfoSigningAlgValuesSupported []string `json:"userinfo_signing_alg_values_supported,omitempty"`
	UserinfoEncryptionAlgValuesSupported []string `json:"userinfo_encryption_alg_values_supported,omitempty"`
	UserinfoEncryptionEncValuesSupported []string `json:"userinfo_encryption_enc_values_supported,omitempty"`
	RequestObjectSigningAlgValuesSupported []string `json:"request_object_signing_alg_values_supported,omitempty"`
	RequestObjectEncryptionAlgValuesSupported []string `json:"request_object_encryption_alg_values_supported,omitempty"`
	RequestObjectEncryptionEncValuesSupported []string `json:"request_object_encryption_enc_values_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
	DisplayValuesSupported           []string `json:"display_values_supported,omitempty"`
	ClaimTypesSupported              []string `json:"claim_types_supported,omitempty"`
	ClaimsSupported                  []string `json:"claims_supported,omitempty"`
	ServiceDocumentation             string   `json:"service_documentation,omitempty"`
	ClaimsLocalesSupported           []string `json:"claims_locales_supported,omitempty"`
	UILocalesSupported               []string `json:"ui_locales_supported,omitempty"`
	ClaimsParameterSupported         bool     `json:"claims_parameter_supported,omitempty"`
	RequestParameterSupported        bool     `json:"request_parameter_supported,omitempty"`
	RequestURIParameterSupported     bool     `json:"request_uri_parameter_supported,omitempty"`
	RequireRequestURIRegistration    bool     `json:"require_request_uri_registration,omitempty"`
	OpPolicyURI                      string   `json:"op_policy_uri,omitempty"`
	OpTosURI                         string   `json:"op_tos_uri,omitempty"`
	RevocationEndpoint               string   `json:"revocation_endpoint,omitempty"`
	RevocationEndpointAuthMethodsSupported []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	RevocationEndpointAuthSigningAlgValuesSupported []string `json:"revocation_endpoint_auth_signing_alg_values_supported,omitempty"`
	IntrospectionEndpoint            string   `json:"introspection_endpoint,omitempty"`
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	IntrospectionEndpointAuthSigningAlgValuesSupported []string `json:"introspection_endpoint_auth_signing_alg_values_supported,omitempty"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
}

// Predefined scopes for AO3
var AO3OAuthScopes = map[string]OAuthScope{
	"openid": {
		Name:        "openid",
		Description: "OpenID Connect authentication",
		Category:    "authentication", 
		IsDefault:   true,
		RequiresConsent: false,
		IsAdminOnly: false,
	},
	"profile": {
		Name:        "profile",
		Description: "Access your basic profile information",
		Category:    "profile",
		IsDefault:   true,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"email": {
		Name:        "email",
		Description: "Access your email address",
		Category:    "profile",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"read": {
		Name:        "read",
		Description: "Read your works, bookmarks, and other content",
		Category:    "works",
		IsDefault:   true,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"write": {
		Name:        "write",
		Description: "Create and edit your works",
		Category:    "works",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"works:manage": {
		Name:        "works:manage",
		Description: "Full access to manage your works (create, edit, delete)",
		Category:    "works",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"comments:write": {
		Name:        "comments:write", 
		Description: "Post comments on works",
		Category:    "social",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"bookmarks:manage": {
		Name:        "bookmarks:manage",
		Description: "Manage your bookmarks",
		Category:    "works",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"collections:manage": {
		Name:        "collections:manage",
		Description: "Manage collections you own",
		Category:    "works",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: false,
	},
	"tags:wrangle": {
		Name:        "tags:wrangle",
		Description: "Tag wrangling permissions",
		Category:    "moderation",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: true,
	},
	"admin": {
		Name:        "admin",
		Description: "Administrative access to the platform",
		Category:    "admin",
		IsDefault:   false,
		RequiresConsent: true,
		IsAdminOnly: true,
	},
}