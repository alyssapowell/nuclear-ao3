#!/bin/bash

# Script to run subscription system tests
# This script handles the full test setup including backend services

set -e

echo "🚀 Starting Subscription System Test Suite"
echo "=========================================="

# Check if required ports are available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        echo "✅ Port $port is already in use (service running)"
        return 0
    else
        echo "❌ Port $port is not in use"
        return 1
    fi
}

# Start backend services if not running
start_backend() {
    echo "🔧 Checking backend services..."
    
    if ! check_port 8080; then
        echo "📡 Starting backend services..."
        cd backend && docker-compose up -d
        echo "⏳ Waiting for services to be ready..."
        sleep 30
    fi
    
    # Verify critical services
    echo "🔍 Verifying service health..."
    
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "✅ API Gateway is healthy"
    else
        echo "❌ API Gateway is not responding"
        exit 1
    fi
    
    if curl -s http://localhost:8082/health > /dev/null; then
        echo "✅ Work Service is healthy"
    else
        echo "❌ Work Service is not responding"
        exit 1
    fi
    
    if curl -s http://localhost:8081/health > /dev/null; then
        echo "✅ Auth Service is healthy"
    else
        echo "❌ Auth Service is not responding"
        exit 1
    fi
}

# Run the tests
run_tests() {
    echo "🧪 Running subscription tests..."
    cd frontend
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        echo "📦 Installing frontend dependencies..."
        npm install
    fi
    
    # Run the subscription tests
    case "${1:-basic}" in
        "basic")
            echo "🔬 Running basic subscription tests with mocking..."
            npm run test:subscription
            ;;
        "integration")
            echo "🔬 Running full integration tests..."
            npm run test:subscription:headed
            ;;
        "ui")
            echo "🔬 Running tests with UI mode..."
            npm run test:subscription:ui
            ;;
        *)
            echo "❓ Unknown test mode: $1"
            echo "Available modes: basic, integration, ui"
            exit 1
            ;;
    esac
}

# Main execution
main() {
    local test_mode="${1:-basic}"
    
    echo "📋 Test mode: $test_mode"
    
    # For basic tests, we only need minimal setup
    if [ "$test_mode" != "basic" ]; then
        start_backend
    fi
    
    run_tests "$test_mode"
    
    echo "✅ Subscription tests completed!"
}

# Handle script arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        echo "Usage: $0 [test_mode]"
        echo ""
        echo "Test modes:"
        echo "  basic       - Run tests with API mocking (default, fastest)"
        echo "  integration - Run full integration tests with real backend"
        echo "  ui          - Run tests with Playwright UI mode"
        echo "  help        - Show this help message"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac