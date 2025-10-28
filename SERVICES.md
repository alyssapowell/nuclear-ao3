# Nuclear AO3 Services

This document describes the microservices architecture of Nuclear AO3.

## Services Overview

### Auth Service (Port 8081)
- **Purpose**: User authentication, authorization, and user management
- **Key Features**:
  - JWT-based authentication
  - OAuth integration (Google, Twitter, Discord)
  - User registration and login
  - Password reset and email verification
  - Role-based access control
- **Dependencies**: PostgreSQL, Redis

### Work Service (Port 8082)  
- **Purpose**: Fanfiction works, chapters, and content management
- **Key Features**:
  - Create, read, update, delete works
  - Chapter management
  - Series management
  - Collections and bookmarks
  - Comments and kudos
  - Work statistics tracking
- **Dependencies**: PostgreSQL, Redis

### Tag Service (Port 8083)
- **Purpose**: Tag management, wrangling, and hierarchies
- **Key Features**:
  - Tag creation and management
  - Tag wrangling system
  - Fandom, character, and relationship tags
  - Tag autocomplete and suggestions
  - Tag hierarchies and relationships
  - Popular and trending tags
- **Dependencies**: PostgreSQL, Redis

### Search Service (Port 8084)
- **Purpose**: Full-text search and discovery
- **Key Features**:
  - Work search with advanced filters
  - Tag, user, collection, and series search
  - Autocomplete and suggestions
  - Search analytics and trending
  - Faceted search results
  - Search history and saved searches
- **Dependencies**: PostgreSQL, Redis, Elasticsearch

### Notification Service (Port 8085)
- **Purpose**: User notifications and messaging system
- **Key Features**:
  - Real-time notifications via WebSocket
  - Email notifications and digests
  - Push notifications
  - User notification preferences
  - Notification history and management
  - Smart batching and rate limiting
- **Dependencies**: PostgreSQL, Redis

### Export Service (Port 8086)
- **Purpose**: Data export functionality
- **Key Features**:
  - Work export in multiple formats (EPUB, PDF, HTML)
  - Collection and series exports
  - User data exports for GDPR compliance
  - Bulk export operations
  - Export scheduling and management
  - Format optimization and compression
- **Dependencies**: PostgreSQL, Redis

## API Endpoints

Each service exposes REST APIs under `/api/v1/`. See individual service handlers for detailed endpoint documentation.

### Common Patterns

- **Health Check**: `GET /health` - Service health status
- **Metrics**: `GET /metrics` - Prometheus metrics
- **Authentication**: JWT tokens via Authorization header
- **Pagination**: `page` and `limit` query parameters
- **Error Responses**: Consistent JSON error format

### Cross-Service Communication

Services communicate via HTTP APIs. Future improvements could include:
- Service mesh (Istio)
- gRPC for internal communication
- Event-driven architecture with message queues

## Infrastructure

### Database Schema
- PostgreSQL with separate schemas per service concern
- Migrations handled via `migrations/` directory
- Connection pooling and health checks

### Caching Strategy
- Redis used for:
  - Session storage (Auth Service)
  - API response caching
  - Search analytics
  - Rate limiting
  - Popular/trending data

### Search Index
- Elasticsearch for full-text search
- Separate indices for works, tags, users, collections
- Real-time indexing on data changes

### Monitoring
- Prometheus metrics collection
- Grafana dashboards
- Health check endpoints
- Structured logging

## Deployment

### Development
```bash
docker-compose up -d
```

### Production Considerations
- Use managed databases (RDS, Cloud SQL)
- Container orchestration (Kubernetes)
- Load balancing and auto-scaling
- SSL termination
- Environment-specific configurations
- Backup and disaster recovery

## Security

### Authentication
- JWT tokens with RS256 signing
- Token refresh mechanism
- Role-based access control

### Data Protection
- Input validation and sanitization
- SQL injection prevention
- XSS protection headers
- Rate limiting
- CORS configuration

### Infrastructure Security
- Container security scanning
- Network policies
- Secrets management
- TLS encryption in transit
- Database encryption at rest

## Development Guidelines

### Code Organization
- Shared models in `backend/shared/models/`
- Middleware in `backend/shared/middleware/`
- Service-specific logic in respective directories

### Testing
- Unit tests for business logic
- Integration tests for API endpoints
- End-to-end tests for critical workflows

### API Design
- RESTful conventions
- Consistent error handling
- OpenAPI/Swagger documentation
- Versioning strategy

## Future Enhancements

### Performance
- Database read replicas
- CDN for static assets
- Query optimization
- Caching improvements

### Features
- Real-time notifications (WebSockets)
- Background job processing
- Admin dashboard
- API rate limiting per user

### Operations
- Automated deployments
- Blue-green deployments
- Canary releases
- Enhanced monitoring and alerting