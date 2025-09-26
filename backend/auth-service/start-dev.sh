#!/bin/bash

# Nuclear AO3 Auth Service - Development Startup Script
# This script starts the auth service locally for development and testing

set -e

echo "🚀 Starting Nuclear AO3 Auth Service..."

# Check dependencies
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL client not found. Please install: brew install postgresql"
    exit 1
fi

if ! command -v redis-cli &> /dev/null; then
    echo "❌ Redis client not found. Please install: brew install redis"
    exit 1
fi

# Check if PostgreSQL is running
if ! pg_isready -h localhost &> /dev/null; then
    echo "❌ PostgreSQL is not running. Please start it: brew services start postgresql"
    exit 1
fi

# Check if Redis is running  
if ! redis-cli ping &> /dev/null; then
    echo "❌ Redis is not running. Please start it: brew services start redis"
    exit 1
fi

# Set environment variables
export DATABASE_URL="postgres://$(whoami)@localhost/ao3_nuclear_test?sslmode=disable"
export REDIS_URL="localhost:6379"
export BASE_URL="http://localhost:8081"
export GIN_MODE="debug"
export PORT="8081"
export JWT_SECRET="dev-secret-change-in-production"
export JWT_ISSUER="nuclear-ao3-dev"

echo "🔧 Environment configured:"
echo "   Database: $DATABASE_URL"
echo "   Redis: $REDIS_URL"
echo "   API URL: $BASE_URL"
echo "   Mode: $GIN_MODE"

# Build if needed
if [ ! -f "./auth-service" ] || [ "main.go" -nt "./auth-service" ]; then
    echo "🔨 Building auth service..."
    go build .
fi

echo "✅ Dependencies OK"
echo "🌐 Starting auth service on http://localhost:8081"
echo ""
echo "📚 API Documentation: See API_TESTING_GUIDE.md"
echo "🔍 Health Check: curl http://localhost:8081/health"
echo "🔍 Discovery: curl http://localhost:8081/.well-known/openid-configuration"
echo ""
echo "Press Ctrl+C to stop the service"
echo "----------------------------------------"

# Start the service
./auth-service