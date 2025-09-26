# Nuclear AO3 OAuth2/OIDC API Testing Guide

This guide helps you test the Nuclear AO3 authentication service OAuth2/OIDC implementation locally.

## 🚀 Quick Start

### Prerequisites
- PostgreSQL running (`brew services start postgresql`)
- Redis running (`brew services start redis`)
- Go 1.25+ installed

### Starting the Service

**Option 1: Use the startup script (recommended)**
```bash
cd nuclear-ao3/backend/auth-service
./start-dev.sh
```

**Option 2: Manual startup**
```bash
cd nuclear-ao3/backend/auth-service
export DATABASE_URL="postgres://$(whoami)@localhost/ao3_nuclear_test?sslmode=disable"
export REDIS_URL="localhost:6379" 
export BASE_URL="http://localhost:8081"
export GIN_MODE="debug"
export PORT="8081"
go build . && ./auth-service
```

### Stopping the Service
- Press `Ctrl+C` in the terminal running the service
- Or from another terminal: `pkill -f auth-service`

## 📋 API Endpoint Reference

### Service Status
```bash
# Health check
curl http://localhost:8081/health | jq

# Metrics (Prometheus format)
curl http://localhost:8081/metrics
```

### OAuth2/OIDC Discovery
```bash
# OIDC Discovery Document (shows all available endpoints and capabilities)
curl http://localhost:8081/.well-known/openid-configuration | jq

# OAuth2 Authorization Server Metadata
curl http://localhost:8081/.well-known/oauth-authorization-server | jq

# JWKS (JSON Web Key Set) for token verification
curl http://localhost:8081/auth/jwks | jq
```

## 🔐 OAuth2 Flow Testing

### 1. Client Registration

**Register a Confidential Client (server-side app)**
```bash
curl -X POST http://localhost:8081/auth/register-client \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My Server App",
    "description": "A server-side application", 
    "website": "https://myapp.example.com",
    "redirect_uris": ["https://myapp.example.com/auth/callback"],
    "scopes": ["read", "write", "profile"],
    "grant_types": ["authorization_code", "client_credentials", "refresh_token"],
    "response_types": ["code"],
    "is_public": false
  }' | jq
```

**Register a Public Client (mobile/SPA)**
```bash
curl -X POST http://localhost:8081/auth/register-client \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My Mobile App", 
    "description": "A mobile application",
    "website": "https://myapp.example.com",
    "redirect_uris": ["myapp://auth/callback", "http://localhost:3000/callback"],
    "scopes": ["read", "profile"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "is_public": true
  }' | jq
```

💡 **Save the `client_id` and `client_secret` from the response for the next steps!**

### 2. Client Credentials Flow (Machine-to-Machine)

```bash
# Replace YOUR_CLIENT_ID and YOUR_CLIENT_SECRET with values from registration
curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET&scope=read"
```

**Expected Response:**
```json
{
  "access_token": "base64-encoded-token",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read"
}
```

### 3. Authorization Code Flow (User Authorization)

**Step 1: Get Authorization Code (open in browser)**
```
http://localhost:8081/auth/authorize?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=https://myapp.example.com/auth/callback&scope=read%20profile&state=random123
```

🔄 **In test mode, consent is auto-approved and you'll be redirected to:**
```
https://myapp.example.com/auth/callback?code=AUTHORIZATION_CODE&state=random123
```

**Step 2: Exchange Code for Token**
```bash
# Replace AUTHORIZATION_CODE with the code from step 1
curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=AUTHORIZATION_CODE&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET&redirect_uri=https://myapp.example.com/auth/callback"
```

### 4. PKCE Flow (Public Clients)

**Generate PKCE parameters:**
```bash
# Generate code verifier and challenge
CODE_VERIFIER=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-43)
CODE_CHALLENGE=$(echo -n $CODE_VERIFIER | shasum -a 256 | cut -d' ' -f1 | xxd -r -p | base64 | tr -d "=+/")
echo "Code Verifier: $CODE_VERIFIER"
echo "Code Challenge: $CODE_CHALLENGE"
```

**Authorization URL (open in browser):**
```
http://localhost:8081/auth/authorize?response_type=code&client_id=YOUR_PUBLIC_CLIENT_ID&redirect_uri=http://localhost:3000/callback&scope=read&code_challenge=CODE_CHALLENGE&code_challenge_method=S256&state=pkce123
```

**Exchange code with PKCE:**
```bash
curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=AUTHORIZATION_CODE&client_id=YOUR_PUBLIC_CLIENT_ID&redirect_uri=http://localhost:3000/callback&code_verifier=$CODE_VERIFIER"
```

### 5. Refresh Token Flow

```bash
# Use refresh_token from previous authorization code flow
curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token&refresh_token=YOUR_REFRESH_TOKEN&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"
```

## 🔍 Token Management

### Token Introspection
```bash
# Check if a token is active and get metadata
curl -X POST http://localhost:8081/auth/introspect \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=YOUR_ACCESS_TOKEN&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"
```

### Token Revocation
```bash
# Revoke an access or refresh token
curl -X POST http://localhost:8081/auth/revoke \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=YOUR_TOKEN&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"
```

### UserInfo Endpoint (OIDC)
```bash
# Get user information (requires 'profile' or 'email' scope)
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8081/auth/userinfo
```

## 👤 User Management (For Testing User Flows)

### User Registration
```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "SecurePassword123!",
    "display_name": "Test User"
  }' | jq
```

### User Login  
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePassword123!"
  }' | jq
```

### Get User Profile
```bash
# Use access_token from login response
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8081/api/v1/auth/me | jq
```

## 🧪 Test Scenarios

### Happy Path Testing
1. Register a client
2. Get client credentials token
3. Use token to access protected resources
4. Test token introspection
5. Revoke token

### Error Testing
```bash
# Invalid client credentials
curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=invalid&client_secret=invalid"

# Invalid scope
curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET&scope=invalid_scope"

# Invalid redirect URI
curl "http://localhost:8081/auth/authorize?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=https://evil.com/callback&scope=read"
```

## 🎯 Available Scopes

| Scope | Description |
|-------|-------------|
| `openid` | OIDC authentication (required for ID tokens) |
| `profile` | Basic profile information (username, display name) |
| `email` | Email address access |
| `read` | Read access to user's content |
| `write` | Write access to create content |
| `works:manage` | Full work management (create, edit, delete) |
| `comments:write` | Post comments |
| `bookmarks:manage` | Manage bookmarks |
| `collections:manage` | Manage collections |

## 🔧 Development Tips

### Viewing Logs
The service outputs detailed request logs in debug mode:
```
::1 - [2025-09-26T13:41:56-07:00] "POST /auth/token HTTP/1.1 200 52ms"
```

### Database Inspection
```bash
# Connect to test database
psql ao3_nuclear_test

# View registered clients
SELECT client_id, client_name, is_public, scopes FROM oauth_clients;

# View active tokens
SELECT token, user_id, client_id, scopes, expires_at FROM oauth_access_tokens WHERE is_revoked = false;
```

### Redis Inspection
```bash
# Connect to Redis
redis-cli

# View stored data
KEYS *
GET "consent:some-id"
```

## 🚨 Common Issues

**"Connection refused"** - Service not running or wrong port
**"Invalid client"** - Check client_id/client_secret from registration response
**"Invalid scope"** - Use scopes from the supported list above
**"PKCE required"** - Public clients must use PKCE flow
**"Invalid redirect URI"** - Must exactly match registered redirect_uris

## 📊 Performance Testing

```bash
# Measure client registration performance
time curl -X POST http://localhost:8081/auth/register-client \
  -H "Content-Type: application/json" \
  -d '{"client_name":"Perf Test","redirect_uris":["https://test.com"],"scopes":["read"],"grant_types":["client_credentials"],"response_types":["code"]}'

# Measure token generation performance  
time curl -X POST http://localhost:8081/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"
```

---

## 🤝 Contributing

Found an issue or want to add a test case? Please update this guide and submit a PR!

**Latest Update:** September 26, 2025 - Initial OAuth2/OIDC implementation testing guide