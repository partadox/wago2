#!/bin/bash

# Test script untuk Multi-Account functionality
set -e

BASE_URL="http://localhost:3000"
ACCOUNT1="test_account1"
ACCOUNT2="test_account2"

echo "=========================================="
echo "Testing Multi-Account Functionality"
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

# Test 1: Health Check
print_info "Test 1: Health Check"
if curl -s -f "$BASE_URL/" > /dev/null; then
    print_success "Service is running"
else
    print_error "Service is not responding"
    exit 1
fi
echo ""

# Test 2: Create Account 1
print_info "Test 2: Create Account 1 ($ACCOUNT1)"
response=$(curl -s -X POST "$BASE_URL/accounts" \
    -H "Content-Type: application/json" \
    -d "{\"account_id\": \"$ACCOUNT1\"}")
echo "$response" | jq .

if echo "$response" | jq -e '.code == "SUCCESS"' > /dev/null; then
    print_success "Account 1 created"
else
    print_error "Failed to create Account 1"
fi
echo ""

# Test 3: Create Account 2
print_info "Test 3: Create Account 2 ($ACCOUNT2)"
response=$(curl -s -X POST "$BASE_URL/accounts" \
    -H "Content-Type: application/json" \
    -d "{\"account_id\": \"$ACCOUNT2\"}")
echo "$response" | jq .

if echo "$response" | jq -e '.code == "SUCCESS"' > /dev/null; then
    print_success "Account 2 created"
else
    print_error "Failed to create Account 2"
fi
echo ""

# Test 4: List All Accounts
print_info "Test 4: List All Accounts"
response=$(curl -s "$BASE_URL/accounts")
echo "$response" | jq .

count=$(echo "$response" | jq '.results | length')
if [ "$count" -ge 2 ]; then
    print_success "Found $count accounts"
else
    print_error "Expected at least 2 accounts, found $count"
fi
echo ""

# Test 5: Get Account Details
print_info "Test 5: Get Account 1 Details"
response=$(curl -s "$BASE_URL/accounts/$ACCOUNT1")
echo "$response" | jq .

if echo "$response" | jq -e '.code == "SUCCESS"' > /dev/null; then
    print_success "Account details retrieved"
    status=$(echo "$response" | jq -r '.results.status')
    print_info "Account status: $status"
else
    print_error "Failed to get account details"
fi
echo ""

# Test 6: Try to send message (will fail because not logged in)
print_info "Test 6: Try Send Message (should fail - not logged in)"
response=$(curl -s -X POST "$BASE_URL/send/message" \
    -H "Content-Type: application/json" \
    -d "{
        \"account_id\": \"$ACCOUNT1\",
        \"phone\": \"6281234567890\",
        \"message\": \"Test message\"
    }")
echo "$response" | jq .

if echo "$response" | jq -e '.code == "ERROR"' > /dev/null; then
    print_success "Correctly returned error for not logged in account"
else
    print_error "Should have returned error"
fi
echo ""

# Test 7: Login with QR (just generate, not actually scan)
print_info "Test 7: Generate Login QR for Account 1"
response=$(curl -s -X POST "$BASE_URL/accounts/$ACCOUNT1/login")
echo "$response" | jq .

if echo "$response" | jq -e '.results.image_path' > /dev/null; then
    qr_path=$(echo "$response" | jq -r '.results.image_path')
    print_success "QR code generated: $qr_path"

    if [ -f "$qr_path" ]; then
        print_success "QR code file exists"
    else
        print_error "QR code file not found"
    fi
else
    # Could be already logged in
    if echo "$response" | jq -e '.results.code == "ALREADY_LOGGED_IN"' > /dev/null; then
        print_success "Account already logged in"
    else
        print_error "Failed to generate QR code"
    fi
fi
echo ""

# Test 8: Delete Account 2
print_info "Test 8: Delete Account 2"
response=$(curl -s -X DELETE "$BASE_URL/accounts/$ACCOUNT2")
echo "$response" | jq .

if echo "$response" | jq -e '.code == "SUCCESS"' > /dev/null; then
    print_success "Account 2 deleted"
else
    print_error "Failed to delete Account 2"
fi
echo ""

# Test 9: Verify Account 2 is deleted
print_info "Test 9: Verify Account 2 is deleted"
response=$(curl -s "$BASE_URL/accounts/$ACCOUNT2")
echo "$response" | jq .

if echo "$response" | jq -e '.code == "ERROR"' > /dev/null; then
    print_success "Account 2 correctly not found"
else
    print_error "Account 2 should not exist"
fi
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
print_success "Health check passed"
print_success "Account creation tested"
print_success "Account listing tested"
print_success "Account details retrieval tested"
print_success "Send message validation tested"
print_success "QR generation tested"
print_success "Account deletion tested"
echo ""
echo "Note: To fully test, you need to:"
echo "1. Scan the QR code with WhatsApp"
echo "2. Then test sending actual messages"
echo ""
echo "Cleanup command:"
echo "  curl -X DELETE $BASE_URL/accounts/$ACCOUNT1"
echo ""
