#!/bin/bash
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (c) 2023-2026 Nicholas R. Perez

# losp Conformance Test Runner
# Usage: LOSP_BIN=/path/to/losp [LOSP_PROVIDER=ollama] [LOSP_MODEL=model] ./run_tests.sh [category]

set -e

LOSP_BIN="${LOSP_BIN:-losp}"
LOSP_PROVIDER="${LOSP_PROVIDER:-}"
LOSP_MODEL="${LOSP_MODEL:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Build extra flags from environment
LOSP_EXTRA_FLAGS=""
[[ -n "$LOSP_PROVIDER" ]] && LOSP_EXTRA_FLAGS="$LOSP_EXTRA_FLAGS -provider $LOSP_PROVIDER"
[[ -n "$LOSP_MODEL" ]] && LOSP_EXTRA_FLAGS="$LOSP_EXTRA_FLAGS -model $LOSP_MODEL"
PASSED=0
FAILED=0
SKIPPED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# --- Fake HTTP server (for 36_http tests) ---
# Serves deterministic responses on 127.0.0.1:$LOSP_HTTP_PORT (default 8473):
#   /hello  -> "hello-from-server"
#   /method -> the request method (GET, POST, ...)
#   /echo   -> the request body
#   /header -> the value of the X-Losp request header
#   *       -> "not-found"
# A single `nc -lk` listener keeps the port bound between requests (no accept
# gap between sequential tests). Responses set Content-Length and
# Connection: close so clients read the full body and hang up; nc keeps
# listening for the next connection.
HTTP_PORT="${LOSP_HTTP_PORT:-8473}"
HTTP_FIFO=""
HTTP_SERVER_PID=""

http_handle_requests() {
    local reqline method rest path hdr clen xlosp body resp
    while IFS=$'\r' read -r reqline _; do
        [[ -z "$reqline" ]] && continue
        method=${reqline%% *}
        rest=${reqline#* }
        path=${rest%% *}

        clen=0
        xlosp=""
        while IFS=$'\r' read -r hdr _; do
            [[ -z "$hdr" ]] && break
            case "${hdr,,}" in
                content-length:*) clen=$(tr -d ' ' <<< "${hdr#*:}") ;;
                x-losp:*) xlosp=$(sed 's/^ *//' <<< "${hdr#*:}") ;;
            esac
        done

        body=""
        if (( clen > 0 )); then
            IFS= read -r -N "$clen" body
        fi

        case "$path" in
            /hello)  resp="hello-from-server" ;;
            /method) resp="$method" ;;
            /echo)   resp="$body" ;;
            /header) resp="$xlosp" ;;
            *)       resp="not-found" ;;
        esac

        printf 'HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %s\r\nConnection: close\r\n\r\n%s' "${#resp}" "$resp"
    done
}

start_http_server() {
    HTTP_FIFO=$(mktemp -u)
    mkfifo "$HTTP_FIFO"
    ( LC_ALL=C http_handle_requests < "$HTTP_FIFO" | nc -lk 127.0.0.1 "$HTTP_PORT" > "$HTTP_FIFO" ) &
    HTTP_SERVER_PID=$!
    # Wait for the listener to come up
    for _ in $(seq 1 50); do
        if nc -z 127.0.0.1 "$HTTP_PORT" 2>/dev/null; then
            return 0
        fi
        sleep 0.1
    done
    echo -e "${YELLOW}WARNING${NC} fake HTTP server failed to start on port $HTTP_PORT" >&2
}

stop_http_server() {
    if [[ -n "$HTTP_SERVER_PID" ]]; then
        # Kill the server subshell and its children (handler + nc)
        pkill -P "$HTTP_SERVER_PID" 2>/dev/null || true
        kill "$HTTP_SERVER_PID" 2>/dev/null || true
    fi
    rm -f "$HTTP_FIFO"
}

trap stop_http_server EXIT
start_http_server

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
        actual=$(echo "$input" | "$LOSP_BIN" -db "$tmpdb" $LOSP_EXTRA_FLAGS -f "$tmpcode" 2>&1) || true
        rm -f "$tmpcode"
    else
        actual=$(echo "$code" | "$LOSP_BIN" -db "$tmpdb" $LOSP_EXTRA_FLAGS 2>&1) || true
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
export LOSP_BIN LOSP_EXTRA_FLAGS SCRIPT_DIR RED GREEN YELLOW NC

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
