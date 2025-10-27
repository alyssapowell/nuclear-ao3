#!/bin/bash

# Nuclear AO3 - Work Management E2E Test Runner
# This script runs comprehensive Playwright tests for work creation, editing, and management workflows

set -e

echo "🎭 Nuclear AO3 Work Management Test Suite"
echo "========================================"

# Navigate to frontend directory
cd frontend

# Check if dependencies are installed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

# Check if Playwright browsers are installed
if [ ! -d "node_modules/@playwright/test" ]; then
    echo "🎭 Installing Playwright..."
    npx playwright install
fi

# Start the development server in background if not running
if ! curl -f http://localhost:3000 >/dev/null 2>&1; then
    echo "🚀 Starting development server..."
    npm run dev &
    DEV_PID=$!
    
    # Wait for server to be ready
    echo "⏳ Waiting for development server to start..."
    for i in {1..30}; do
        if curl -f http://localhost:3000 >/dev/null 2>&1; then
            echo "✅ Development server is ready!"
            break
        fi
        sleep 2
        if [ $i -eq 30 ]; then
            echo "❌ Development server failed to start"
            exit 1
        fi
    done
else
    echo "✅ Development server is already running"
    DEV_PID=""
fi

# Function to cleanup background processes
cleanup() {
    if [ -n "$DEV_PID" ]; then
        echo "🧹 Stopping development server..."
        kill $DEV_PID 2>/dev/null || true
    fi
}

# Set trap to cleanup on script exit
trap cleanup EXIT

echo ""
echo "🧪 Running Work Management & Multi-User Tests..."
echo "================================================="

# Run the work management tests with detailed reporting
npx playwright test \
    --config=playwright.work-management.config.ts \
    --reporter=html \
    --reporter=json \
    --headed \
    --workers=1

# Check test results
TEST_RESULT=$?

echo ""
if [ $TEST_RESULT -eq 0 ]; then
    echo "✅ All work management tests passed!"
    echo ""
    echo "📊 Test Coverage Areas:"
    echo "  ✓ Single User: Work Creation (basic, advanced, validation)"
    echo "  ✓ Single User: Work Editing (metadata, privacy, tags)"
    echo "  ✓ Single User: Multi-Chapter Management (create, edit, delete, navigation)"
    echo "  ✓ Single User: Draft to Published Workflow"
    echo "  ✓ Single User: Dashboard Integration"
    echo "  ✓ Multi-User: User Authentication & Sessions"
    echo "  ✓ Multi-User: Work Access Control & Privacy"
    echo "  ✓ Multi-User: Author vs Reader Permissions"
    echo "  ✓ Multi-User: Cross-User Interactions (bookmarks, comments, kudos)"
    echo "  ✓ Multi-User: Anonymous User Limitations"
    echo "  ✓ Multi-User: Comment Policy Enforcement"
    echo ""
    echo "📋 Test Report: file://$(pwd)/playwright-report-work-management/index.html"
else
    echo "❌ Some work management tests failed!"
    echo ""
    echo "📋 Test Report: file://$(pwd)/playwright-report-work-management/index.html"
    echo "📄 Test Results: $(pwd)/test-results-work-management.json"
fi

# Display summary statistics if jq is available
if command -v jq >/dev/null 2>&1 && [ -f "test-results-work-management.json" ]; then
    echo ""
    echo "📈 Test Statistics:"
    echo "=================="
    
    TOTAL_TESTS=$(jq '.suites[].specs | length' test-results-work-management.json | awk '{sum += $1} END {print sum}')
    PASSED_TESTS=$(jq '[.suites[].specs[].tests[] | select(.results[0].status == "passed")] | length' test-results-work-management.json)
    FAILED_TESTS=$(jq '[.suites[].specs[].tests[] | select(.results[0].status == "failed")] | length' test-results-work-management.json)
    
    echo "Total Tests: $TOTAL_TESTS"
    echo "Passed: $PASSED_TESTS"
    echo "Failed: $FAILED_TESTS"
    
    if [ $FAILED_TESTS -gt 0 ]; then
        echo ""
        echo "❌ Failed Tests:"
        jq -r '.suites[].specs[].tests[] | select(.results[0].status == "failed") | "  • " + .title' test-results-work-management.json
    fi
fi

echo ""
echo "🎯 Next Steps:"
echo "=============="
echo "1. Review test results in the HTML report"
echo "2. Fix any failing tests"
echo "3. Run tests again to verify fixes"
echo "4. Consider adding more edge cases"

exit $TEST_RESULT