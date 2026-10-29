# OAuth2/OIDC Testing Suite Documentation

## Overview

This comprehensive testing suite validates the OAuth2/OIDC authentication implementation for the Nuclear AO3 modernization project. The test suite ensures full compliance with OAuth2 RFC 6749, OIDC Core 1.0, and AO3-specific authentication requirements.

## 🎯 Test Coverage

### ✅ **Core OAuth2 Flows** (`oauth_test.go`)
- **Client Registration**: Dynamic client registration (RFC 7591)
- **Authorization Code Flow**: Standard OAuth2 flow with PKCE
- **Token Exchange**: Authorization code → access/refresh tokens
- **Token Introspection**: RFC 7662 compliance
- **Token Revocation**: RFC 7009 compliance
- **Scope Validation**: Fine-grained permission control
- **Security Validation**: PKCE enforcement, redirect URI validation

### ✅ **OpenID Connect (OIDC)** (`oidc_test.go`)
- **Discovery Endpoints**: `.well-known/openid-configuration`
- **ID Token Generation**: JWT structure and claims validation
- **UserInfo Endpoint**: Profile and email claims
- **JWKS Endpoint**: Public key distribution
- **Consent Management**: User authorization tracking
- **Nonce Validation**: Replay attack prevention

### ✅ **Performance & Load Testing** (`oauth_performance_test.go`)
- **Token Introspection**: 50+ requests/sec, <50ms average
- **Authorization Flow**: Complete flow <200ms average  
- **OIDC Discovery**: 500+ requests/sec (cached)
- **UserInfo Endpoint**: 100+ requests/sec, <30ms average
- **Mixed Load Testing**: Real-world usage patterns
- **Concurrent Operations**: Multi-client scenarios

### ✅ **Security Compliance**
- **PKCE Enforcement**: Public client protection
- **Authorization Code Reuse**: Prevention of replay attacks
- **Token Expiration**: Proper TTL handling
- **Scope Escalation**: Prevention of privilege escalation
- **Client Authentication**: Confidential vs public clients
- **Cross-Client Isolation**: Security boundary validation

## 🚀 Running the Tests

### Quick Start
```bash
# Run all tests
./run_oauth_tests.sh

# Skip performance tests (faster)
./run_oauth_tests.sh --skip-performance

# Generate coverage report only
./run_oauth_tests.sh --coverage-only

# Cleanup test data only
./run_oauth_tests.sh --clean-only
```

### Prerequisites
- **Go 1.21+**: Test runtime
- **PostgreSQL**: Test database (`ao3_nuclear_test`)
- **Redis**: Session and cache storage
- **Database Migrations**: OAuth2 schema setup

### Manual Test Execution
```bash
# Individual test suites
go test -v ./oauth_test.go
go test -v ./oidc_test.go  
go test -v ./oauth_performance_test.go

# With coverage
go test -cover -coverprofile=coverage.out ./oauth_test.go
go tool cover -html=coverage.out
```

## 📊 Test Structure

### Test Suites

#### 1. **OAuth2TestSuite**
- **Purpose**: Core OAuth2 functionality validation
- **Database**: Redis DB 3
- **Focus**: Authorization flows, token management, client validation

#### 2. **OIDCTestSuite**  
- **Purpose**: OpenID Connect specific features
- **Database**: Redis DB 4
- **Focus**: ID tokens, discovery, user claims

#### 3. **OAuth2PerformanceTestSuite**
- **Purpose**: Performance and load validation
- **Database**: Redis DB 5
- **Focus**: Throughput, latency, concurrent operations

### Test Data Management

Each test suite creates isolated test data:
```go
// Test users
oauth_user, oauth_admin, app_developer, unverified

// Test clients  
confidential_client, public_client, first_party_app, service_client

// Test scenarios
Multi-user, multi-client, concurrent access patterns
```

**Automatic Cleanup**: All test data is automatically removed after test execution.

## 🔍 Key Test Scenarios

### Client Registration Tests
```go
TestClientRegistration_ConfidentialClient_Success()
TestClientRegistration_PublicClient_Success()
TestClientRegistration_InvalidRedirectURI()
TestClientRegistration_InvalidScope()
```

### Authorization Flow Tests
```go
TestAuthorizationFlow_ConfidentialClient_Success()
TestAuthorizationFlow_PKCE_Success()
TestAuthorizationFlow_InvalidPKCE()
TestAuthorizationCode_DoubleUse_ShouldFail()
```

### Token Management Tests
```go
TestTokenIntrospection_ValidToken()
TestTokenRevocation_AccessToken()
TestRefreshToken_Success()
TestRefreshToken_ScopeReduction()
```

### OIDC Specific Tests
```go
TestOIDCFlow_WithIDToken()
TestIDToken_ClaimsValidation()
TestUserInfo_ValidToken_ProfileScope()
TestConsentFlow_FirstTime()
```

### Performance Benchmarks
```go
TestTokenIntrospection_Performance()    // Target: 50+ req/sec, <50ms
TestAuthorizationFlow_Performance()     // Target: <200ms per flow
TestMixedOperations_LoadTest()          // Target: 50+ mixed ops/sec
```

## 📈 Performance Targets

| Operation | Target Throughput | Target Latency | Notes |
|-----------|------------------|----------------|--------|
| Token Introspection | 50+ req/sec | <50ms avg | Database-backed validation |
| Authorization Flow | N/A | <200ms avg | Complete code→token flow |
| OIDC Discovery | 500+ req/sec | <10ms avg | Cached response |
| UserInfo Endpoint | 100+ req/sec | <30ms avg | User data retrieval |
| Token Refresh | 20+ req/sec | <100ms avg | Token rotation |
| Mixed Operations | 50+ req/sec | Various | Real-world patterns |

## 🛡️ Security Test Coverage

### Critical Security Validations

1. **PKCE Enforcement**
   - Public clients MUST use PKCE
   - Code verifier validation
   - Challenge method support (S256, plain)

2. **Authorization Code Security**
   - Single-use enforcement
   - Expiration validation (10 minutes)
   - Redirect URI binding

3. **Token Security**
   - JWT signature validation
   - Expiration enforcement
   - Revocation status checking

4. **Client Authentication**
   - Confidential client secret validation
   - Public client identification
   - Client-specific scope enforcement

5. **Scope Validation**
   - Requested vs granted scope verification
   - Client-specific scope limitations
   - Admin scope protection

## 🔧 Configuration

### Environment Variables
```bash
TEST_DATABASE_URL="postgres://ao3_user:ao3_password@localhost/ao3_nuclear_test?sslmode=disable"
TEST_REDIS_URL="localhost:6379"
BASE_URL="https://test.nuclear-ao3.com"
GIN_MODE="test"
```

### Database Schema
Required tables (created by migrations):
- `users`, `oauth_clients`, `authorization_codes`
- `oauth_access_tokens`, `oauth_refresh_tokens`
- `user_consents`, `user_sessions`

### Redis Usage
- **DB 0**: Production data (not used in tests)
- **DB 1**: Development cache
- **DB 2**: Core auth tests
- **DB 3**: OAuth2 tests  
- **DB 4**: OIDC tests
- **DB 5**: Performance tests

## 🐛 Debugging Tests

### Test Logs
```bash
# View detailed test output
cat test_results_oauth_test.log
cat test_results_oidc_test.log
cat test_results_performance.log
```

### Common Issues

1. **Database Connection**
   ```bash
   # Check PostgreSQL status
   pg_isready -d ao3_nuclear_test
   
   # Create test database if missing
   createdb ao3_nuclear_test
   ```

2. **Redis Connection**
   ```bash
   # Check Redis status
   redis-cli ping
   
   # Clear test databases
   redis-cli -n 3 FLUSHDB
   ```

3. **Migration Issues**
   ```bash
   # Run migrations manually
   psql ao3_nuclear_test -f ../migrations/003_oauth2_tables.sql
   ```

### Verbose Testing
```bash
# Run with maximum verbosity
go test -v -count=1 -timeout=10m ./oauth_test.go

# Run specific test
go test -v -run="TestAuthorizationFlow_PKCE_Success" ./oauth_test.go
```

## 📋 Test Checklist

Before deploying OAuth2/OIDC to production:

- [ ] All core OAuth2 tests pass
- [ ] All OIDC tests pass  
- [ ] Performance benchmarks met
- [ ] Security tests validate
- [ ] Coverage report shows >90%
- [ ] No flaky or intermittent failures
- [ ] Database migrations applied
- [ ] Redis configuration verified

## 🔄 Continuous Integration

### GitHub Actions Integration
```yaml
name: OAuth2/OIDC Tests
on: [push, pull_request]
jobs:
  oauth-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: ao3_nuclear_test
          POSTGRES_USER: ao3_user
          POSTGRES_PASSWORD: ao3_password
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run OAuth2/OIDC Tests
        run: |
          cd nuclear-ao3/backend/auth-service
          ./run_oauth_tests.sh --skip-performance
```

### Pre-commit Hooks
```bash
# Add to .git/hooks/pre-commit
#!/bin/bash
cd nuclear-ao3/backend/auth-service
./run_oauth_tests.sh --skip-performance
```

## 📚 Additional Resources

### OAuth2/OIDC Standards
- [RFC 6749 - OAuth 2.0 Authorization Framework](https://tools.ietf.org/html/rfc6749)
- [RFC 7636 - PKCE](https://tools.ietf.org/html/rfc7636)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [RFC 7662 - Token Introspection](https://tools.ietf.org/html/rfc7662)

### Nuclear AO3 Documentation
- [AO3 Modernization RFC](../../AO3_MODERNIZATION_RFC.md)
- [Modern Architecture](../../MODERN_ARCHITECTURE.md)
- [Auth Service Implementation](./README.md)

## 🎉 Success Criteria

Your OAuth2/OIDC implementation is **production-ready** when:

✅ **All test suites pass consistently**  
✅ **Performance targets are met**  
✅ **Security validations succeed**  
✅ **Code coverage >90%**  
✅ **Zero critical security vulnerabilities**  
✅ **Documentation is complete**

---

*This testing suite represents enterprise-grade OAuth2/OIDC validation, ensuring the Nuclear AO3 authentication system meets the highest standards for security, performance, and compliance.*