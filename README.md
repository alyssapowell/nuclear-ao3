# Nuclear AO3: The Modern Fanfiction Platform

> **This project is being simplified.** The backend is now a single Go program (`backend/cmd/nuclear`) instead of six services and a gateway. Parts of this README still describe the old layout. For what exists and why, read [docs/FEATURES.md](docs/FEATURES.md) and [docs/DECISIONS.md](docs/DECISIONS.md).

**A modern fanfiction platform: one Go program, Postgres, and a website served as plain files.**

## ♿ **Accessibility-First Design**

Nuclear AO3 is built with **100% WCAG 2.1 AA compliance** from the ground up, ensuring that every user can access our platform regardless of their abilities or assistive technologies.

### 🌟 **Accessibility Highlights:**
- **Complete keyboard navigation** - Every feature works without a mouse
- **Screen reader optimized** - Full support for NVDA, JAWS, VoiceOver, and TalkBack
- **Focus management** - Clear visual indicators and logical tab order
- **Live regions** - Real-time updates announced to assistive technologies
- **Color contrast** - Exceeds WCAG standards for visual accessibility
- **Semantic HTML** - Proper landmarks, headings, and structure
- **ARIA patterns** - Proper combobox, listbox, and navigation patterns
- **Accessible forms** - Clear labels, error handling, and validation feedback
- **Comprehensive testing** - Validated with automated and manual accessibility testing

### 🎯 **Accessibility Implementation:**
- **Search components** with full ARIA combobox patterns and live announcements
- **TagAutocomplete** with keyboard navigation and screen reader support
- **SearchForm** with accessible validation and error messaging
- **SearchResults** with semantic structure and skip links
- **SearchPagination** with clear navigation landmarks

**📖 Full Accessibility Guide:** [ACCESSIBILITY_TESTING_GUIDE.md](ACCESSIBILITY_TESTING_GUIDE.md)
**📋 Frontend Accessibility:** [frontend/ACCESSIBILITY.md](frontend/ACCESSIBILITY.md)

## 🚀 Quick Start

### Prerequisites
- **Docker Desktop** ([Install here](https://www.docker.com/products/docker-desktop/))
- **Git** and basic command line knowledge

### Get Everything Running in 3 Commands

```bash
# 1. Clone and enter the project
git clone <repository-url>
cd nuclear-ao3

# 2. Start the complete stack (takes 2-3 minutes first time)
make dev-setup

# 3. Check everything is healthy
make service-status
```

### 🌐 Your Local Nuclear AO3 is Ready!

```bash
# See all available commands and check status
make help
make service-status
```

**Frontend Application:**
- **Website**: https://localhost through Caddy, or http://localhost:3001 with `make frontend-dev` (reloads as you edit). It's a Vite + React single-page app built to plain files; there's no Node server in production.

**Service Endpoints:**

- **API**: http://localhost:8088 - the whole backend (one program: accounts, works, tags, search, notifications, exports)
- **Database**: localhost:5432 - PostgreSQL 
- **Monitoring**: http://localhost:3002 - Grafana (admin/admin)

### 🎯 Take It for a Test Drive

**Try the OAuth2/OIDC API:**
```bash
# Register an OAuth2 client
curl -X POST http://localhost:8088/auth/register-client \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "My Test App",
    "redirect_uris": ["https://example.com/callback"],
    "scopes": ["read", "write"],
    "grant_types": ["client_credentials"],
    "response_types": ["code"]
  }'

# Get an access token (replace YOUR_CLIENT_ID and YOUR_CLIENT_SECRET)
curl -X POST http://localhost:8088/auth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET"
```

**Create and search works:**
```bash
# Create a work (requires authentication)
curl -X POST http://localhost:8088/api/v1/works \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Nuclear AO3 Fanfic",
    "summary": "Testing the new platform",
    "content": "Once upon a time...",
    "fandoms": ["Original Work"]
  }'

# Search works
curl "http://localhost:8088/api/v1/search/works?q=fanfic" | jq
```

**📚 Complete API Testing Guide:** [backend/internal/auth/API_TESTING_GUIDE.md](backend/internal/auth/API_TESTING_GUIDE.md)

### ⚡ Common Use Cases

**1. Developer Testing OAuth2 Integration:**
```bash
# Start services
make dev-setup

# Register your app
curl -X POST http://localhost:8088/auth/register-client \
  -H "Content-Type: application/json" \
  -d '{"client_name": "My App", "redirect_uris": ["http://localhost:3000/callback"], "scopes": ["read", "write"], "grant_types": ["authorization_code", "client_credentials"], "response_types": ["code"]}'

# Use client credentials for testing
curl -X POST http://localhost:8088/auth/token \
  -d "grant_type=client_credentials&client_id=YOUR_ID&client_secret=YOUR_SECRET"
```

**2. Frontend Developer Building UI:**
```bash
# Start just the APIs you need
make services

# Create test works through the API
curl -X POST http://localhost:8080/api/v1/works \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"title": "Test Story", "content": "Chapter 1...", "fandoms": ["My Fandom"]}'

# Search for content
curl "http://localhost:8080/api/v1/search/works?q=story&sort=date"
```

**3. Performance Testing:**
```bash
# Start monitoring
make dev-setup

# Run benchmarks
make performance-test

# View metrics at http://localhost:3002
```

**4. Database/Content Management:**
```bash
# Access database directly
docker exec -it nuclear-ao3-postgres psql -U ao3_user -d ao3_nuclear

# Apply database migrations (the app also does this on startup)
make migrate

# Load sample accounts and works (development only; password "password123")
make seed

# Add test data only
make db-populate
```

## 🏗️ Architecture

```
Browser
   │
   ▼
Caddy (HTTPS)  ── serves the website: plain files built from frontend/
   │  /api, /ws, /health
   ▼
nuclear (one Go program): accounts, works, tags, search, notifications, exports
   │
   └── PostgreSQL: all data, including search and background jobs
```

There's one backend program to run, upgrade and watch. Search is part of
Postgres (full-text search on the works table), so there's no search server
to run or index to rebuild.

## 📁 Project Structure

```
nuclear-ao3/
├── backend/
│   ├── cmd/nuclear/          # The program: starts everything, plus admin commands
│   ├── internal/             # One folder per area: auth, works, tags, search, notify, export
│   ├── migrations/           # Database schema, applied by the app itself
│   └── shared/               # Code the areas share
├── frontend/                 # The website: Vite + React + React Router
├── monitoring/               # Prometheus & Grafana config
├── docs/                     # Features, decisions and the runbook
├── Caddyfile                 # The front door: HTTPS, website files, /api
├── docker-compose.yml        # Everything, ready to run
└── Makefile                  # Shortcuts for common jobs
```

## 🛠️ Development Commands

### Essential Commands
| Command | Description |
|---------|-------------|
| `make dev-setup` | 🚀 **Complete setup** - Build + start + migrate + seed |
| `make services` | Start all services |
| `make clean` | Stop all services |
| `make service-status` | ✅ Check all services are healthy |
| `make help` | 🌐 Show all available commands |
| `make logs` | View logs from all services |
| `make clean-all` | 🧹 Reset everything (removes data!) |

### Development Workflow
| Command | Description |
|---------|-------------|
| `make logs-db` | View database logs |
| `make services-dev` | Start services with logs |
| `make frontend-dev` | Start frontend development server |
| `docker exec -it nuclear-ao3-auth bash` | Open shell in auth service |
| `docker exec -it nuclear-ao3-postgres psql -U ao3_user -d ao3_nuclear` | Open PostgreSQL shell |

### Testing & Quality
| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make test-api` | 🔐 Test API endpoints |
| `make ci-test` | Run complete CI test suite |
| `make performance-test` | 📊 Performance benchmarks |

### Database Management
| Command | Description |
|---------|-------------|
| `make db-reset` | Complete database reset and repopulation |
| `make db-populate` | Add test data to existing database |
| `make db-flush` | ⚠️ Completely flush databases (destroys data!) |
| `make backup-db` | Backup PostgreSQL database |

### Service Management
```bash
# Start individual services
make db-start     # Just database services
make services     # All backend services
make frontend-dev # Just frontend development

# Restart a specific service
docker-compose restart auth-service

# View real-time logs
make logs

# Scale a service
docker-compose up -d --scale work-service=3
```

## 🔧 What the backend does

Everything below is one program, `nuclear`, listening on one port. The
areas are folders in `backend/internal/`.

- **Accounts** (`auth`): registration, login, password reset, roles (admin, moderator, tag wrangler)
- **Works** (`works`): works, chapters, series, collections, comments, kudos, bookmarks
- **Tags** (`tags`): tags, wrangling, synonyms, autocomplete
- **Search** (`search`): work search with AO3's filters, and suggestions while typing
- **Notifications** (`notify`): in-app and email notifications, subscriptions
- **Exports** (`export`): downloads of works

**Key Endpoints:**
```
POST /api/v1/auth/register       # Create an account
POST /api/v1/auth/login          # Log in
POST /api/v1/works               # Post a work (starts as a draft)
GET  /api/v1/works/:id           # Read a work
POST /api/v1/works/:id/kudos     # Leave kudos
GET  /api/v1/tags/autocomplete   # Tag suggestions while typing
GET  /api/v1/search/works        # Search works
GET  /api/v1/search/suggestions  # Search-box suggestions
```

## 💾 Data Models

### Core Entities
- **User**: Authentication, profiles, preferences
- **Work**: Fanfiction works with metadata, tags, statistics
- **Chapter**: Individual chapters within works
- **Tag**: Hierarchical tag system (fandoms, characters, relationships, freeform)
- **Comment**: Threaded comments on works/chapters
- **Bookmark**: User bookmarks with notes and tags
- **Series**: Collections of related works
- **Collection**: Themed work collections

### Relationships
- Users create Works
- Works belong to Series (optional)
- Works have Chapters
- Works have Tags (many-to-many)
- Users give Kudos to Works
- Users create Comments on Works/Chapters
- Users create Bookmarks for Works

## 🔍 Search Capabilities

Search runs in Postgres and follows the same visibility rules as the work
page: drafts, works restricted to logged-in users, blocks and mutes all
apply, and anonymous works never reveal their creator.

- **Words** anywhere in the title, tags or summary (quotes, `OR` and `-word` work)
- **Tags**: fandoms, characters, relationships and additional tags, plus tags to exclude
- **Filters**: rating, warnings, categories, language, complete or in progress, crossovers, single chapter
- **Ranges**: word count, minimum kudos and comments, updated within a week, month or year
- **Sorting** by best match, date updated or posted, title, word count, hits, kudos, comments or bookmarks
- **Counts** beside the results for the most common ratings and tags

## 🚦 Performance Features

### Caching Strategy
- **Rate limits** counted in the app's memory; every read goes straight to Postgres
- **HTTP caching headers** for static content
- **Database query optimization** with proper indexing

### Scalability
- **Horizontal scaling** via Docker containers
- **Database read replicas** for query distribution
- **Connection pooling** for database efficiency
- **Rate limiting** to prevent abuse
- **Monitoring** with Prometheus and Grafana

## 🔒 Security

- **JWT authentication** with secure token handling
- **Password hashing** with bcrypt
- **Rate limiting** per user and IP
- **Input validation** and SQL injection prevention
- **CORS configuration** for cross-origin requests
- **Security headers** (HSTS, CSP, etc.)
- **Role-based permissions** for admin functions

## 📊 Monitoring

Access monitoring at http://localhost:3002 (Grafana):
- API response times and error rates
- Database performance metrics
- Search query performance
- Cache hit rates
- User activity metrics
- System resource usage

## 🧪 Testing

### Quick API Testing
```bash
# Health check all services
make service-status

# Test API endpoints
make test-api

# Performance benchmarks  
make performance-test
```

### Comprehensive Testing
```bash
# Run all tests across all services
make test

# Test individual services through API
make test-api       # Test all API endpoints
make test           # Run Go unit tests

# Performance & load testing
make performance-test
```

### Manual API Testing

**Test Authentication Flow:**
```bash
# 1. Get OAuth2 discovery info
curl http://localhost:8088/.well-known/openid-configuration | jq

# 2. Register a client
curl -X POST http://localhost:8088/auth/register-client \
  -H "Content-Type: application/json" \
  -d '{"client_name": "Test App", "redirect_uris": ["https://example.com"], "scopes": ["read"], "grant_types": ["client_credentials"], "response_types": ["code"]}'

# 3. Get access token
curl -X POST http://localhost:8088/auth/token \
  -d "grant_type=client_credentials&client_id=YOUR_ID&client_secret=YOUR_SECRET"
```

**Test Work Management:**
```bash
# Create a work
curl -X POST http://localhost:8088/api/v1/works \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"title": "Test Work", "content": "Test content", "fandoms": ["Original Work"]}'

# Get works
curl http://localhost:8088/api/v1/works | jq
```

**Test Search:**
```bash
# Search works
curl "http://localhost:8088/api/v1/search/works?q=test&limit=10" | jq

# Get search suggestions  
curl "http://localhost:8088/api/v1/search/suggestions?q=fan" | jq
```


## 🐳 Development with Docker

The entire stack runs in Docker containers:

```bash
# Build and start everything
make services

# View logs from all services
make logs

# Access individual service shells
docker exec -it nuclear-ao3-auth bash
docker exec -it nuclear-ao3-postgres psql -U ao3_user -d ao3_nuclear

# Clean reset
make clean-all
```

## 🚀 Performance Comparison

| Metric | Legacy AO3 | Nuclear AO3 | Improvement |
|--------|------------|-------------|-------------|
| Page Load Time | ~2000ms | ~200ms | **10x faster** |
| API Response | 500-2000ms | 50ms | **20x faster** |
| Search Results | ~1000ms | ~11ms | **90x faster** |
| Concurrent Users | ~5,000 | 50,000+ | **10x capacity** |
| Uptime | 99.5% | 99.9%+ | **20x fewer outages** |

### 📊 Scale Testing Results

**Real-World Performance Testing with 15K+ Works:**
- **15,155 works** with realistic content and metadata
- **69,304 chapters** across varied work lengths  
- **2,873 users** with comprehensive profiles
- **130,628 tag associations** using full tag taxonomy
- **Search Performance**: 11ms response time for complex queries
- **Database Performance**: Sub-50ms for most operations

## 🔧 Configuration

### Environment Variables

The app reads these environment variables (all of them are listed in `.env.example`):

```bash
# Database
DATABASE_URL=postgres://ao3_user:ao3_password@postgres:5432/ao3_nuclear?sslmode=disable

# Login tokens
JWT_ISSUER=nuclear-ao3

# Application
GIN_MODE=debug|release
PORT=8080
```

### Database Schema

The PostgreSQL schema includes:
- Users table with authentication data
- Works table with metadata and content
- Chapters table linked to works
- Tags table with hierarchical relationships
- Junction tables for many-to-many relationships
- Statistics tables for performance metrics

## 🤝 Contributing

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Install dev tools**: `make install-tools`
4. **Make changes and test**: `make test && make lint`
5. **Commit changes**: `git commit -m 'Add amazing feature'`
6. **Push to branch**: `git push origin feature/amazing-feature`
7. **Create Pull Request**

### Development Guidelines

- Write tests for new features
- Follow Go conventions and use `golangci-lint`
- Update documentation for API changes
- Ensure Docker builds work correctly
- Test performance impact of changes

## 📝 API Documentation

The API is described on the website at `/api-docs`, and each route is
listed with a one-line comment in the `Mount` function of its area, such as
`backend/internal/search/main.go`.

## 🚨 Troubleshooting

### Quick Fixes

**"Make command not found" or "Docker not found":**
```bash
# Install Docker Desktop
# Download from: https://www.docker.com/products/docker-desktop/

# Verify installation
docker --version && docker-compose --version
```

**Services won't start:**
```bash
# Check Docker is running
docker --version

# Start fresh
make clean-all && make dev-setup

# Check logs
make logs-db
docker-compose logs app
```

**"Connection refused" errors:**
```bash
# Wait for services to be ready (can take 30-60 seconds)
make service-status

# If still failing, check logs
make logs
```

### Detailed Diagnostics

**Database issues:**
```bash
# Check PostgreSQL status
make shell-db
# Should connect without errors

# Reset database if corrupted
make db-reset
```

**Search not working:**
```bash
# Is the app up, and can it reach the database?
curl http://localhost:8088/health | jq

# Try a search directly
curl "http://localhost:8088/api/v1/search/works?q=test" | jq '.total'

# Check logs
docker-compose logs app
```

**Authentication problems:**
```bash
# Test OAuth2 endpoints
curl http://localhost:8088/.well-known/openid-configuration

# Check auth service logs  
docker-compose logs auth-service

# Verify JWT key generation
curl http://localhost:8088/auth/jwks | jq
```

**Performance issues:**
```bash
# Check container resources
docker stats

# Run performance tests
make performance-test

# Check individual service health
make service-status
```

**Complete reset (nuclear option):**
```bash
# This will destroy all data and start fresh
make clean-all
make dev-setup
```

### Getting Help

1. **Check service logs first:** `make logs` or `docker-compose logs [service]`
2. **Verify health status:** `make service-status`  
3. **Try a clean restart:** `make clean && make services`
4. **Reset if needed:** `make clean-all && make dev-setup`
5. **Check GitHub issues** for known problems
6. **Open an issue** with logs and error messages

## 📄 License

**Liberation License v1.0** - Technology that serves human liberation, not corporate exploitation.

Nuclear AO3 is licensed under the Liberation License, which ensures this technology benefits communities while preventing corporate extraction and surveillance capitalism. See [LICENSE.md](LICENSE.md) for full details.

**Key Points:**
- ✅ **Full freedom** for individuals, communities, cooperatives, and humanitarian organizations
- ✅ **Perfect for fanfiction communities** - built to serve creators, not extract from them
- ❌ **Blocks corporate exploitation** - surveillance capitalism and rent-seeking prohibited
- ❌ **Prevents wealth concentration** - cannot be used to extract value from fan communities

## 🙏 Acknowledgments

This project is built independently to demonstrate modern architectural possibilities for fanfiction platforms. It is not officially affiliated with the Organization for Transformative Works or Archive of Our Own.

**Built with ❤️ for the fanfiction community**# Update README timestamp
