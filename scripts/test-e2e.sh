#!/bin/bash

# E2E Test Runner for Sshifu
# This script runs all end-to-end tests

set -e

echo "=========================================="
echo "  Sshifu End-to-End Test Suite"
echo "=========================================="
echo ""

cd "$(dirname "$0")/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

echo "Running Go e2e tests..."
echo ""

# Run e2e tests
if go test -v ./e2e/... -count=1; then
    echo ""
    echo -e "${GREEN}✓ All e2e tests passed${NC}"
    TESTS_PASSED=1
else
    echo ""
    echo -e "${RED}✗ Some e2e tests failed${NC}"
    TESTS_FAILED=1
fi

echo ""
echo "=========================================="
echo "  Test Summary"
echo "=========================================="

if [ $TESTS_PASSED -eq 1 ] && [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
