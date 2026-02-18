#!/bin/bash

# ==========================================
# CodeDrop Robust Integration Test Suite
# ==========================================

CLI="./codedrop"
SERVER_URL="http://localhost:8080"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Initializing CodeDrop Test Suite...${NC}"

# 1. Check if CLI binary exists
if [ ! -f "$CLI" ]; then
    echo -e "${RED}Error: 'codedrop' binary not found.${NC}"
    echo "Run: go build -o codedrop cmd/cli/main.go"
    exit 1
fi

# 2. Check if Server is running
echo "pinging server..."
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    echo -e "${RED}Error: Cannot connect to server at $SERVER_URL${NC}"
    echo "Make sure you are running 'go run cmd/server/main.go' in another terminal!"
    exit 1
fi

# 3. Setup Test Artifacts
echo "Creating test files..."
echo "This is a small test payload." > test_small.txt
# Create 10MB file for CAS test
dd if=/dev/urandom of=test_large.bin bs=1M count=10 2>/dev/null

# ==========================================
# TEST 1: Standard Push & Pull
# ==========================================
echo -e "\n${YELLOW}TEST 1: Standard E2E Push, Pull, and Integrity${NC}"

# Capture both stdout and stderr (2>&1) so we see errors
echo "   Pushing file..."
PUSH_OUTPUT=$($CLI push test_small.txt --expire 10m --max-views 2 2>&1)
PUSH_EXIT_CODE=$?

# Validate Push
if [ $PUSH_EXIT_CODE -ne 0 ]; then
    echo -e "${RED}Push Failed! Output:${NC}"
    echo "$PUSH_OUTPUT"
    exit 1
fi

# Extract URL
URL=$(echo "$PUSH_OUTPUT" | grep "Secure URL" | awk '{print $NF}')
if [ -z "$URL" ]; then
    echo -e "${RED}Failed to extract URL from output.${NC}"
    echo "$PUSH_OUTPUT"
    exit 1
fi
echo "   Secure URL: $URL"

# Pull
echo "   Pulling file..."
$CLI pull "$URL" > /dev/null
PULL_EXIT_CODE=$?

if [ $PULL_EXIT_CODE -ne 0 ]; then
    echo -e "${RED}Pull Failed!${NC}"
    exit 1
fi

# Verify Integrity
if cmp -s test_small.txt downloaded_test_small.txt; then
    echo -e "${GREEN}   PASS: Integrity Verified.${NC}"
else
    echo -e "${RED}   FAIL: Decrypted file does not match original!${NC}"
    exit 1
fi
rm downloaded_test_small.txt

# ==========================================
# TEST 2: Atomic Limits (Redis)
# ==========================================
echo -e "\n${YELLOW}TEST 2: Atomic Download Limits (--max-views 1)${NC}"

PUSH_OUTPUT=$($CLI push test_small.txt --expire 10m --max-views 1 2>&1)
URL=$(echo "$PUSH_OUTPUT" | grep "Secure URL" | awk '{print $NF}')

echo "   Attempt 1 (Should Succeed)..."
$CLI pull "$URL" > /dev/null

echo "   Attempt 2 (Should Fail)..."
PULL_OUT=$($CLI pull "$URL" 2>&1)

if echo "$PULL_OUT" | grep -q "reached its download limit"; then
    echo -e "${GREEN}   PASS: Race condition blocked.${NC}"
else
    echo -e "${RED}   FAIL: Limit not enforced! Output: $PULL_OUT${NC}"
    exit 1
fi

# ==========================================
# TEST 3: Garbage Collection (CAS)
# ==========================================
echo -e "\n${YELLOW}TEST 3: CAS Deduplication & Garbage Collection${NC}"

echo "   Pushing 10MB file (Run 1)..."
$CLI push test_large.bin --expire 1h --max-views 1 > /dev/null

echo "   Pushing 10MB file (Run 2 - Should trigger Dedup)..."
$CLI push test_large.bin --expire 1h --max-views 1 > /dev/null

# Check Stats
STATS_OUT=$($CLI stats 2>&1)
echo "$STATS_OUT"

if echo "$STATS_OUT" | grep -q "Storage Saved"; then
    if echo "$STATS_OUT" | grep -q "0 B"; then
         echo -e "${RED}   FAIL: Storage Saved is 0 B. Deduplication failed.${NC}"
         exit 1
    else
         echo -e "${GREEN}   PASS: Storage deduplication active.${NC}"
    fi
else
    echo -e "${RED}   FAIL: Could not read stats.${NC}"
    exit 1
fi

# ==========================================
# CLEANUP
# ==========================================
echo -e "\n${BLUE}Cleaning up...${NC}"
rm test_small.txt test_large.bin downloaded_test_small.txt 2>/dev/null

echo -e "\n${GREEN}SUCCESS: All Integration Tests Passed!${NC}"