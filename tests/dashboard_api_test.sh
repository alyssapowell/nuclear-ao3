#!/bin/bash

# Dashboard API Test Suite - Shell Version
# Testing authentication and dashboard functionality

echo "🚀 Starting Dashboard API Test Suite..."
echo "=========================================="

BASE_URL="http://localhost:8080"
AUTH_URL="http://localhost:8081"

# Test credentials
EMAIL="testuser30d_v2@example.com"
PASSWORD="TestPassword123!"
USER_ID="1545cf30-6a5c-4e9f-9245-6b5803ec8552"
TEST_WORK_ID="99baa6c5-35e2-4ad8-9fe8-cbc8d641c211"

echo "📡 Testing service health checks..."

# Check API Gateway health
echo -n "API Gateway (8080): "
if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
    echo "✅ HEALTHY"
else
    echo "❌ UNHEALTHY"
fi

# Check Auth Service health  
echo -n "Auth Service (8081): "
if curl -s -f "$AUTH_URL/health" > /dev/null 2>&1; then
    echo "✅ HEALTHY"
else
    echo "❌ UNHEALTHY"
fi

# Check Work Service health directly (gateway doesn't proxy health endpoints)
echo -n "Work Service (8082): "
if curl -s -f "http://localhost:8082/health" > /dev/null 2>&1; then
    echo "✅ HEALTHY"
else
    echo "❌ UNHEALTHY"
fi

echo ""
echo "🔐 Testing authentication flow..."

# Login and get token
echo "Logging in with: $EMAIL"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "Login response: $LOGIN_RESPONSE"

# Extract token
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ FAILED to get authentication token"
    echo "Login response was: $LOGIN_RESPONSE"
    exit 1
else
    echo "✅ Successfully obtained auth token (${TOKEN:0:20}...)"
fi

echo ""
echo "📊 Testing Dashboard API..."

# Test GetMyWorks endpoint
echo "Testing GetMyWorks endpoint..."
MY_WORKS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/my/works" \
    -H "Authorization: Bearer $TOKEN")

echo "GetMyWorks response: $MY_WORKS_RESPONSE"

# Check if response contains works array
if echo "$MY_WORKS_RESPONSE" | grep -q '"works"'; then
    echo "✅ GetMyWorks endpoint working"
    
    # Check if our test work is in the response
    if echo "$MY_WORKS_RESPONSE" | grep -q "$TEST_WORK_ID"; then
        echo "✅ Test work found in user's works"
    else
        echo "⚠️  Test work not found in user's works"
    fi
else
    echo "❌ GetMyWorks endpoint failed"
    echo "Response: $MY_WORKS_RESPONSE"
fi

echo ""
echo "📖 Testing individual work access..."

# Test GetWork endpoint with authentication
echo "Testing GetWork with authentication..."
GET_WORK_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/works/$TEST_WORK_ID" \
    -H "Authorization: Bearer $TOKEN")

echo "GetWork response: $GET_WORK_RESPONSE"

if echo "$GET_WORK_RESPONSE" | grep -q '"id"'; then
    echo "✅ GetWork with auth working"
else
    echo "❌ GetWork with auth failed"
fi

# Test GetWork without authentication
echo "Testing GetWork without authentication..."
GET_WORK_NO_AUTH_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/works/$TEST_WORK_ID")

echo "GetWork (no auth) response: $GET_WORK_NO_AUTH_RESPONSE"

if echo "$GET_WORK_NO_AUTH_RESPONSE" | grep -q '"id"'; then
    echo "✅ GetWork without auth working"
else
    echo "❌ GetWork without auth failed"
fi

echo ""
echo "💬 Testing Comments API..."

# Test GetComments endpoint
echo "Testing GetComments..."
GET_COMMENTS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/works/$TEST_WORK_ID/comments" \
    -H "Authorization: Bearer $TOKEN")

echo "GetComments response: $GET_COMMENTS_RESPONSE"

if echo "$GET_COMMENTS_RESPONSE" | grep -q '"comments"'; then
    echo "✅ GetComments working"
else
    echo "❌ GetComments failed"
    if echo "$GET_COMMENTS_RESPONSE" | grep -q "403"; then
        echo "   Issue: 403 Forbidden - likely routing conflict"
    fi
fi

echo ""
echo "🔖 Testing Bookmark Status API..."

# Test bookmark status endpoint
echo "Testing bookmark status..."
BOOKMARK_STATUS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/works/$TEST_WORK_ID/bookmark-status" \
    -H "Authorization: Bearer $TOKEN")

echo "Bookmark status response: $BOOKMARK_STATUS_RESPONSE"

if echo "$BOOKMARK_STATUS_RESPONSE" | grep -q '"is_bookmarked"'; then
    echo "✅ Bookmark status working"
else
    echo "❌ Bookmark status failed"
    if echo "$BOOKMARK_STATUS_RESPONSE" | grep -q "500"; then
        echo "   Issue: 500 Internal Server Error - likely user ID type conversion"
    fi
fi

echo ""
echo "🔧 Testing error handling..."

# Test with invalid token
echo "Testing with invalid token..."
INVALID_TOKEN_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/my/works" \
    -H "Authorization: Bearer invalid_token_123")

echo "Invalid token response: $INVALID_TOKEN_RESPONSE"

if echo "$INVALID_TOKEN_RESPONSE" | grep -q "Invalid token"; then
    echo "✅ Invalid token properly rejected"
else
    echo "❌ Invalid token not properly handled"
fi

# Test with nonexistent work
echo "Testing with nonexistent work..."
NONEXISTENT_WORK_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/works/00000000-0000-0000-0000-000000000000" \
    -H "Authorization: Bearer $TOKEN")

echo "Nonexistent work response: $NONEXISTENT_WORK_RESPONSE"

if echo "$NONEXISTENT_WORK_RESPONSE" | grep -q "404\|not found\|Cannot view this work"; then
    echo "✅ Nonexistent work properly handled"
else
    echo "❌ Nonexistent work not properly handled"
fi

echo ""
echo "🏁 Dashboard API Test Suite Complete!"
echo "====================================="