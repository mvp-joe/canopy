#!/bin/bash
# Test: canopy query deps / dependents
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query deps / dependents ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: deps for main.go (imports fmt and strings)
echo "--- Test: deps for main.go ---"
OUTPUT=$(canopy query deps "$MAIN_FILE" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "deps" ] && [ "$TOTAL" -gt 0 ]; then
  pass "deps returns imports for main.go"
else
  fail "deps returns imports for main.go" "command=deps, total>0" "command=$CMD, total=$TOTAL"
fi

# Test: deps result fields
echo "--- Test: deps result fields ---"
FIRST=$(echo "$OUTPUT" | jq '.results[0]')
for field in file_id file_path source kind; do
  HAS=$(echo "$FIRST" | jq "has(\"$field\")")
  if [ "$HAS" != "true" ]; then
    fail "dep result has '$field'" "true" "$HAS"
  fi
done
pass "deps results have expected fields"

# Test: deps shows fmt import
echo "--- Test: deps includes fmt ---"
HAS_FMT=$(echo "$OUTPUT" | jq '[.results[].source] | any(. == "fmt")')
if [ "$HAS_FMT" = "true" ]; then
  pass "deps for main.go includes 'fmt'"
else
  fail "deps for main.go includes 'fmt'" "true" "$HAS_FMT"
fi

# Test: deps shows strings import
echo "--- Test: deps includes strings ---"
HAS_STRINGS=$(echo "$OUTPUT" | jq '[.results[].source] | any(. == "strings")')
if [ "$HAS_STRINGS" = "true" ]; then
  pass "deps for main.go includes 'strings'"
else
  fail "deps for main.go includes 'strings'" "true" "$HAS_STRINGS"
fi

# Test: deps text format
echo "--- Test: deps --format text ---"
OUTPUT_TEXT=$(canopy query deps "$MAIN_FILE" --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT_TEXT" | grep -q "fmt"; then
  pass "deps text format shows imports"
else
  fail "deps text format shows imports" "contains fmt" "$OUTPUT_TEXT"
fi

# Test: deps with no arg
echo "--- Test: deps with no arg ---"
set +e
OUTPUT=$(canopy query deps --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "deps with no arg returns error"
else
  ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
  if [ -n "$ERR" ]; then
    pass "deps with no arg returns JSON error"
  else
    fail "deps with no arg returns error" "error" "exit=$EXIT_CODE"
  fi
fi

# Test: dependents for "fmt"
echo "--- Test: dependents for fmt ---"
OUTPUT=$(canopy query dependents "fmt" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "dependents" ]; then
  pass "dependents returns command=dependents"
else
  fail "dependents returns command=dependents" "dependents" "$CMD"
fi

# Test: dependents result includes main.go
echo "--- Test: dependents for fmt includes main.go ---"
if [ "$TOTAL" -gt 0 ]; then
  HAS_MAIN=$(echo "$OUTPUT" | jq --arg f "$MAIN_FILE" '[.results[].file_path] | any(. == $f)')
  if [ "$HAS_MAIN" = "true" ]; then
    pass "dependents for 'fmt' includes main.go"
  else
    fail "dependents for 'fmt' includes main.go" "true" "$HAS_MAIN"
  fi
else
  fail "dependents for 'fmt' returns results" "total > 0" "total=$TOTAL"
fi

# Test: dependents text format
echo "--- Test: dependents --format text ---"
OUTPUT_TEXT=$(canopy query dependents "fmt" --format text --db "$DB" 2>/dev/null)
if [ -n "$OUTPUT_TEXT" ]; then
  pass "dependents text format produces output"
else
  fail "dependents text format produces output" "non-empty" "empty"
fi

# Test: dependents for non-existent source
echo "--- Test: dependents for non-existent source ---"
OUTPUT=$(canopy query dependents "nonexistent_module_xyz" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ]; then
  pass "dependents for non-existent source returns 0"
else
  fail "dependents for non-existent source returns 0" "total=0" "total=$TOTAL"
fi

# Test: deps in TypeScript project
echo "--- Test: deps in TypeScript ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
TS_FILE="$SCRATCH/ts-project/src/index.ts"
OUTPUT=$(canopy query deps "$TS_FILE" --db "$TS_DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "deps" ] && [ "$TOTAL" -gt 0 ]; then
  pass "deps works for TypeScript files"
else
  fail "deps works for TypeScript files" "command=deps, total>0" "command=$CMD, total=$TOTAL"
fi

# Test: deps in Python project
echo "--- Test: deps in Python ---"
PY_DB="$SCRATCH/python-project/.canopy/index.db"
PY_FILE="$SCRATCH/python-project/main.py"
OUTPUT=$(canopy query deps "$PY_FILE" --db "$PY_DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "deps" ]; then
  pass "deps works for Python files"
else
  fail "deps works for Python files" "command=deps" "command=$CMD"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
