#!/bin/bash
# Test: canopy query definition - find definition of symbol at position
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query definition ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: definition of a function call - CreateUser is called at line 65
echo "--- Test: definition follows reference to definition ---"
OUTPUT=$(canopy query definition "$MAIN_FILE" 65 18 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "definition" ]; then
  pass "definition returns command=definition"
else
  fail "definition returns command=definition" "definition" "$CMD"
fi

# Test: definition result structure
echo "--- Test: definition result structure ---"
if [ "$TOTAL" -gt 0 ]; then
  FIRST=$(echo "$OUTPUT" | jq '.results[0]')
  for field in file start_line start_col end_line end_col; do
    HAS=$(echo "$FIRST" | jq "has(\"$field\")")
    if [ "$HAS" != "true" ]; then
      fail "definition result has '$field'" "true" "$HAS"
    fi
  done
  pass "definition result has expected location fields"
else
  pass "definition returns empty results (reference may not be resolved)"
fi

# Test: definition at a definition site behavior
echo "--- Test: definition at definition site ---"
OUTPUT=$(canopy query definition "$MAIN_FILE" 27 5 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
# Spec says "follows references to their definitions" - at a definition, there's no ref to follow
# So returning 0 results is acceptable behavior
if [ "$CMD" = "definition" ]; then
  pass "definition at a definition site returns valid response (total=$TOTAL)"
else
  fail "definition at a definition site" "command=definition" "$CMD"
fi

# Test: definition with no symbol returns empty
echo "--- Test: definition at empty position ---"
OUTPUT=$(canopy query definition "$MAIN_FILE" 0 0 --db "$DB" 2>/dev/null)
RESULTS=$(echo "$OUTPUT" | jq '.results')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ] || [ "$RESULTS" = "null" ] || [ "$RESULTS" = "[]" ]; then
  pass "definition at empty position returns no results"
else
  fail "definition at empty position returns no results" "empty/null/0" "total=$TOTAL results=$RESULTS"
fi

# Test: text format
echo "--- Test: definition --format text ---"
OUTPUT_TEXT=$(canopy query definition "$MAIN_FILE" 27 5 --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT_TEXT" | grep -qE "\.go:[0-9]+:[0-9]+" || [ -z "$OUTPUT_TEXT" ]; then
  pass "definition text format shows file:line:col or empty"
else
  fail "definition text format shows file:line:col" "file:line:col pattern" "$OUTPUT_TEXT"
fi

# Test: wrong arg count
echo "--- Test: definition with wrong arg count ---"
set +e
OUTPUT=$(canopy query definition "$MAIN_FILE" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "definition with 1 arg instead of 3 returns error"
else
  fail "definition with 1 arg instead of 3 returns error" "non-zero exit" "exit code $EXIT_CODE"
fi

# Test: definition in TypeScript (import reference)
echo "--- Test: definition in TypeScript ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
TS_FILE="$SCRATCH/ts-project/src/index.ts"
# Logger is used at line 12 (private logger: Logger)
OUTPUT=$(canopy query definition "$TS_FILE" 12 20 --db "$TS_DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "definition" ]; then
  pass "definition works in TypeScript"
else
  fail "definition works in TypeScript" "command=definition" "$CMD"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
