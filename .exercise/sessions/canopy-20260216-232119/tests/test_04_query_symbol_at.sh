#!/bin/bash
# Test: canopy query symbol-at - find symbol at position
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query symbol-at ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: Find User struct (line 8, 0-based = line 8 "type User struct")
echo "--- Test: symbol-at finds User struct ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 8 5 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
if [ "$CMD" = "symbol-at" ] && [ "$NAME" = "User" ]; then
  pass "symbol-at finds User at line 8"
else
  fail "symbol-at finds User at line 8" "command=symbol-at, name=User" "command=$CMD, name=$NAME"
fi

# Test: symbol-at result fields
echo "--- Test: symbol-at result fields ---"
for field in id name kind visibility file start_line start_col end_line end_col; do
  HAS=$(echo "$OUTPUT" | jq ".results | has(\"$field\")")
  if [ "$HAS" != "true" ]; then
    fail "symbol-at result has '$field'" "true" "$HAS"
  fi
done
pass "symbol-at result has all expected fields"

# Test: total_count should be 1
echo "--- Test: symbol-at total_count ---"
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" = "1" ]; then
  pass "symbol-at returns total_count=1"
else
  fail "symbol-at returns total_count=1" "1" "$TOTAL"
fi

# Test: symbol-at on a function (NewUserService at line 27, 0-based)
echo "--- Test: symbol-at finds function ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 27 5 --db "$DB" 2>/dev/null)
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
KIND=$(echo "$OUTPUT" | jq -r '.results.kind // empty')
if [ "$NAME" = "NewUserService" ] && [ "$KIND" = "function" ]; then
  pass "symbol-at finds NewUserService function"
else
  fail "symbol-at finds NewUserService function" "NewUserService, function" "name=$NAME, kind=$KIND"
fi

# Test: symbol-at with no symbol (empty area, e.g., blank line)
echo "--- Test: symbol-at returns null for empty position ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 0 0 --db "$DB" 2>/dev/null)
RESULTS=$(echo "$OUTPUT" | jq -r '.results')
if [ "$RESULTS" = "null" ]; then
  pass "symbol-at returns null for position with no symbol"
else
  fail "symbol-at returns null for position with no symbol" "null" "$RESULTS"
fi

# Test: text format
echo "--- Test: symbol-at --format text ---"
OUTPUT_TEXT=$(canopy query symbol-at "$MAIN_FILE" 8 5 --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT_TEXT" | grep -q "User"; then
  pass "symbol-at text format shows symbol name"
else
  fail "symbol-at text format shows symbol name" "contains User" "$OUTPUT_TEXT"
fi

# Test: wrong number of args (missing col)
echo "--- Test: symbol-at with wrong arg count ---"
set +e
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 8 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "symbol-at with 2 args instead of 3 returns error"
else
  fail "symbol-at with 2 args instead of 3 returns error" "non-zero exit" "exit code $EXIT_CODE"
fi

# Test: symbol-at with huge line number
echo "--- Test: symbol-at with huge line number ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 99999 0 --db "$DB" 2>/dev/null)
RESULTS=$(echo "$OUTPUT" | jq -r '.results')
if [ "$RESULTS" = "null" ]; then
  pass "symbol-at with line 99999 returns null"
else
  fail "symbol-at with line 99999 returns null" "null" "$RESULTS"
fi

# Test: symbol-at with huge column number
echo "--- Test: symbol-at with huge column number ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 8 99999 --db "$DB" 2>/dev/null)
RESULTS=$(echo "$OUTPUT" | jq -r '.results')
if [ "$RESULTS" = "null" ]; then
  pass "symbol-at with col 99999 returns null"
else
  fail "symbol-at with col 99999 returns null" "null" "$RESULTS"
fi

# Test: symbol-at in TypeScript project
echo "--- Test: symbol-at in TypeScript ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
TS_FILE="$SCRATCH/ts-project/src/index.ts"
# Application class should be at line 10 (0-based)
OUTPUT=$(canopy query symbol-at "$TS_FILE" 10 6 --db "$TS_DB" 2>/dev/null)
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
if [ "$NAME" = "Application" ]; then
  pass "symbol-at finds Application class in TypeScript"
else
  fail "symbol-at finds Application class in TypeScript" "Application" "$NAME"
fi

# Test: symbol-at in Python project
echo "--- Test: symbol-at in Python ---"
PY_DB="$SCRATCH/python-project/.canopy/index.db"
PY_FILE="$SCRATCH/python-project/main.py"
# MathService class at line 13 (0-based)
OUTPUT=$(canopy query symbol-at "$PY_FILE" 13 6 --db "$PY_DB" 2>/dev/null)
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
if [ "$NAME" = "MathService" ]; then
  pass "symbol-at finds MathService class in Python"
else
  fail "symbol-at finds MathService class in Python" "MathService" "$NAME"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
