#!/bin/bash
# losp Conformance Test Runner
# Usage: LOSP_BIN=/path/to/losp ./run_tests.sh [category]

set -e

LOSP_BIN="${LOSP_BIN:-losp}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PASSED=0
FAILED=0
SKIPPED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

run_test() {
    local test_file="$1"
    local test_name="${test_file#$SCRIPT_DIR/}"

    # Extract expected output (all # EXPECTED: lines, joined with newlines)
    # Note: We strip exactly one space after "EXPECTED:" if present
    local expected=""
    local input=""
    local in_expected=true
    while IFS= read -r line; do
        if [[ "$line" =~ ^#\ EXPECTED:\ ?(.*)$ ]]; then
            [[ -n "$expected" ]] && expected+=$'\n'
            expected+="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ^#\ INPUT:\ ?(.*)$ ]]; then
            # INPUT directive provides stdin for READ calls (supports \n escapes)
            input=$(echo -e "${BASH_REMATCH[1]}")
        else
            break
        fi
    done < "$test_file"

    # Get losp code (everything after directive lines)
    local code
    code=$(sed '/^# EXPECTED:/d; /^# INPUT:/d' "$test_file")

    # Run test with fresh database (isolation)
    local tmpdb=$(mktemp)
    local actual
    if [[ -n "$input" ]]; then
        # Use file-based execution with separate stdin for READ
        local tmpcode=$(mktemp)
        echo "$code" > "$tmpcode"
        actual=$(echo "$input" | "$LOSP_BIN" -db "$tmpdb" -no-prompt -f "$tmpcode" 2>&1) || true
        rm -f "$tmpcode"
    else
        actual=$(echo "$code" | "$LOSP_BIN" -db "$tmpdb" -no-prompt 2>&1) || true
    fi
    rm -f "$tmpdb"

    # Compare
    if [[ "$actual" == "$expected" ]]; then
        echo -e "${GREEN}PASS${NC} $test_name"
        ((PASSED++)) || true
    else
        echo -e "${RED}FAIL${NC} $test_name"
        echo "  Expected: '$(echo "$expected" | head -c 80)'"
        echo "  Actual:   '$(echo "$actual" | head -c 80)'"
        ((FAILED++)) || true
    fi
}

# Export function for use in subshell
export -f run_test
export LOSP_BIN SCRIPT_DIR RED GREEN YELLOW NC

# Find and run tests
if [[ -n "$1" ]]; then
    # Run specific category
    while IFS= read -r f; do
        run_test "$f"
    done < <(find "$SCRIPT_DIR/$1" -name "*.losp" -type f | sort)
else
    # Run all tests
    while IFS= read -r f; do
        run_test "$f"
    done < <(find "$SCRIPT_DIR" -name "*.losp" -type f | sort)
fi

echo ""
echo "Results: $PASSED passed, $FAILED failed, $SKIPPED skipped"
[[ $FAILED -eq 0 ]] && exit 0 || exit 1
