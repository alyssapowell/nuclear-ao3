# Nuclear AO3 - Main Makefile
# Complete development and deployment commands

# Variables
COMPOSE_FILE=docker-compose.yml
COMPOSE_TEST_FILE=docker-compose.test.yml
BACKEND_DIR=backend
FRONTEND_DIR=frontend
MIGRATION_DIR=migrations

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build test clean deploy dev stop logs

help: ## Show this help message
	@echo 'Usage: make [TARGET]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Start all services for development
	@echo "$(GREEN)🚀 Starting Nuclear AO3 development environment...$(NC)"
	docker-compose up -d --build
	@echo "$(GREEN)✅ Development environment started!$(NC)"
	@echo "$(BLUE)📱 Frontend: http://localhost:3000$(NC)"
	@echo "$(BLUE)🔧 API Gateway: http://localhost:8080$(NC)"
	@echo "$(BLUE)🔐 Auth Service: http://localhost:8081$(NC)"
	@echo "$(BLUE)📚 Works Service: http://localhost:8082$(NC)"
	@echo "$(BLUE)🏷️  Tags Service: http://localhost:8083$(NC)"
	@echo "$(BLUE)🔍 Search Service: http://localhost:8084$(NC)"
	@echo "$(BLUE)📊 Grafana: http://localhost:3001 (admin/admin)$(NC)"
	@echo "$(BLUE)📈 Prometheus: http://localhost:9090$(NC)"

dev-build: ## Build and start development environment
	@echo "$(GREEN)🔨 Building Nuclear AO3...$(NC)"
	docker-compose build --no-cache
	@make dev

stop: ## Stop all services
	@echo "$(YELLOW)🛑 Stopping Nuclear AO3...$(NC)"
	docker-compose down
	@echo "$(GREEN)✅ All services stopped$(NC)"

restart: ## Restart all services
	@make stop
	@make dev

logs: ## View logs from all services
	docker-compose logs -f

logs-auth: ## View auth service logs
	docker-compose logs -f auth-service

logs-works: ## View work service logs  
	docker-compose logs -f work-service

logs-frontend: ## View frontend logs
	docker-compose logs -f frontend

# Build commands
build: ## Build all services
	@echo "$(GREEN)🔨 Building all services...$(NC)"
	docker-compose build

build-backend: ## Build backend services only
	@echo "$(GREEN)🔨 Building backend services...$(NC)"
	cd $(BACKEND_DIR) && make build-all

build-frontend: ## Build frontend only
	@echo "$(GREEN)🔨 Building frontend...$(NC)"
	cd $(FRONTEND_DIR) && npm run build

# Database commands
db-setup: ## Set up database with migrations
	@echo "$(GREEN)🗄️  Setting up database...$(NC)"
	docker-compose exec postgres psql -U ao3_user -d ao3_nuclear -f /docker-entrypoint-initdb.d/001_create_users_and_auth.sql
	docker-compose exec postgres psql -U ao3_user -d ao3_nuclear -f /docker-entrypoint-initdb.d/002_create_content_tables.sql
	@echo "$(GREEN)✅ Database setup complete$(NC)"

db-reset: ## Reset database (WARNING: destroys all data)
	@echo "$(RED)⚠️  WARNING: This will destroy all data! Press Ctrl+C to cancel...$(NC)"
	@sleep 5
	docker-compose stop postgres
	docker volume rm nuclear-ao3_postgres_data || true
	docker-compose up -d postgres
	@sleep 5
	@make db-setup

db-seed: ## Seed database with test data
	@echo "$(GREEN)🌱 Seeding database...$(NC)"
	cd $(BACKEND_DIR) && make seed-all
	@echo "$(GREEN)✅ Database seeded$(NC)"

db-migrate: ## Run database migrations
	@echo "$(GREEN)🗄️  Running migrations...$(NC)"
	cd $(BACKEND_DIR) && make migrate-all

# Testing commands
test: ## Run all tests
	@echo "$(GREEN)🧪 Running all tests...$(NC)"
	@make test-backend
	@make test-frontend
	@echo "$(GREEN)✅ All tests passed!$(NC)"

test-backend: ## Run backend tests
	@echo "$(GREEN)🧪 Running backend tests...$(NC)"
	cd $(BACKEND_DIR) && make test-all

test-frontend: ## Run frontend tests
	@echo "$(GREEN)🧪 Running frontend tests...$(NC)"
	cd $(FRONTEND_DIR) && npm test

test-integration: ## Run integration tests
	@echo "$(GREEN)🧪 Running integration tests...$(NC)"
	docker-compose -f $(COMPOSE_TEST_FILE) up --build --abort-on-container-exit
	docker-compose -f $(COMPOSE_TEST_FILE) down

test-load: ## Run load tests
	@echo "$(GREEN)⚡ Running load tests...$(NC)"
	cd $(BACKEND_DIR) && make load-test-all
	@echo "$(GREEN)📊 Load test results available in backend/results/$(NC)"

test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)🔄 Running E2E tests...$(NC)"
	cd $(FRONTEND_DIR) && npm run test:e2e

# Performance and monitoring
benchmark: ## Run performance benchmarks
	@echo "$(GREEN)⚡ Running performance benchmarks...$(NC)"
	cd $(BACKEND_DIR) && make benchmark-all
	@echo "$(GREEN)📊 Benchmark results:$(NC)"
	@cd $(BACKEND_DIR) && make benchmark-report

monitor: ## Start monitoring stack
	@echo "$(GREEN)📊 Starting monitoring stack...$(NC)"
	docker-compose up -d prometheus grafana loki
	@echo "$(GREEN)📈 Monitoring available:$(NC)"
	@echo "$(BLUE)  Grafana: http://localhost:3001$(NC)"
	@echo "$(BLUE)  Prometheus: http://localhost:9090$(NC)"

health: ## Check health of all services
	@echo "$(GREEN)🏥 Checking service health...$(NC)"
	@echo "$(YELLOW)Auth Service:$(NC)"
	@curl -f http://localhost:8081/health || echo "❌ Auth service unhealthy"
	@echo "$(YELLOW)Work Service:$(NC)"
	@curl -f http://localhost:8082/health || echo "❌ Work service unhealthy"
	@echo "$(YELLOW)Tag Service:$(NC)" 
	@curl -f http://localhost:8083/health || echo "❌ Tag service unhealthy"
	@echo "$(YELLOW)Search Service:$(NC)"
	@curl -f http://localhost:8084/health || echo "❌ Search service unhealthy"
	@echo "$(YELLOW)API Gateway:$(NC)"
	@curl -f http://localhost:8080/health || echo "❌ API Gateway unhealthy"
	@echo "$(YELLOW)Frontend:$(NC)"
	@curl -f http://localhost:3000 || echo "❌ Frontend unhealthy"

# Performance comparison
compare-ao3: ## Compare performance with original AO3
	@echo "$(GREEN)⚡ Comparing performance with AO3...$(NC)"
	cd $(BACKEND_DIR) && make compare-ao3
	@echo "$(GREEN)📊 Performance comparison complete$(NC)"

# Security
security-scan: ## Run security scans
	@echo "$(GREEN)🔒 Running security scans...$(NC)"
	docker run --rm -v $$(pwd):/app securecodewarrior/docker-image-security-scan /app
	cd $(BACKEND_DIR) && make security-scan
	cd $(FRONTEND_DIR) && npm audit

# Deployment commands
deploy-staging: ## Deploy to staging environment
	@echo "$(GREEN)🚀 Deploying to staging...$(NC)"
	./scripts/deploy-staging.sh

deploy-production: ## Deploy to production environment
	@echo "$(GREEN)🚀 Deploying to production...$(NC)"
	./scripts/deploy-production.sh

# Kubernetes commands
k8s-deploy: ## Deploy to Kubernetes
	@echo "$(GREEN)☸️  Deploying to Kubernetes...$(NC)"
	kubectl apply -f k8s/
	@echo "$(GREEN)✅ Kubernetes deployment complete$(NC)"

k8s-delete: ## Delete from Kubernetes
	@echo "$(YELLOW)☸️  Deleting from Kubernetes...$(NC)"
	kubectl delete -f k8s/

k8s-status: ## Check Kubernetes deployment status
	@echo "$(GREEN)☸️  Kubernetes deployment status:$(NC)"
	kubectl get pods -l app=nuclear-ao3
	kubectl get services -l app=nuclear-ao3

# Maintenance commands
clean: ## Clean up Docker containers and images
	@echo "$(YELLOW)🧹 Cleaning up...$(NC)"
	docker-compose down -v --remove-orphans
	docker system prune -f
	docker volume prune -f
	@echo "$(GREEN)✅ Cleanup complete$(NC)"

backup: ## Backup database and volumes
	@echo "$(GREEN)💾 Creating backup...$(NC)"
	./scripts/backup.sh
	@echo "$(GREEN)✅ Backup complete$(NC)"

restore: ## Restore database and volumes
	@echo "$(GREEN)📥 Restoring from backup...$(NC)"
	./scripts/restore.sh
	@echo "$(GREEN)✅ Restore complete$(NC)"

# Documentation
docs: ## Generate documentation
	@echo "$(GREEN)📚 Generating documentation...$(NC)"
	cd $(BACKEND_DIR) && make docs
	cd $(FRONTEND_DIR) && npm run docs
	@echo "$(GREEN)📖 Documentation available in docs/$(NC)"

# Quality checks
lint: ## Run linters on all code
	@echo "$(GREEN)🔍 Running linters...$(NC)"
	cd $(BACKEND_DIR) && make lint-all
	cd $(FRONTEND_DIR) && npm run lint

format: ## Format all code
	@echo "$(GREEN)✨ Formatting code...$(NC)"
	cd $(BACKEND_DIR) && make format-all
	cd $(FRONTEND_DIR) && npm run format

quality: ## Run all quality checks
	@make lint
	@make test
	@make security-scan
	@echo "$(GREEN)✅ Quality checks passed$(NC)"

# Development helpers
setup: ## Initial setup for development
	@echo "$(GREEN)🏗️  Setting up Nuclear AO3 development environment...$(NC)"
	@echo "$(YELLOW)1. Installing dependencies...$(NC)"
	cd $(BACKEND_DIR) && make install-deps
	cd $(FRONTEND_DIR) && npm install
	@echo "$(YELLOW)2. Starting services...$(NC)"
	@make dev
	@echo "$(YELLOW)3. Setting up database...$(NC)"
	@sleep 10 # Wait for DB to be ready
	@make db-setup
	@make db-seed
	@echo "$(GREEN)🎉 Setup complete! Nuclear AO3 is ready for development$(NC)"
	@echo "$(BLUE)Visit http://localhost:3000 to see the application$(NC)"

reset: ## Reset everything to clean state
	@make clean
	@make setup

# Release commands
release: ## Create a release build
	@echo "$(GREEN)📦 Creating release build...$(NC)"
	@make clean
	@make build
	@make test
	@make benchmark
	@echo "$(GREEN)🎉 Release build complete!$(NC)"

# Statistics
stats: ## Show project statistics
	@echo "$(GREEN)📊 Nuclear AO3 Statistics:$(NC)"
	@echo "$(YELLOW)Backend:$(NC)"
	@find $(BACKEND_DIR) -name "*.go" | xargs wc -l | tail -1 | awk '{print "  Go files: " $$1 " lines"}'
	@find $(BACKEND_DIR) -name "*_test.go" | wc -l | awk '{print "  Test files: " $$1}'
	@echo "$(YELLOW)Frontend:$(NC)"
	@find $(FRONTEND_DIR) -name "*.tsx" -o -name "*.ts" | xargs wc -l | tail -1 | awk '{print "  TypeScript files: " $$1 " lines"}'
	@find $(FRONTEND_DIR) -name "*.test.*" | wc -l | awk '{print "  Test files: " $$1}'
	@echo "$(YELLOW)Database:$(NC)"
	@find $(MIGRATION_DIR) -name "*.sql" | wc -l | awk '{print "  Migration files: " $$1}'
	@echo "$(YELLOW)Docker:$(NC)"
	@docker-compose ps --services | wc -l | awk '{print "  Services: " $$1}'

# Demo commands
demo: ## Set up demo environment with sample data
	@echo "$(GREEN)🎭 Setting up demo environment...$(NC)"
	@make setup
	cd $(BACKEND_DIR) && make demo-data
	@echo "$(GREEN)🎉 Demo environment ready!$(NC)"
	@echo "$(BLUE)Demo users created:$(NC)"
	@echo "  Admin: admin@nuclear-ao3.demo / password123"
	@echo "  Author: author@nuclear-ao3.demo / password123"
	@echo "  Reader: reader@nuclear-ao3.demo / password123"

# Container registry commands
push: ## Push images to container registry
	@echo "$(GREEN)📤 Pushing images to registry...$(NC)"
	./scripts/push-images.sh

pull: ## Pull images from container registry
	@echo "$(GREEN)📥 Pulling images from registry...$(NC)"
	./scripts/pull-images.sh