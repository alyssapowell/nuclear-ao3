#!/bin/bash

# Enhanced Tag System Test Runner
# Tests the tag prominence system and anti-spam features

set -e

echo "🏷️  Nuclear AO3 Enhanced Tag System Test Suite"
echo "=============================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if services are running
print_status "Checking required services..."

# Check if Docker services are running
if ! docker-compose ps | grep -q "nuclear-ao3-tags.*Up"; then
    print_error "Tag service not running. Starting services..."
    docker-compose up -d
    sleep 10
fi

# Check if frontend is accessible
if ! curl -s http://localhost:3001 > /dev/null; then
    print_warning "Frontend not accessible on localhost:3001"
    print_status "Starting frontend..."
    cd frontend
    npm run dev &
    FRONTEND_PID=$!
    cd ..
    sleep 5
else
    print_success "Frontend accessible"
fi

# Check if backend services are healthy
if curl -s http://localhost:8083/health | grep -q "healthy"; then
    print_success "Tag service healthy"
else
    print_error "Tag service not healthy"
    exit 1
fi

if curl -s http://localhost:8080/health | grep -q "healthy"; then
    print_success "API Gateway healthy"
else
    print_error "API Gateway not healthy"
    exit 1
fi

print_status "All services ready. Starting enhanced tag system tests..."

cd frontend

# Run the comprehensive tag enhancement test suite
print_status "Running Enhanced Tag Prominence Tests..."
if npm run test:tag-enhancement; then
    print_success "Enhanced tag prominence tests passed"
else
    print_error "Enhanced tag prominence tests failed"
    EXIT_CODE=1
fi

print_status "Running Tag Spam Prevention Tests..."
if npm run test:tag-spam; then
    print_success "Tag spam prevention tests passed"
else
    print_error "Tag spam prevention tests failed"
    EXIT_CODE=1
fi

# Run specific test scenarios
print_status "Running Orgy Problem Prevention Test..."
if npx playwright test e2e/tag-spam-prevention.spec.ts -g "orgy-style tag spam"; then
    print_success "Orgy problem prevention working"
else
    print_error "Orgy problem prevention failed"
    EXIT_CODE=1
fi

print_status "Running Character Detection Test..."
if npx playwright test e2e/enhanced-tag-prominence.spec.ts -g "missing character"; then
    print_success "Character detection working"
else
    print_error "Character detection failed"
    EXIT_CODE=1
fi

print_status "Running Relationship Limit Test..."
if npx playwright test e2e/tag-spam-prevention.spec.ts -g "limiting primary relationships"; then
    print_success "Relationship limit enforcement working"
else
    print_error "Relationship limit enforcement failed"
    EXIT_CODE=1
fi

# Generate test report
print_status "Generating test report..."
REPORT_DIR="test-results/tag-enhancement-report"
if [ -d "$REPORT_DIR" ]; then
    print_success "Test report generated: $REPORT_DIR/index.html"
    
    # Count test results
    if [ -f "test-results/tag-enhancement-results.json" ]; then
        PASSED=$(grep -o '"status":"passed"' test-results/tag-enhancement-results.json | wc -l)
        FAILED=$(grep -o '"status":"failed"' test-results/tag-enhancement-results.json | wc -l)
        
        echo ""
        echo "📊 Test Results Summary:"
        echo "  ✅ Passed: $PASSED"
        echo "  ❌ Failed: $FAILED"
        echo ""
    fi
else
    print_warning "No test report generated"
fi

# Test specific scenarios from the problem description
print_status "Running Real-World Problem Scenarios..."

echo ""
echo "🎯 Testing scenarios from https://archiveofourown.org/works/72365631"
echo "   (Excessive relationship tagging example)"

if npx playwright test e2e/tag-spam-prevention.spec.ts -g "prevent orgy-style"; then
    print_success "✅ Successfully prevents excessive relationship tagging"
else
    print_error "❌ Failed to prevent excessive relationship tagging"
    EXIT_CODE=1
fi

print_status "Testing character auto-detection..."
if npx playwright test e2e/enhanced-tag-prominence.spec.ts -g "auto-detect missing characters"; then
    print_success "✅ Character auto-detection working"
else
    print_error "❌ Character auto-detection failed"
    EXIT_CODE=1
fi

print_status "Testing user prominence control (max 2-3 primary relationships)..."
if npx playwright test e2e/enhanced-tag-prominence.spec.ts -g "reprioritize.*3 relationship"; then
    print_success "✅ User prominence control working with limits"
else
    print_error "❌ User prominence control failed"
    EXIT_CODE=1
fi

# Cleanup
if [ ! -z "$FRONTEND_PID" ]; then
    print_status "Stopping frontend process..."
    kill $FRONTEND_PID 2>/dev/null || true
fi

cd ..

echo ""
echo "🏁 Enhanced Tag System Test Suite Complete"
echo "=========================================="

if [ "${EXIT_CODE:-0}" -eq 0 ]; then
    print_success "All enhanced tag system tests passed! 🎉"
    print_success "The system successfully:"
    echo "   ✅ Limits primary relationships to 2-3 maximum"
    echo "   ✅ Auto-detects missing characters from relationships"
    echo "   ✅ Prevents tag spam scenarios like the example work"
    echo "   ✅ Allows user reprioritization within limits"
    echo "   ✅ Provides clear guidance about tagging best practices"
    echo ""
    echo "🚀 Ready for production: Enhanced tagging system prevents the 'orgy problem'"
else
    print_error "Some enhanced tag system tests failed"
    print_error "Please review the test output above and fix failing tests"
    echo ""
    echo "Common issues to check:"
    echo "   - EnhancedTagProminenceSelector component integration"
    echo "   - Prominence limit enforcement (2-3 primary relationships max)"
    echo "   - Character auto-detection from relationship parsing"
    echo "   - Tag spam warning systems"
fi

exit ${EXIT_CODE:-0}