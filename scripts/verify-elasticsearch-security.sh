#!/bin/bash

# Elasticsearch Security Verification Script
# Verifies that Elasticsearch is properly secured and accessible only internally

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Elasticsearch Security Verification"
echo "=========================================="
echo ""

TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local expected_result="$2"
    shift 2
    local command="$@"

    echo -e "${BLUE}Testing: ${test_name}${NC}"

    if eval "$command" > /dev/null 2>&1; then
        if [ "$expected_result" = "success" ]; then
            echo -e "${GREEN}✓ PASS${NC}"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}✗ FAIL - Expected to fail but succeeded${NC}"
            ((TESTS_FAILED++))
        fi
    else
        if [ "$expected_result" = "fail" ]; then
            echo -e "${GREEN}✓ PASS - Correctly blocked${NC}"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}✗ FAIL - Expected to succeed but failed${NC}"
            ((TESTS_FAILED++))
        fi
    fi
    echo ""
}

# Test 1: Check UFW status
echo -e "${YELLOW}Step 1: Checking UFW Rules${NC}"
echo "----------------------------------------"
if command -v ufw &> /dev/null; then
    if [ "$EUID" -eq 0 ]; then
        ufw status | grep -E "9200|9300" || echo "No UFW rules found for ports 9200/9300"
    else
        echo "Run with sudo to check UFW status"
        sudo ufw status | grep -E "9200|9300" || echo "No UFW rules found for ports 9200/9300"
    fi
else
    echo -e "${YELLOW}UFW not found - may not be needed if using Docker binding${NC}"
fi
echo ""

# Test 2: Local Elasticsearch access (should work)
echo -e "${YELLOW}Step 2: Testing Local Elasticsearch Access${NC}"
echo "----------------------------------------"
run_test "Localhost access via curl" "success" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:9200 | grep -q 200"

# Test 3: Check Elasticsearch cluster health
echo -e "${YELLOW}Step 3: Checking Elasticsearch Health${NC}"
echo "----------------------------------------"
if curl -s http://localhost:9200/_cluster/health 2>/dev/null; then
    CLUSTER_STATUS=$(curl -s http://localhost:9200/_cluster/health 2>/dev/null | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    if [ "$CLUSTER_STATUS" = "green" ] || [ "$CLUSTER_STATUS" = "yellow" ]; then
        echo -e "${GREEN}✓ Cluster status: $CLUSTER_STATUS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ Cluster status: $CLUSTER_STATUS${NC}"
        ((TESTS_FAILED++))
    fi
else
    echo -e "${RED}✗ Cannot reach Elasticsearch${NC}"
    ((TESTS_FAILED++))
fi
echo ""

# Test 4: Check Docker containers
echo -e "${YELLOW}Step 4: Checking Docker Containers${NC}"
echo "----------------------------------------"
run_test "Elasticsearch container running" "success" \
    "docker ps | grep -q nuclear-ao3-elasticsearch"

run_test "Search service container running" "success" \
    "docker ps | grep -q nuclear-ao3-search-service"

run_test "Tag service container running" "success" \
    "docker ps | grep -q nuclear-ao3-tags"

# Test 5: Check Docker service logs for ES connection errors
echo -e "${YELLOW}Step 5: Checking Service Logs for Errors${NC}"
echo "----------------------------------------"
echo "Recent logs from search-service:"
docker logs nuclear-ao3-search-service --tail 20 2>&1 | grep -i "elasticsearch\|error\|fatal" || echo -e "${GREEN}No Elasticsearch errors found${NC}"
echo ""

echo "Recent logs from tag-service:"
docker logs nuclear-ao3-tags --tail 20 2>&1 | grep -i "elasticsearch\|error\|fatal" || echo -e "${GREEN}No Elasticsearch errors found${NC}"
echo ""

# Test 6: Check Docker port bindings
echo -e "${YELLOW}Step 6: Checking Port Bindings${NC}"
echo "----------------------------------------"
PORT_BINDING=$(docker ps --format "{{.Names}}\t{{.Ports}}" | grep elasticsearch | grep -o "127.0.0.1:9200" || echo "")
if [ -n "$PORT_BINDING" ]; then
    echo -e "${GREEN}✓ Elasticsearch bound to localhost only (127.0.0.1:9200)${NC}"
    ((TESTS_PASSED++))
else
    BINDING=$(docker ps --format "{{.Ports}}" | grep elasticsearch | grep 9200)
    if echo "$BINDING" | grep -q "0.0.0.0:9200"; then
        echo -e "${YELLOW}⚠ Warning: Elasticsearch bound to 0.0.0.0:9200 (publicly accessible)${NC}"
        echo "  Recommend changing to 127.0.0.1:9200:9200 in docker-compose.yml"
    else
        echo "Current binding: $BINDING"
    fi
    ((TESTS_FAILED++))
fi
echo ""

# Test 7: Check indices
echo -e "${YELLOW}Step 7: Checking Elasticsearch Indices${NC}"
echo "----------------------------------------"
if curl -s http://localhost:9200/_cat/indices?v 2>/dev/null; then
    echo -e "${GREEN}✓ Successfully retrieved indices${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Cannot retrieve indices${NC}"
    ((TESTS_FAILED++))
fi
echo ""

# Summary
echo "=========================================="
echo "Verification Summary"
echo "=========================================="
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed! Elasticsearch is properly secured.${NC}"
    echo ""
    echo "Security measures in place:"
    echo "  - UFW blocking external access (if configured)"
    echo "  - Docker binding to localhost only"
    echo "  - Services can communicate via internal Docker network"
    exit 0
else
    echo -e "${YELLOW}⚠ Some checks failed. Review the output above.${NC}"
    echo ""
    echo "Common issues:"
    echo "  - Elasticsearch container not running: docker-compose up -d elasticsearch"
    echo "  - UFW not configured: Run scripts/secure-elasticsearch-ufw.sh"
    echo "  - Services not started: docker-compose up -d"
    exit 1
fi
