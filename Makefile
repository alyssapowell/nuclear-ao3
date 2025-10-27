# Nuclear AO3 Development Makefile
# Provides convenient commands for database management, testing, and development

.PHONY: help db-flush db-populate db-reset db-status test-data dev-setup clean build test services logs

# Default target
help: ## Show this help message
	@echo "Nuclear AO3 Development Commands"
	@echo "================================="
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Database Management
db-status: ## Check database status and connection
	@echo "🔍 Checking database status..."
	@docker-compose ps postgres elasticsearch redis
	@echo "\n📊 Database statistics:"
	@docker exec ao3_postgres psql -U ao3_user -d ao3_nuclear -c "\
		SELECT 'Users' as table, COUNT(*) as count FROM users \
		UNION ALL SELECT 'Works', COUNT(*) FROM works \
		UNION ALL SELECT 'Tags', COUNT(*) FROM tags \
		UNION ALL SELECT 'Work-Tag Relations', COUNT(*) FROM work_tags;" 2>/dev/null || echo "Database not ready"

db-flush: ## Completely flush all databases (DESTRUCTIVE)
	@echo "🗑️  Flushing all databases..."
	@echo "⚠️  This will DELETE ALL DATA. Press Ctrl+C to cancel, Enter to continue..."
	@read
	@docker-compose down
	@docker volume rm nuclear-ao3_postgres_data nuclear-ao3_elasticsearch_data nuclear-ao3_redis_data 2>/dev/null || true
	@echo "✅ Databases flushed"

db-start: ## Start database services only
	@echo "🚀 Starting database services..."
	@docker-compose up -d postgres elasticsearch redis
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 15
	@make db-status

db-populate: ## Populate databases with test data (requires clean DB)
	@echo "📚 Populating databases with test data..."
	@cd tools && node populate-databases.js

db-reset: db-flush db-start ## Complete database reset and repopulation
	@echo "🔄 Complete database reset..."
	@sleep 10
	@make db-populate
	@echo "✅ Database reset complete"

# Test Data Management
test-data-generate: ## Generate new test dataset (15k works)
	@echo "🎯 Generating comprehensive test dataset..."
	@cd tools && node comprehensive-data-generator.js

test-data-small: ## Generate small test dataset (100 works)
	@echo "🎯 Generating small test dataset..."
	@cd tools && node -e "const gen = require('./comprehensive-data-generator.js'); const g = new gen(); g.generateAllWorks(100); g.saveToFile('small-test-data.json');"

# Service Management
services: ## Start all Nuclear AO3 services
	@echo "🚀 Starting all Nuclear AO3 services..."
	@docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@make service-status

services-dev: ## Start services in development mode (with logs)
	@echo "🚀 Starting services in development mode..."
	@docker-compose up

service-status: ## Check status of all services
	@echo "🔍 Service Status:"
	@docker-compose ps
	@echo "\n🏥 Health Check:"
	@curl -s http://localhost:8080/health | jq '.' 2>/dev/null || echo "API Gateway not ready"

logs: ## Show logs for all services
	@docker-compose logs -f

logs-db: ## Show database logs only
	@docker-compose logs -f postgres

# Development Setup
dev-setup: ## Complete development environment setup
	@echo "🛠️  Setting up development environment..."
	@make db-reset
	@make services
	@cd frontend && npm install
	@echo "✅ Development environment ready!"
	@echo "🌐 Frontend: http://localhost:3000"
	@echo "🔌 API Gateway: http://localhost:8080"
	@echo "📊 Grafana: http://localhost:3002"

frontend-dev: ## Start frontend development server
	@echo "🌐 Starting frontend development server..."
	@cd frontend && npm run dev

# Testing
test: ## Run all tests
	@echo "🧪 Running tests..."
	@cd backend && go test ./... -v

test-api: ## Test API endpoints with sample data
	@echo "🔌 Testing API endpoints..."
	@curl -s "http://localhost:8080/health" | jq '.'
	@echo "\n📚 Sample works:"
	@curl -s "http://localhost:8080/api/v1/works/?limit=3" | jq '.works | length'
	@echo "\n🔍 Search test:"
	@curl -s "http://localhost:8080/api/v1/search?q=Harry&limit=5" | jq '.' || echo "Search service not ready"

# Build and Deployment
build: ## Build all services
	@echo "🔨 Building all services..."
	@docker-compose build

build-no-cache: ## Build all services without cache
	@echo "🔨 Building all services (no cache)..."
	@docker-compose build --no-cache

# Cleanup
clean: ## Clean up containers and unused resources
	@echo "🧹 Cleaning up..."
	@docker-compose down
	@docker system prune -f
	@docker volume prune -f

clean-all: ## Complete cleanup including images and volumes
	@echo "🧹 Complete cleanup..."
	@docker-compose down --rmi all --volumes
	@docker system prune -a -f

# Data Management
backup-db: ## Backup PostgreSQL database
	@echo "💾 Backing up PostgreSQL database..."
	@mkdir -p backups
	@docker exec ao3_postgres pg_dump -U ao3_user ao3_nuclear > backups/ao3_backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup saved to backups/ directory"

restore-db: ## Restore PostgreSQL database (requires BACKUP_FILE variable)
	@if [ -z "$(BACKUP_FILE)" ]; then echo "❌ Please specify BACKUP_FILE=filename"; exit 1; fi
	@echo "📥 Restoring database from $(BACKUP_FILE)..."
	@docker exec -i ao3_postgres psql -U ao3_user ao3_nuclear < $(BACKUP_FILE)
	@echo "✅ Database restored"

# Performance and Monitoring
monitor: ## Open monitoring dashboard
	@echo "📊 Opening Grafana dashboard..."
	@open http://localhost:3002 || echo "Open http://localhost:3002 manually"

performance-test: ## Run performance tests with large dataset
	@echo "⚡ Running performance tests..."
	@make test-data-generate
	@make db-reset
	@cd tools && time node populate-databases.js
	@make test-api

# Quick Development Workflows
quick-reset: ## Quick reset for development (keeps containers running)
	@echo "🔄 Quick development reset..."
	@docker exec ao3_postgres psql -U ao3_user -d ao3_nuclear -c "TRUNCATE TABLE work_tags, work_statistics, works, tags, users RESTART IDENTITY CASCADE;"
	@curl -X DELETE http://localhost:9200/works 2>/dev/null || true
	@make db-populate

demo-data: ## Generate and load demo data optimized for demonstrations
	@echo "🎭 Setting up demo data..."
	@cd tools && node -e "\
		const gen = require('./comprehensive-data-generator.js');\
		const g = new gen();\
		g.generateAllWorks(1000);\
		g.saveToFile('demo-data.json');"
	@make quick-reset

# Examples and Documentation
examples: ## Show example API calls
	@echo "📖 Example API Calls:"
	@echo ""
	@echo "Health Check:"
	@echo "curl http://localhost:8080/health"
	@echo ""
	@echo "List Works:"
	@echo "curl 'http://localhost:8080/api/v1/works/?limit=10'"
	@echo ""
	@echo "Search Works:"
	@echo "curl 'http://localhost:8080/api/v1/search?q=Agatha&limit=5'"
	@echo ""
	@echo "Filter by Fandom:"
	@echo "curl 'http://localhost:8080/api/v1/works/?fandom=Harry%20Potter&limit=10'"

# CI/CD Helpers
ci-test: ## Run tests suitable for CI environment
	@echo "🤖 Running CI tests..."
	@make db-flush
	@make demo-data
	@make test
	@make test-api

# Version and Info
version: ## Show version information
	@echo "Nuclear AO3 Development Environment"
	@echo "==================================="
	@echo "Docker version: $(shell docker --version)"
	@echo "Docker Compose version: $(shell docker-compose --version)"
	@echo "Node.js version: $(shell node --version 2>/dev/null || echo 'Not installed')"
	@echo "Go version: $(shell go version 2>/dev/null || echo 'Not installed')"