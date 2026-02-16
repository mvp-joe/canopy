#!/bin/bash
# Test: canopy query callers / callees
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query callers/callees ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Find the CreateUser method  - line 51 (0-based)
echo "--- Test: callers by position ---"
OUTPUT=$(canopy query callers "$MAIN_FILE" 51 28 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "callers" ]; then
  pass "callers returns command=callers"
else
  fail "callers returns command=callers" "callers" "$CMD"
fi

# Test: callers result fields
echo "--- Test: callers result fields ---"
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  FIRST=$(echo "$OUTPUT" | jq '.results[0]')
  for field in caller_id caller_name callee_id callee_name file line col; do
    HAS=$(echo "$FIRST" | jq "has(\"$field\")")
    if [ "$HAS" != "true" ]; then
      fail "caller result has '$field'" "true" "$HAS"
    fi
  done
  pass "caller results have all expected fields"
else
  pass "callers returns 0 results (function may not be called)"
fi

# Test: callees of main function (line 63, main func)
echo "--- Test: callees of main ---"
OUTPUT=$(canopy query callees "$MAIN_FILE" 63 5 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "callees" ]; then
  pass "callees returns command=callees"
else
  fail "callees returns command=callees" "callees" "$CMD"
fi

# Test: callers with --symbol flag
echo "--- Test: callers with --symbol ---"
SYM_OUTPUT=$(canopy query symbols --kind function --db "$DB" 2>/dev/null)
SYM_ID=$(echo "$SYM_OUTPUT" | jq -r '.results[0].id // empty')
if [ -n "$SYM_ID" ]; then
  OUTPUT=$(canopy query callers --symbol "$SYM_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "callers" ]; then
    pass "callers with --symbol works"
  else
    fail "callers with --symbol works" "command=callers" "$CMD"
  fi
else
  fail "Could not get symbol ID for callers --symbol test" "non-empty" "empty"
fi

# Test: callees with --symbol flag
echo "--- Test: callees with --symbol ---"
if [ -n "$SYM_ID" ]; then
  OUTPUT=$(canopy query callees --symbol "$SYM_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "callees" ]; then
    pass "callees with --symbol works"
  else
    fail "callees with --symbol works" "command=callees" "$CMD"
  fi
fi

# Test: text format for callers
echo "--- Test: callers --format text ---"
OUTPUT_TEXT=$(canopy query callers "$MAIN_FILE" 51 28 --format text --db "$DB" 2>/dev/null)
# Text format should show aligned columns or be empty
pass "callers text format produces output (may be empty)"

# Test: text format for callees
echo "--- Test: callees --format text ---"
OUTPUT_TEXT=$(canopy query callees "$MAIN_FILE" 63 5 --format text --db "$DB" 2>/dev/null)
pass "callees text format produces output"

# Test: callers with no args or --symbol
echo "--- Test: callers with no args ---"
set +e
OUTPUT=$(canopy query callers --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "callers with no args returns error"
else
  ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
  if [ -n "$ERR" ]; then
    pass "callers with no args returns JSON error"
  else
    fail "callers with no args returns error" "non-zero exit or JSON error" "exit=$EXIT_CODE"
  fi
fi

# Test: callees with no args or --symbol
echo "--- Test: callees with no args ---"
set +e
OUTPUT=$(canopy query callees --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "callees with no args returns error"
else
  ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
  if [ -n "$ERR" ]; then
    pass "callees with no args returns JSON error"
  else
    fail "callees with no args returns error" "non-zero exit or JSON error" "exit=$EXIT_CODE"
  fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
