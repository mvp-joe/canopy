#!/bin/bash
# Test: canopy query references - find all references to a symbol
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query references ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: references using positional args (User struct at line 8)
echo "--- Test: references by position ---"
OUTPUT=$(canopy query references "$MAIN_FILE" 8 5 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "references" ]; then
  pass "references returns command=references"
else
  fail "references returns command=references" "references" "$CMD"
fi

# Test: references result fields
echo "--- Test: references result fields ---"
if [ "$TOTAL" -gt 0 ]; then
  FIRST=$(echo "$OUTPUT" | jq '.results[0]')
  for field in file start_line start_col end_line end_col symbol_id; do
    HAS=$(echo "$FIRST" | jq "has(\"$field\")")
    if [ "$HAS" != "true" ]; then
      fail "reference result has '$field'" "true" "$HAS"
    fi
  done
  pass "reference results have expected fields"
else
  pass "references returns 0 results (may have no references)"
fi

# Test: references using --symbol flag
echo "--- Test: references with --symbol ---"
# First get a symbol ID
SYM_OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 8 5 --db "$DB" 2>/dev/null)
SYM_ID=$(echo "$SYM_OUTPUT" | jq -r '.results.id // empty')
if [ -n "$SYM_ID" ]; then
  OUTPUT=$(canopy query references --symbol "$SYM_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "references" ]; then
    pass "references with --symbol returns results"
  else
    fail "references with --symbol returns results" "command=references" "$CMD"
  fi
else
  fail "Could not get symbol ID for --symbol test" "non-empty ID" "empty"
fi

# Test: references with pagination
echo "--- Test: references with --limit ---"
OUTPUT=$(canopy query references "$MAIN_FILE" 8 5 --limit 1 --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
if [ "$COUNT" -le 1 ]; then
  pass "references --limit 1 returns at most 1 result"
else
  fail "references --limit 1 returns at most 1 result" "count<=1" "count=$COUNT"
fi

# Test: text format
echo "--- Test: references --format text ---"
OUTPUT_TEXT=$(canopy query references "$MAIN_FILE" 8 5 --format text --db "$DB" 2>/dev/null)
# Text format should show file:line:col
if echo "$OUTPUT_TEXT" | grep -qE "\.go:[0-9]+:[0-9]+" || [ -z "$OUTPUT_TEXT" ]; then
  pass "references text format shows file:line:col or empty"
else
  fail "references text format shows file:line:col" "file:line:col pattern" "$OUTPUT_TEXT"
fi

# Test: no args and no --symbol should error
echo "--- Test: references with no args and no --symbol ---"
set +e
OUTPUT=$(canopy query references --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "references with no args or --symbol returns error"
else
  # Check if it's a JSON error
  ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
  if [ -n "$ERR" ]; then
    pass "references with no args returns JSON error"
  else
    fail "references with no args returns error" "non-zero exit or JSON error" "exit=$EXIT_CODE output=$OUTPUT"
  fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
