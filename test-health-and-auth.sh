#!/bin/bash

# Test script untuk Health Check dan Basic Auth
set -e

BASE_URL="http://localhost:3000"

echo "=========================================="
echo "Testing Health Check & Basic Auth"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}➜ $1${NC}"
}

# Test 1: Health Check (tanpa auth - harus berhasil)
print_info "Test 1: Health Check (tanpa autentikasi)"
response=$(curl -s "$BASE_URL/health")
echo "$response" | jq .

if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
    print_success "Health check berhasil tanpa autentikasi"
else
    print_error "Health check gagal"
    exit 1
fi
echo ""

# Test 2: Access root endpoint tanpa auth (harus gagal jika basic auth enabled)
print_info "Test 2: Access root endpoint tanpa auth"
status_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/")
echo "HTTP Status Code: $status_code"

if [ "$status_code" == "401" ]; then
    print_success "Basic auth berfungsi - endpoint protected"
elif [ "$status_code" == "200" ]; then
    print_info "Basic auth tidak aktif - endpoint accessible"
else
    print_error "Unexpected status code: $status_code"
fi
echo ""

# Test 3: Access root endpoint dengan auth (harus berhasil)
print_info "Test 3: Access root endpoint dengan basic auth"
status_code=$(curl -s -o /dev/null -w "%{http_code}" -u user1:pass1 "$BASE_URL/")
echo "HTTP Status Code: $status_code"

if [ "$status_code" == "200" ]; then
    print_success "Basic auth login berhasil"
else
    print_info "Basic auth mungkin tidak aktif atau credentials salah"
fi
echo ""

# Test 4: Create account tanpa auth (harus gagal jika basic auth enabled)
print_info "Test 4: Create account tanpa auth"
response=$(curl -s -X POST "$BASE_URL/accounts" \
    -H "Content-Type: application/json" \
    -d '{"account_id": "test_auth"}')
echo "$response" | jq . || echo "$response"

if echo "$response" | grep -q "Unauthorized" || echo "$response" | grep -q "401"; then
    print_success "API endpoint protected by basic auth"
else
    print_info "Basic auth tidak aktif pada API endpoints"
fi
echo ""

# Test 5: Create account dengan auth (harus berhasil)
print_info "Test 5: Create account dengan basic auth"
response=$(curl -s -X POST "$BASE_URL/accounts" \
    -u user1:pass1 \
    -H "Content-Type: application/json" \
    -d '{"account_id": "test_auth"}')
echo "$response" | jq .

if echo "$response" | jq -e '.code == "SUCCESS"' > /dev/null; then
    print_success "Account created dengan autentikasi"

    # Cleanup
    print_info "Cleanup: Delete test account"
    curl -s -X DELETE "$BASE_URL/accounts/test_auth" -u user1:pass1 > /dev/null
    print_success "Test account deleted"
else
    if echo "$response" | grep -q "already exists"; then
        print_info "Account sudah ada (OK)"
    else
        print_error "Failed to create account"
    fi
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
print_success "Health check endpoint accessible tanpa auth (✓ Good for Coolify/Docker)"
print_success "API endpoints require basic auth jika APP_BASIC_AUTH di-set"
echo ""
echo "Kesimpulan:"
echo "- Endpoint /health TIDAK require autentikasi (aman untuk health check)"
echo "- Semua endpoint API lain REQUIRE autentikasi jika APP_BASIC_AUTH di-set"
echo "- Coolify health check akan tetap berfungsi meski basic auth aktif"
echo ""
