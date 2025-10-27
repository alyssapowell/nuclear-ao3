#!/bin/bash

# Setup script for E2E testing
# Ensures all services are running and test data is seeded

set -e

echo "🚀 Setting up Nuclear AO3 for E2E testing..."

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Start services if not already running
echo "📦 Starting Nuclear AO3 services..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if PostgreSQL is ready
echo "🗄️ Checking PostgreSQL connection..."
timeout=30
counter=0
while ! docker-compose exec -T db pg_isready -h localhost -p 5432 -U ao3_user; do
    counter=$((counter + 1))
    if [ $counter -gt $timeout ]; then
        echo "❌ PostgreSQL failed to start within ${timeout} seconds"
        exit 1
    fi
    echo "Waiting for PostgreSQL... ($counter/$timeout)"
    sleep 1
done

# Check if Redis is ready
echo "🔴 Checking Redis connection..."
if ! docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
    echo "❌ Redis is not responding"
    exit 1
fi

# Check if backend services are ready
echo "🔧 Checking backend services..."
services=("auth-service:8081" "work-service:8082" "tag-service:8083" "search-service:8084" "api-gateway:8080")
for service in "${services[@]}"; do
    service_name=${service%:*}
    port=${service#*:}
    echo "Checking $service_name on port $port..."
    
    timeout=30
    counter=0
    while ! curl -f http://localhost:$port/health >/dev/null 2>&1; do
        counter=$((counter + 1))
        if [ $counter -gt $timeout ]; then
            echo "❌ $service_name failed to respond within ${timeout} seconds"
            docker-compose logs $service_name
            exit 1
        fi
        echo "Waiting for $service_name... ($counter/$timeout)"
        sleep 1
    done
    echo "✅ $service_name is ready"
done

# Seed test data
echo "🌱 Seeding E2E test data..."
cd tools
node e2e-test-seeder.js
cd ..

# Check if frontend is ready (if running)
if curl -f http://localhost:3000 >/dev/null 2>&1; then
    echo "✅ Frontend is ready on http://localhost:3000"
else
    echo "⚠️ Frontend not detected. You may need to start it manually with:"
    echo "   cd frontend && npm run dev"
fi

echo ""
echo "🎉 E2E test environment is ready!"
echo ""
echo "Test credentials:"
echo "  Email: testuser30d_v2@example.com"
echo "  Password: TestPassword123!"
echo ""
echo "Services:"
echo "  Frontend: http://localhost:3000"
echo "  API Gateway: http://localhost:8080"
echo "  Auth Service: http://localhost:8081"
echo "  Work Service: http://localhost:8082"
echo "  Tag Service: http://localhost:8083"
echo "  Search Service: http://localhost:8084"
echo ""
echo "Run E2E tests with:"
echo "  cd frontend && npx playwright test e2e/integration.spec.ts"