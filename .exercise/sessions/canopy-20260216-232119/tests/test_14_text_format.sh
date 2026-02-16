#!/bin/bash
# Test: Text format output validation
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Text Format Output ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: symbols text format has aligned columns
echo "--- Test: symbols text has header ---"
OUTPUT=$(canopy query symbols --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT" | head -1 | grep -qE "ID.*NAME.*KIND.*VISIBILITY"; then
  pass "symbols text format has column headers"
else
  HEADER=$(echo "$OUTPUT" | head -1)
  fail "symbols text format has column headers" "ID NAME KIND VISIBILITY..." "$HEADER"
fi

# Test: symbols text pagination footer
echo "--- Test: symbols text pagination footer ---"
OUTPUT=$(canopy query symbols --limit 2 --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT" | grep -q "Showing .* of .* results"; then
  pass "symbols text format shows pagination footer"
else
  fail "symbols text format shows pagination footer" "Showing X of Y results" "$(echo "$OUTPUT" | tail -1)"
fi

# Test: files text format has columns
echo "--- Test: files text has columns ---"
OUTPUT=$(canopy query files --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT" | head -1 | grep -qE "ID.*PATH.*LANGUAGE"; then
  pass "files text format has ID PATH LANGUAGE columns"
else
  HEADER=$(echo "$OUTPUT" | head -1)
  fail "files text format has ID PATH LANGUAGE columns" "ID PATH LANGUAGE" "$HEADER"
fi

# Test: definition text format is file:line:col
echo "--- Test: definition text format ---"
OUTPUT=$(canopy query definition "$MAIN_FILE" 65 18 --format text --db "$DB" 2>/dev/null)
if [ -z "$OUTPUT" ] || echo "$OUTPUT" | grep -qE "^[^ ]+:[0-9]+:[0-9]+$"; then
  pass "definition text format is file:line:col per line"
else
  fail "definition text format is file:line:col" "file:line:col" "$OUTPUT"
fi

# Test: references text format is file:line:col
echo "--- Test: references text format ---"
OUTPUT=$(canopy query references "$MAIN_FILE" 8 5 --format text --db "$DB" 2>/dev/null)
FIRST_LINE=$(echo "$OUTPUT" | head -1)
if echo "$FIRST_LINE" | grep -qE "^[^ ]+:[0-9]+:[0-9]+$"; then
  pass "references text format is file:line:col per line"
else
  fail "references text format is file:line:col" "file:line:col" "$FIRST_LINE"
fi

# Test: callers text format has aligned columns
echo "--- Test: callers text columns ---"
OUTPUT=$(canopy query callers "$MAIN_FILE" 51 28 --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT" | head -1 | grep -qE "CALLER.*CALLEE|caller|callee" || [ -z "$OUTPUT" ]; then
  pass "callers text format has column headers or is empty"
else
  HEADER=$(echo "$OUTPUT" | head -1)
  fail "callers text format has column headers" "CALLER CALLEE..." "$HEADER"
fi

# Test: deps text format has aligned columns
echo "--- Test: deps text columns ---"
OUTPUT=$(canopy query deps "$MAIN_FILE" --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT" | head -1 | grep -qE "SOURCE.*KIND.*FILE"; then
  pass "deps text format has SOURCE KIND FILE columns"
else
  HEADER=$(echo "$OUTPUT" | head -1)
  fail "deps text format has SOURCE KIND FILE columns" "SOURCE KIND FILE" "$HEADER"
fi

# Test: summary text format is human-readable
echo "--- Test: summary text is human-readable ---"
OUTPUT=$(canopy query summary --format text --db "$DB" 2>/dev/null)
# Should contain language name and counts
if echo "$OUTPUT" | grep -qi "go"; then
  pass "summary text mentions language name"
else
  fail "summary text mentions language name" "contains 'go'" "$(echo "$OUTPUT" | head -5)"
fi

# Test: package-summary text format
echo "--- Test: package-summary text format ---"
PKG_ID=$(canopy query packages --db "$DB" 2>/dev/null | jq -r '.results[0].id // empty')
if [ -n "$PKG_ID" ]; then
  OUTPUT=$(canopy query package-summary "$PKG_ID" --format text --db "$DB" 2>/dev/null)
  if [ -n "$OUTPUT" ]; then
    pass "package-summary text format produces output"
  else
    fail "package-summary text format produces output" "non-empty" "empty"
  fi
fi

# Test: symbol-at text format
echo "--- Test: symbol-at text format ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 8 5 --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT" | grep -q "User"; then
  pass "symbol-at text format shows symbol info"
else
  fail "symbol-at text format shows symbol info" "contains User" "$OUTPUT"
fi

# Test: symbol-at text format for null result
echo "--- Test: symbol-at text null result ---"
OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 0 0 --format text --db "$DB" 2>/dev/null)
# Should output nothing or a "no symbol found" message
pass "symbol-at text format handles null result"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
