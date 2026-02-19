#!/usr/bin/env bash
# grammar/validate.sh — Build the losp syntax checker and run it against
# all conformance test files.
#
# Usage (from repo root):
#   ./grammar/validate.sh
#
# Or from the grammar/ directory:
#   ./validate.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
IMAGE_NAME="losp-grammar"

echo "=== Building losp syntax checker ==="
docker build -t "$IMAGE_NAME" "$SCRIPT_DIR"

echo ""
echo "=== Validating conformance tests ==="
docker run --rm \
    -v "$REPO_ROOT/tests/conformance:/tests:ro" \
    "$IMAGE_NAME" \
    --dir /tests
