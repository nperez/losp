#!/bin/bash
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (c) 2023-2026 Nicholas R. Perez

# Primer Fidelity Test Harness
# Tests how well a small model can write losp using only PRIMER_COMPACT.md

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

PRIMER="PRIMER_COMPACT.md"
LOSP="./losp"
MODEL="${LOSP_TEST_MODEL:-qwen3:30b-a3b-instruct-2507-q4_K_M}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"

# Build losp if needed
if [ ! -x "$LOSP" ]; then
    echo "Building losp..."
    go build -o losp ./cmd/losp
fi

PRIMER_CONTENT=$(cat "$PRIMER")
PASS=0
FAIL=0
RESULTS=""

run_test() {
    local category="$1"
    local name="$2"
    local expected="$3"
    local prompt="$4"

    local full_prompt="${PRIMER_CONTENT}

---

Write losp code: ${prompt}

Output ONLY losp code. No markdown. No explanation."

    echo '{"model": "'"$MODEL"'", "stream": false}' > /tmp/req.json
    jq --arg p "$full_prompt" '.prompt = $p' /tmp/req.json > /tmp/req2.json

    local code
    code=$(curl -s --max-time 180 "$OLLAMA_URL/api/generate" -d @/tmp/req2.json | jq -r '.response // empty')

    if [ -z "$code" ]; then
        echo "FAIL: $category/$name (no response)"
        ((FAIL++))
        RESULTS="${RESULTS}FAIL|$category|$name|no response||$expected\n"
        return
    fi

    local result
    result=$(echo "$code" | $LOSP 2>&1) || true
    local expected_norm
    expected_norm=$(echo -e "$expected")

    if [ "$result" = "$expected_norm" ]; then
        echo "PASS: $category/$name"
        ((PASS++))
        RESULTS="${RESULTS}PASS|$category|$name\n"
    else
        echo "FAIL: $category/$name"
        echo "  Expected: [$expected_norm]"
        echo "  Got:      [$result]"
        ((FAIL++))
        # Escape code for CSV-like storage
        local code_escaped="${code//$'\n'/\\n}"
        RESULTS="${RESULTS}FAIL|$category|$name|$code_escaped|$result|$expected_norm\n"
    fi
}

echo "========================================"
echo "Primer Fidelity Test"
echo "Model: $MODEL"
echo "Primer: $(wc -w < "$PRIMER") words"
echo "========================================"
echo ""

# Store tests
run_test "store" "imm_retrieve" "hello" "Store 'hello' in X, then retrieve X."
run_test "store" "def_execute" "world" "Define Greet with body 'world', then execute Greet."
run_test "store" "overwrite" "second" "Store 'first' in X, store 'second' in X, retrieve X."

# Execute tests
run_test "execute" "basic" "hello" "Define Msg with body 'hello' using deferred store. Execute Msg."

# Placeholder tests
run_test "placeholder" "single" "Value: Alice" "Define Show taking 'name' returning 'Value: <name>'. Call with Alice."
run_test "placeholder" "two_args" "B A" "Define Swap taking 'a' then 'b' returning '<b> <a>'. Call with A then B on separate lines."
run_test "placeholder" "global" "val" "Define Set taking 'x' with empty body. Call Set with 'val'. Retrieve x."

# IF tests
run_test "if" "true_literal" "yes" "IF with condition TRUE. Return 'yes' for true, 'no' for false."
run_test "if" "false_literal" "no" "IF with condition FALSE. Return 'yes' for true, 'no' for false."
run_test "if" "compare_eq" "match" "IF with COMPARE checking 'a' equals 'a'. Return 'match' or 'nomatch'."
run_test "if" "compare_neq" "nomatch" "IF with COMPARE checking 'a' equals 'b'. Return 'match' or 'nomatch'."

# Timing tests
run_test "timing" "immediate" "first" "Store 'first' in X. Define Tmpl that captures X at definition time. Store 'second' in X. Execute Tmpl."
run_test "timing" "deferred" "second" "Store 'first' in X. Define Tmpl that looks up X at execution time. Store 'second' in X. Execute Tmpl."

# Utility tests
run_test "util" "append" "alpha
beta" "Store 'alpha' in Data. Append 'beta' to Data. Retrieve Data."

# FOREACH test
run_test "foreach" "basic" "[a]
[b]
[c]" "Define Show taking 'item' returning '[<item>]'. Define Items with lines a, b, c. FOREACH Items with Show."

echo ""
echo "========================================"
echo "SUMMARY"
echo "========================================"
TOTAL=$((PASS + FAIL))
PCT=$((PASS * 100 / TOTAL))
echo "Passed: $PASS / $TOTAL ($PCT%)"
echo ""

# Show failure summary
echo "Failures by category:"
echo -e "$RESULTS" | grep "^FAIL" | cut -d'|' -f2 | sort | uniq -c | sort -rn

echo ""
echo "Fidelity: $PCT%"
