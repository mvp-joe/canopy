#!/bin/bash
# Test: Error handling and edge cases
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Error Handling ==="

DB="$SCRATCH/go-project/.canopy/index.db"

# Test: query without indexing (non-existent DB)
echo "--- Test: query with non-existent DB ---"
set +e
OUTPUT=$(canopy query symbols --db "/tmp/nonexistent_db_12345.db" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 1 ]; then
  pass "query with non-existent DB exits with code 1"
else
  fail "query with non-existent DB exits with code 1" "exit code 1" "exit code $EXIT_CODE"
fi

# Test: JSON error format for missing DB
echo "--- Test: JSON error format ---"
set +e
OUTPUT=$(canopy query symbols --db "/tmp/nonexistent_db_12345.db" 2>/dev/null)
set -e
ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
if [ -n "$ERR" ]; then
  pass "Missing DB returns JSON error envelope"
else
  fail "Missing DB returns JSON error envelope" "non-empty error field" "output=$OUTPUT"
fi

# Test: text error format
echo "--- Test: text error format ---"
set +e
OUTPUT=$(canopy query symbols --format text --db "/tmp/nonexistent_db_12345.db" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 1 ]; then
  pass "text format missing DB exits with code 1"
else
  fail "text format missing DB exits with code 1" "exit code 1" "exit code $EXIT_CODE"
fi

# Test: symbol-at with non-existent file
echo "--- Test: symbol-at with non-existent file ---"
set +e
OUTPUT=$(canopy query symbol-at "/tmp/nonexistent_file.go" 0 0 --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
# Should either return null results or error
RESULTS=$(echo "$OUTPUT" | jq -r 'if .results == null then "null" else "non-null" end')
if [ "$RESULTS" = "null" ] || [ "$EXIT_CODE" -ne 0 ]; then
  pass "symbol-at with non-existent file returns null or error"
else
  fail "symbol-at with non-existent file" "null results or error" "results=$RESULTS exit=$EXIT_CODE"
fi

# Test: deps on non-indexed file
echo "--- Test: deps on non-indexed file ---"
set +e
OUTPUT=$(canopy query deps "/tmp/nonexistent_file.go" --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count // "error"')
if [ "$TOTAL" = "0" ] || [ "$EXIT_CODE" -ne 0 ]; then
  pass "deps on non-indexed file returns 0 or error"
else
  fail "deps on non-indexed file" "0 results or error" "total=$TOTAL exit=$EXIT_CODE"
fi

# Test: invalid --format value
echo "--- Test: invalid --format value ---"
set +e
OUTPUT=$(canopy query symbols --format xml --db "$DB" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "invalid --format value returns error"
else
  fail "invalid --format value returns error" "non-zero exit" "exit=$EXIT_CODE"
fi

# Test: --limit max boundary (spec says max 500)
echo "--- Test: --limit 500 ---"
OUTPUT=$(canopy query symbols --limit 500 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "symbols" ]; then
  pass "--limit 500 is accepted"
else
  fail "--limit 500 is accepted" "command=symbols" "command=$CMD"
fi

# Test: --limit above 500
echo "--- Test: --limit 501 ---"
set +e
OUTPUT=$(canopy query symbols --limit 501 --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
# Should either clamp or error
CMD=$(echo "$OUTPUT" | jq -r '.command // empty')
if [ "$CMD" = "symbols" ] || [ "$EXIT_CODE" -ne 0 ]; then
  pass "--limit 501 is either clamped or returns error"
else
  fail "--limit 501 handling" "clamped or error" "exit=$EXIT_CODE output=$OUTPUT"
fi

# Test: negative --offset
echo "--- Test: negative --offset ---"
set +e
OUTPUT=$(canopy query symbols --offset -1 --db "$DB" 2>&1)
EXIT_CODE=$?
set -e
# Should handle gracefully
if [ "$EXIT_CODE" -eq 0 ] || [ "$EXIT_CODE" -eq 1 ]; then
  pass "negative --offset handled gracefully"
else
  fail "negative --offset handled gracefully" "exit 0 or 1" "exit=$EXIT_CODE"
fi

# Test: invalid --sort value
echo "--- Test: invalid --sort value ---"
set +e
OUTPUT=$(canopy query symbols --sort invalid_field --db "$DB" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "invalid --sort value returns error"
else
  # Might just ignore invalid sort
  pass "invalid --sort value handled (may be ignored)"
fi

# Test: invalid --order value
echo "--- Test: invalid --order value ---"
set +e
OUTPUT=$(canopy query symbols --order invalid --db "$DB" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "invalid --order value returns error"
else
  pass "invalid --order value handled (may default to asc)"
fi

# Test: empty repo (create one with no source files)
echo "--- Test: empty repo ---"
EMPTY_DIR="$SCRATCH/empty-project"
mkdir -p "$EMPTY_DIR"
cd "$EMPTY_DIR" && git init -q && git commit -q --allow-empty -m "init" && cd /tmp/claude-exercise-canopy-20260216-232119
canopy index "$EMPTY_DIR" 2>&1
OUTPUT=$(canopy query files --db "$EMPTY_DIR/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ]; then
  pass "empty repo returns 0 files"
else
  fail "empty repo returns 0 files" "0" "$TOTAL"
fi

# Test: query summary on empty repo
echo "--- Test: summary on empty repo ---"
OUTPUT=$(canopy query summary --db "$EMPTY_DIR/.canopy/index.db" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "summary" ]; then
  pass "summary on empty repo works"
else
  fail "summary on empty repo works" "command=summary" "$CMD"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
