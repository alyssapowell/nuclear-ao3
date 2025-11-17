# Nuclear AO3 - Testing Guide

## Overview

This document provides a comprehensive guide to testing Nuclear AO3 for both development and CI/CD purposes.

## Test Coverage Summary

### Backend (Go)
- **Total Test Files:** 24
- **Coverage Areas:**
  - Auth Service (OAuth2, OIDC, JWT)
  - Work Service (CRUD, chapters, comments)
  - Tag Service (prominence, smart filtering)
  - Search Service (Elasticsearch integration)
  - Notification Service (events, subscriptions)
  - Email Service (SMTP, templates)
  - Shared libraries (messaging, notifications)

### Frontend (React/Next.js)
- **Test Files:** 30+ component tests
- **E2E Tests:** 56 Playwright tests
- **Coverage Areas:**
  - Component unit tests (Jest)
  - E2E user flows (Playwright)
  - Accessibility tests
  - PWA functionality tests
  - Multi-user scenarios

## Quick Start

### Run All Tests
```bash
# From project root
./scripts/run-all-tests.sh
```

### Run Backend Tests Only
```bash
cd backend
go test ./... -v -race -coverprofile=coverage.out

# View coverage
go tool cover -html=coverage.out
```

### Run Frontend Tests Only
```bash
cd frontend

# TypeScript type checking
npm run build

# Jest unit tests
npm test

# E2E tests
npm run test:e2e
```

### Run Email Package Tests
```bash
cd backend/shared/email
go test -v -cover
```

## Continuous Integration (CI)

### GitHub Actions Workflows

#### 1. Main CI Pipeline (`.github/workflows/ci.yml`)
Runs on every pull request to `main`:

**Jobs:**
- ✅ Backend Go tests with race detector
- ✅ Frontend build and type checking
- ✅ Docker builds for all services
- ✅ Docker Compose stack test
- ✅ Security scanning (Trivy, TruffleHog)
- ✅ Integration tests with full stack

**Status:** Comprehensive coverage

#### 2. Accessibility CI (`.github/workflows/accessibility-ci.yml`)
- Runs accessibility checks on PR
- Validates WCAG compliance

#### 3. Frontend CI (`.github/workflows/frontend-ci.yml`)
- Quick frontend-only checks
- Faster feedback loop

### What CI Tests

```
┌─────────────────────────────────────────────────────────┐
│ Backend Tests                                           │
├─────────────────────────────────────────────────────────┤
│ • Unit tests (24 test files)                           │
│ • Integration tests                                     │
│ • Race condition detection                              │
│ • Code coverage reporting                               │
│ • Linting (golangci-lint)                              │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Frontend Tests                                          │
├─────────────────────────────────────────────────────────┤
│ • TypeScript compilation                                │
│ • Component unit tests (Jest)                           │
│ • E2E tests (Playwright)                                │
│ • Accessibility tests                                   │
│ • Build verification                                    │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Docker Tests                                            │
├─────────────────────────────────────────────────────────┤
│ • Individual service builds (7 services)                │
│ • Container startup verification                        │
│ • Full docker-compose stack                             │
│ • Health endpoint checks                                │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Integration Tests                                       │
├─────────────────────────────────────────────────────────┤
│ • Database connectivity (Postgres, Redis, ES)           │
│ • Service health endpoints                              │
│ • API Gateway routing                                   │
│ • Frontend accessibility                                │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Security Tests                                          │
├─────────────────────────────────────────────────────────┤
│ • Vulnerability scanning (Trivy)                        │
│ • Secret detection (TruffleHog)                         │
│ • Dependency audits                                     │
└─────────────────────────────────────────────────────────┘
```

## Test Categories

### Unit Tests
**Purpose:** Test individual functions and components in isolation

**Backend Example:**
```go
func TestWelcomeEmailTemplate(t *testing.T) {
    username := "testuser"
    subject, body := WelcomeEmailTemplate(username)
    
    if subject != "Welcome to Nuclear AO3!" {
        t.Errorf("Unexpected subject: %s", subject)
    }
}
```

**Frontend Example:**
```typescript
test('BookmarkButton renders correctly', () => {
    render(<BookmarkButton workId="123" />);
    expect(screen.getByRole('button')).toBeInTheDocument();
});
```

### Integration Tests
**Purpose:** Test interactions between components/services

**Location:** `tests/integration/`

**Examples:**
- API endpoint integration
- Database operations
- Service-to-service communication
- Email sending flow

### E2E Tests
**Purpose:** Test complete user workflows

**Location:** `frontend/e2e/`

**Tools:** Playwright

**Examples:**
- User registration and login
- Creating and publishing works
- Search and filtering
- Subscription workflows
- Multi-user interactions

## Test Environment Setup

### Prerequisites
```bash
# Backend
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Elasticsearch 8+

# Frontend
- Node.js 20+
- npm 10+

# E2E Tests
- Playwright browsers (auto-installed)
```

### Environment Variables
Tests use `.env.example` as a base. Override with:

```bash
# Test database (separate from dev)
DATABASE_URL=postgres://test:test@localhost:5432/test_db

# Test SMTP (MailHog for development)
SMTP_HOST=localhost
SMTP_PORT=1025

# JWT test secret
JWT_SECRET=test-secret-key-for-testing-only
```

## Running Specific Test Suites

### Backend

```bash
# Run tests for specific service
cd backend/auth-service && go test -v

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run only short tests (skip slow integration tests)
go test -short ./...

# Run specific test
go test -run TestWelcomeEmailTemplate

# Run with race detector
go test -race ./...
```

### Frontend

```bash
cd frontend

# Run all Jest tests
npm test

# Watch mode for development
npm run test:watch

# Coverage report
npm run test:coverage

# E2E tests
npm run test:e2e

# E2E with UI (interactive)
npm run test:e2e:ui

# Specific E2E test file
npx playwright test auth-flow

# Accessibility tests
npm run test:accessibility
```

## Coverage Goals

### Current Coverage
- **Backend:** ~65-70% (measured)
- **Frontend:** Partial (needs improvement)
- **E2E:** Good coverage of critical paths

### Target Coverage
- **Backend:** 80%+ for critical paths (auth, work management)
- **Frontend:** 70%+ for components
- **E2E:** All critical user journeys

## Known Test Issues

### Backend
❌ **Auth Service Tests:** Some OAuth2 tests failing due to mock setup
- Impact: Low (OAuth works in production)
- Fix: Update test mocks to match current implementation

### Frontend
❌ **MSW Integration:** TextEncoder not defined in Jest
- Impact: Medium (API mocking tests fail)
- Fix: Add polyfill to jest.setup.js

❌ **Missing Components:** Some test files reference deleted components
- Impact: Low
- Fix: Update or remove outdated test files

### Solutions in Progress
1. ✅ Email package tests: **Complete and passing**
2. 🔄 Frontend polyfills: **Needs jest.setup.js update**
3. 🔄 Auth test mocks: **Needs refactoring**

## Adding New Tests

### Backend Go Test
```go
// backend/myservice/myfile_test.go
package myservice

import "testing"

func TestMyFunction(t *testing.T) {
    result := MyFunction("input")
    expected := "expected output"
    
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### Frontend Component Test
```typescript
// frontend/src/components/__tests__/MyComponent.test.tsx
import { render, screen } from '@testing-library/react';
import MyComponent from '../MyComponent';

describe('MyComponent', () => {
    it('renders correctly', () => {
        render(<MyComponent />);
        expect(screen.getByText('Hello')).toBeInTheDocument();
    });
});
```

### E2E Test
```typescript
// frontend/e2e/my-feature.spec.ts
import { test, expect } from '@playwright/test';

test('user can do something', async ({ page }) => {
    await page.goto('/');
    await page.click('button');
    await expect(page.locator('h1')).toContainText('Success');
});
```

## CI/CD Best Practices

### Before Pushing
1. Run `./scripts/run-all-tests.sh`
2. Fix any failing tests
3. Ensure coverage hasn't decreased
4. Check for linting errors

### Pull Request Checklist
- [ ] All tests pass locally
- [ ] Added tests for new features
- [ ] Updated existing tests if behavior changed
- [ ] No test coverage regression
- [ ] CI pipeline passes

### Debugging CI Failures

```bash
# View CI logs
gh run view --log

# Re-run failed jobs
gh run rerun --failed

# Download artifacts
gh run download
```

## Performance Testing

### Load Testing
```bash
# Backend load test
cd tests/performance
node performance-load-test.js

# Concurrent user simulation
./performance-multi-user.sh
```

### Benchmarking
```bash
# Go benchmarks
cd backend
go test -bench=. -benchmem

# Frontend performance
npm run lighthouse
```

## Test Data Management

### Development
```bash
# Seed test data
node tools/test-data-populator.js

# Clean test database
docker-compose exec postgres psql -U ao3_user -d ao3_nuclear -c "TRUNCATE users CASCADE;"
```

### CI
- Uses ephemeral databases
- Seeded with minimal data
- Cleaned after each test run

## Troubleshooting

### "Tests pass locally but fail in CI"
- Check environment variables
- Verify dependencies are installed
- Look for timing/race conditions
- Check Docker resource limits

### "Go tests hang"
- Likely a deadlock or infinite loop
- Run with `-timeout 30s` flag
- Check for unclosed channels/goroutines

### "Frontend tests fail with module not found"
- Run `npm ci` instead of `npm install`
- Check jest.config.js moduleNameMapper
- Verify import paths

## Resources

- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Jest Documentation](https://jestjs.io/)
- [Playwright Documentation](https://playwright.dev/)
- [GitHub Actions](https://docs.github.com/en/actions)

## Contact

For test-related questions:
- Check CI logs in GitHub Actions
- Review this documentation
- Check existing test files for examples
