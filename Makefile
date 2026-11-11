# Nuclear AO3 Development Makefile
# Provides convenient commands for database management, testing, and development

.PHONY: ops-diagnose ops-emails ops-backup ops-upgrade ops-restore ops-logout-everyone ops-logout-user backup-db restore-db help keys keys-oidc keys-show keys-setup migrate migrate-status seed grant jobs jobs-retry db-flush db-populate db-reset db-status test-data dev-setup clean build test services logs

# Default target
help: ## Show this help message
	@echo "Nuclear AO3 Development Commands"
	@echo "================================="
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Database Management
db-status: ## Check database status and connection
	@echo "🔍 Checking database status..."
	@docker-compose ps postgres
	@echo "\n📊 Database statistics:"
	@docker exec nuclear-ao3-postgres psql -U ao3_user -d ao3_nuclear -c "\
		SELECT 'Users' as table, COUNT(*) as count FROM users \
		UNION ALL SELECT 'Works', COUNT(*) FROM works \
		UNION ALL SELECT 'Tags', COUNT(*) FROM tags \
		UNION ALL SELECT 'Work-Tag Relations', COUNT(*) FROM work_tags;" 2>/dev/null || echo "Database not ready"

db-flush: ## Completely flush all databases (DESTRUCTIVE)
	@echo "🗑️  Flushing all databases..."
	@echo "⚠️  This will DELETE ALL DATA. Press Ctrl+C to cancel, Enter to continue..."
	@read
	@docker-compose down
	@docker volume rm nuclear-ao3_postgres_data 2>/dev/null || true
	@echo "✅ Databases flushed"

db-start: ## Start database services only
	@echo "🚀 Starting database services..."
	@docker-compose up -d postgres
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 15
	@make db-status

migrate: ## Apply database migrations (the app also does this when it starts)
	@docker-compose run --rm app nuclear migrate

migrate-status: ## List migrations that haven't been applied yet
	@docker-compose run --rm app nuclear migrate status

# The same command a hosted trial uses (DEMO_DATA=true, see docs/HOSTING.md),
# run once in a container of its own with trial mode on for just that run. It
# is for your own computer, so it refuses unless .env says ENVIRONMENT=development.
DEMO_PASSWORD ?= password123456
seed: ## Load ~25 made-up accounts and 150 works for development: make seed [DEMO_PASSWORD=...]
	@if ! grep -q '^ENVIRONMENT=development' .env 2>/dev/null; then \
		echo "make seed is for your own computer: set ENVIRONMENT=development in .env first."; \
		echo "On a hosted trial, set DEMO_DATA=true instead (see docs/HOSTING.md)."; exit 1; fi
	@docker-compose run --rm -e TRIAL_NO_EMAIL=true -e DEMO_PASSWORD='$(DEMO_PASSWORD)' app nuclear demo-data
	@echo "Log in as demo01@example.org to demo25@example.org with the password $(DEMO_PASSWORD) (none has a role; make grant gives one)"

# Runs inside the app image so you don't need Go installed. It runs as your
# own user (-u) so the key file in ./secrets is yours, mode 0600, and the -v
# makes ./secrets writable for this one command (the app itself gets it
# read-only).
keys: ## Create a new login signing key in ./secrets (then see docs/RUNBOOK.md)
	@mkdir -p secrets && chmod 700 secrets
	@docker-compose run --rm --no-deps -u "$$(id -u):$$(id -g)" -v "$$(pwd)/secrets:/secrets" app nuclear keys generate /secrets/signing-key-$$(date +%Y-%m-%d).pem

keys-oidc: ## Create the separate key for ID tokens given to other apps (OIDC_SIGNING_KEY_FILE)
	@mkdir -p secrets && chmod 700 secrets
	@docker-compose run --rm --no-deps -u "$$(id -u):$$(id -g)" -v "$$(pwd)/secrets:/secrets" app nuclear keys generate /secrets/oidc-signing-key-$$(date +%Y-%m-%d).pem

keys-show: ## Show the ID of the signing key in use and of the previous keys
	@docker-compose run --rm --no-deps app nuclear keys show

# Makes a key and points .env at it, if there is no key yet. Used by dev-setup.
keys-setup:
	@if ls secrets/*.pem >/dev/null 2>&1; then echo "Signing key already exists in secrets/"; else $(MAKE) keys; fi
	@key=$$(ls -t secrets/*.pem | head -1); \
	if ! grep -q '^SIGNING_KEY_FILE=.' .env 2>/dev/null; then \
		touch .env; grep -v '^SIGNING_KEY_FILE=$$' .env > .env.tmp; \
		echo "SIGNING_KEY_FILE=/secrets/$$(basename $$key)" >> .env.tmp; mv .env.tmp .env; \
		echo "Set SIGNING_KEY_FILE in .env. Save a copy of $$key somewhere outside this server."; \
	fi; \
	if ! grep -q '^APP_UID=' .env; then echo "APP_UID=$$(id -u)" >> .env; fi

grant: ## Give an account a role: make grant ACCOUNT=you@example.org ROLE=admin|moderator|tag_wrangler
	@if [ -z "$(ACCOUNT)" ] || [ -z "$(ROLE)" ]; then echo "usage: make grant ACCOUNT=you@example.org ROLE=admin"; exit 1; fi
	@docker-compose run --rm app nuclear grant $(ACCOUNT) $(ROLE)

jobs: ## Show background jobs (emails): waiting, running, failed, and why
	@docker-compose run --rm app nuclear jobs

jobs-retry: ## Send failed background jobs again: make jobs-retry (all) or make jobs-retry ID=42
	@docker-compose run --rm app nuclear jobs retry $(ID)

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
	@curl -s http://localhost:8088/health | jq '.' 2>/dev/null || echo "Backend not ready"

logs: ## Show logs for all services
	@docker-compose logs -f

logs-db: ## Show database logs only
	@docker-compose logs -f postgres

# Development Setup
dev-setup: ## Complete development environment setup
	@echo "🛠️  Setting up development environment..."
	@make keys-setup
	@touch .env
	@grep -q '^ENVIRONMENT=' .env || echo 'ENVIRONMENT=development' >> .env
	@if ! grep -q '^GRAFANA_ADMIN_PASSWORD=.' .env; then \
		grep -v '^GRAFANA_ADMIN_PASSWORD=$$' .env > .env.tmp; \
		echo "GRAFANA_ADMIN_PASSWORD=$$(openssl rand -base64 24)" >> .env.tmp; mv .env.tmp .env; \
		echo "Generated a Grafana password in .env (GRAFANA_ADMIN_PASSWORD)."; fi
	@make db-reset
	@make services
	@cd frontend && npm install
	@echo "✅ Development environment ready!"
	@echo "🌐 Website (dev): run make frontend-dev, then http://localhost:3001"
	@echo "🔌 API: http://localhost:8088"
	@echo "📊 Grafana: http://localhost:3002"

frontend-dev: ## Start the website on http://localhost:3001 (needs the app running)
	@echo "🌐 Starting frontend development server..."
	@cd frontend && npm run dev

# Testing
test: ## Run all tests
	@echo "🧪 Running tests..."
	@cd backend && go test ./... -v

test-api: ## Test API endpoints with sample data
	@echo "🔌 Testing API endpoints..."
	@curl -s "http://localhost:8088/health" | jq '.'
	@echo "\n📚 Sample works:"
	@curl -s "http://localhost:8088/api/v1/works/?limit=3" | jq '.works | length'
	@echo "\n🔍 Search test:"
	@curl -s "http://localhost:8088/api/v1/search?q=Harry&limit=5" | jq '.' || echo "Search service not ready"

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
backup-db: ## Back up the database to backups/ (same as make ops-backup)
	@scripts/ops/backup.sh

restore-db: ## Replace the database with a backup: make restore-db FILE=nuclear-DATE.sql.gz
	@$(MAKE) --no-print-directory ops-restore FILE="$(FILE)"

# Operations. The same scripts run from GitHub (Actions tab) and here; see
# docs/OPS_PIPELINES.md. Each prints a short summary of what it found and did.
ops-diagnose: ## Check that everything is healthy (changes nothing)
	@scripts/ops/diagnose.sh

ops-emails: ## Why emails aren't arriving; add RETRY=1 to send failed ones again
	@scripts/ops/emails.sh $(if $(RETRY),--retry,)

ops-backup: ## Back up the database to backups/ (keeps the newest 14)
	@scripts/ops/backup.sh

ops-upgrade: ## Move to a new version: make ops-upgrade REF=v1.4.0
	@if [ -z "$(REF)" ]; then echo "usage: make ops-upgrade REF=<tag, branch or commit>"; exit 1; fi
	@scripts/ops/upgrade.sh "$(REF)"

ops-restore: ## Replace the database with a backup: make ops-restore FILE=nuclear-DATE.sql.gz
	@if [ -z "$(FILE)" ]; then echo "usage: make ops-restore FILE=nuclear-YYYYmmdd-HHMMSS.sql.gz (see backups/)"; exit 1; fi
	@echo "This replaces the WHOLE database with $(FILE). Everything saved since is lost."
	@printf 'Type "replace the database" to go on: '; read answer; \
	if [ "$$answer" != "replace the database" ]; then echo "Cancelled."; exit 1; fi
	@scripts/ops/restore.sh "$(FILE)" --i-understand-this-replaces-the-database

ops-logout-everyone: ## Log every account out (everyone logs in again)
	@scripts/ops/logout-everyone.sh

ops-logout-user: ## Log one account out: make ops-logout-user ACCOUNT=person@example.org
	@if [ -z "$(ACCOUNT)" ]; then echo "usage: make ops-logout-user ACCOUNT=person@example.org"; exit 1; fi
	@scripts/ops/logout-user.sh "$(ACCOUNT)"

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
	@docker exec nuclear-ao3-postgres psql -U ao3_user -d ao3_nuclear -c "TRUNCATE TABLE work_tags, work_statistics, works, tags, users RESTART IDENTITY CASCADE;"
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
	@echo "curl http://localhost:8088/health"
	@echo ""
	@echo "List Works:"
	@echo "curl 'http://localhost:8088/api/v1/works/?limit=10'"
	@echo ""
	@echo "Search Works:"
	@echo "curl 'http://localhost:8088/api/v1/search?q=Agatha&limit=5'"
	@echo ""
	@echo "Filter by Fandom:"
	@echo "curl 'http://localhost:8088/api/v1/works/?fandom=Harry%20Potter&limit=10'"

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

# Off-server backups (docs/BACKUPS.md). Run these on the server.
.PHONY: offsite-init offsite-list offsite-fetch
offsite-init: ## One-time: create the encrypted off-server backup repository
	@scripts/ops/offsite-init.sh

offsite-list: ## List the backups stored off-server
	@scripts/ops/offsite-list.sh

offsite-fetch: ## Bring a backup back from off-server: make offsite-fetch ID=<snapshot-id|latest>
	@if [ -z "$(ID)" ]; then echo "usage: make offsite-fetch ID=<snapshot-id or latest> (see make offsite-list)"; exit 1; fi
	@scripts/ops/offsite-fetch.sh "$(ID)"
