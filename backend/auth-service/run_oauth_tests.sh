#!/bin/bash

# OAuth2/OIDC Test Runner for Nuclear AO3 Auth Service
# This script runs comprehensive OAuth2/OIDC authentication tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DB_NAME="ao3_nuclear_test"
TEST_TIMEOUT="10m"

echo -e "${BLUE}🔐 Nuclear AO3 OAuth2/OIDC Test Suite${NC}"
echo "=================================================="

# Function to print status
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

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    
    # Check if PostgreSQL is running
    if ! pg_isready -d "$DB_NAME" &> /dev/null; then
        print_warning "PostgreSQL test database '$DB_NAME' not ready"
        print_status "Attempting to create test database..."
        
        # Try to create test database
        createdb "$DB_NAME" 2>/dev/null || print_warning "Database might already exist"
    fi
    
    # Check if Redis is running
    if ! redis-cli ping &> /dev/null; then
        print_error "Redis is not running. Please start Redis server."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Setup test environment
setup_test_env() {
    print_status "Setting up test environment..."
    
    # Set test environment variables
    export TEST_DATABASE_URL="postgres://ao3_user:ao3_password@localhost/$DB_NAME?sslmode=disable"
    export TEST_REDIS_URL="localhost:6379"
    export GIN_MODE="test"
    export BASE_URL="https://test.nuclear-ao3.com"
    
    # Run database migrations if needed
    if [ -f "../migrations/001_create_users_and_auth.sql" ]; then
        print_status "Running database migrations..."
        psql "$TEST_DATABASE_URL" -f "../migrations/001_create_users_and_auth.sql" 2>/dev/null || true
        psql "$TEST_DATABASE_URL" -f "../migrations/002_create_content_tables.sql" 2>/dev/null || true
        psql "$TEST_DATABASE_URL" -f "../migrations/003_oauth2_tables.sql" 2>/dev/null || true
    fi
    
    print_success "Test environment setup complete"
}

# Run specific test suite
run_test_suite() {
    local test_file=$1
    local test_name=$2
    
    print_status "Running $test_name..."
    
    if go test -v -timeout="$TEST_TIMEOUT" -run="Test.*Suite" "./$test_file" 2>&1 | tee "test_results_$(basename $test_file .go).log"; then
        print_success "$test_name completed successfully"
        return 0
    else
        print_error "$test_name failed"
        return 1
    fi
}

# Run performance tests with specific flags
run_performance_tests() {
    print_status "Running OAuth2/OIDC Performance Tests..."
    
    if go test -v -timeout="$TEST_TIMEOUT" -run="TestOAuth2PerformanceTestSuite" "./oauth_performance_test.go" -args -test.benchtime=10s 2>&1 | tee "test_results_performance.log"; then
        print_success "Performance tests completed"
        return 0
    else
        print_error "Performance tests failed"
        return 1
    fi
}

# Generate test coverage report
generate_coverage() {
    print_status "Generating test coverage report..."
    
    go test -coverprofile=oauth_coverage.out -covermode=count \
        "./oauth_test.go" "./oidc_test.go" "./oauth_performance_test.go" \
        "./auth_service_test.go" 2>/dev/null || true
    
    if [ -f "oauth_coverage.out" ]; then
        go tool cover -html=oauth_coverage.out -o oauth_coverage.html
        go tool cover -func=oauth_coverage.out | tail -1
        print_success "Coverage report generated: oauth_coverage.html"
    else
        print_warning "Could not generate coverage report"
    fi
}

# Clean up test data
cleanup() {
    print_status "Cleaning up test data..."
    
    # Clean test database
    psql "$TEST_DATABASE_URL" -c "
        DELETE FROM user_consents WHERE created_at > NOW() - INTERVAL '1 day';
        DELETE FROM oauth_refresh_tokens WHERE created_at > NOW() - INTERVAL '1 day';
        DELETE FROM oauth_access_tokens WHERE created_at > NOW() - INTERVAL '1 day';
        DELETE FROM authorization_codes WHERE created_at > NOW() - INTERVAL '1 day';
        DELETE FROM oauth_clients WHERE created_at > NOW() - INTERVAL '1 day';
        DELETE FROM users WHERE created_at > NOW() - INTERVAL '1 day';
    " 2>/dev/null || true
    
    # Clean Redis test databases
    redis-cli -n 2 FLUSHDB &>/dev/null || true
    redis-cli -n 3 FLUSHDB &>/dev/null || true
    redis-cli -n 4 FLUSHDB &>/dev/null || true
    redis-cli -n 5 FLUSHDB &>/dev/null || true
    
    print_success "Cleanup completed"
}

# Main test execution
main() {
    local exit_code=0
    local failed_tests=()
    
    # Parse command line arguments
    SKIP_PERFORMANCE=false
    COVERAGE_ONLY=false
    CLEAN_ONLY=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-performance)
                SKIP_PERFORMANCE=true
                shift
                ;;
            --coverage-only)
                COVERAGE_ONLY=true
                shift
                ;;
            --clean-only)
                CLEAN_ONLY=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --skip-performance  Skip performance tests"
                echo "  --coverage-only     Only generate coverage report"
                echo "  --clean-only        Only cleanup test data"
                echo "  --help              Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Handle special modes
    if [ "$CLEAN_ONLY" = true ]; then
        cleanup
        exit 0
    fi
    
    if [ "$COVERAGE_ONLY" = true ]; then
        setup_test_env
        generate_coverage
        exit 0
    fi
    
    # Full test suite execution
    echo "Starting comprehensive OAuth2/OIDC test execution..."
    echo "Test configuration:"
    echo "  Database: $DB_NAME"
    echo "  Timeout: $TEST_TIMEOUT"
    echo "  Skip Performance: $SKIP_PERFORMANCE"
    echo ""
    
    # Setup
    check_prerequisites
    setup_test_env
    
    # Run core authentication tests first
    print_status "Phase 1: Core Authentication Tests"
    if ! run_test_suite "auth_service_test.go" "Core Authentication Tests"; then
        failed_tests+=("Core Authentication")
        exit_code=1
    fi
    
    # Run OAuth2 specific tests
    print_status "Phase 2: OAuth2 Flow Tests"
    if ! run_test_suite "oauth_test.go" "OAuth2 Flow Tests"; then
        failed_tests+=("OAuth2 Flows")
        exit_code=1
    fi
    
    # Run OIDC specific tests
    print_status "Phase 3: OIDC Tests"
    if ! run_test_suite "oidc_test.go" "OIDC Tests"; then
        failed_tests+=("OIDC")
        exit_code=1
    fi
    
    # Run performance tests if not skipped
    if [ "$SKIP_PERFORMANCE" = false ]; then
        print_status "Phase 4: Performance Tests"
        if ! run_performance_tests; then
            failed_tests+=("Performance")
            exit_code=1
        fi
    else
        print_warning "Skipping performance tests"
    fi
    
    # Generate coverage report
    print_status "Phase 5: Coverage Analysis"
    generate_coverage
    
    # Cleanup
    cleanup
    
    # Final report
    echo ""
    echo "=================================================="
    echo -e "${BLUE}📊 Test Execution Summary${NC}"
    echo "=================================================="
    
    if [ $exit_code -eq 0 ]; then
        print_success "All tests passed! 🎉"
        echo ""
        echo "✅ Core Authentication Tests"
        echo "✅ OAuth2 Flow Tests" 
        echo "✅ OIDC Tests"
        if [ "$SKIP_PERFORMANCE" = false ]; then
            echo "✅ Performance Tests"
        fi
        echo "✅ Coverage Analysis"
    else
        print_error "Some tests failed! ❌"
        echo ""
        for test in "${failed_tests[@]}"; do
            echo "❌ $test"
        done
        echo ""
        echo "Check log files for details:"
        ls -la test_results_*.log 2>/dev/null || true
    fi
    
    echo ""
    echo "📁 Generated files:"
    echo "  - oauth_coverage.html (coverage report)"
    echo "  - test_results_*.log (detailed test logs)"
    echo ""
    
    if [ $exit_code -eq 0 ]; then
        print_success "OAuth2/OIDC authentication system is ready for production! 🚀"
    else
        print_error "Please fix failing tests before deploying to production"
    fi
    
    exit $exit_code
}

# Execute main function with all arguments
main "$@"