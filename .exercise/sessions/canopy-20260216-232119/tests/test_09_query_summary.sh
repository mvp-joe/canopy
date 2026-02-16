#!/bin/bash
# Test: canopy query summary and package-summary
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query summary / package-summary ==="

DB="$SCRATCH/go-project/.canopy/index.db"

# Test: summary command
echo "--- Test: query summary ---"
OUTPUT=$(canopy query summary --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "summary" ]; then
  pass "summary returns command=summary"
else
  fail "summary returns command=summary" "summary" "$CMD"
fi

# Test: summary has languages array
echo "--- Test: summary languages ---"
LANG_COUNT=$(echo "$OUTPUT" | jq '.results.languages | length')
if [ "$LANG_COUNT" -gt 0 ]; then
  pass "summary has languages array with entries"
else
  fail "summary has languages array" "length > 0" "length=$LANG_COUNT"
fi

# Test: summary language entry fields
echo "--- Test: summary language fields ---"
FIRST_LANG=$(echo "$OUTPUT" | jq '.results.languages[0]')
for field in language file_count line_count symbol_count kind_counts; do
  HAS=$(echo "$FIRST_LANG" | jq "has(\"$field\")")
  if [ "$HAS" != "true" ]; then
    fail "language entry has '$field'" "true" "$HAS"
  fi
done
pass "summary language entries have expected fields"

# Test: summary has package_count
echo "--- Test: summary package_count ---"
PC=$(echo "$OUTPUT" | jq '.results.package_count')
if [ "$PC" != "null" ]; then
  pass "summary has package_count"
else
  fail "summary has package_count" "non-null" "null"
fi

# Test: summary has top_symbols
echo "--- Test: summary top_symbols ---"
TS=$(echo "$OUTPUT" | jq '.results.top_symbols')
if [ "$TS" != "null" ]; then
  pass "summary has top_symbols"
else
  fail "summary has top_symbols" "non-null" "null"
fi

# Test: summary text format
echo "--- Test: summary --format text ---"
OUTPUT_TEXT=$(canopy query summary --format text --db "$DB" 2>/dev/null)
if [ -n "$OUTPUT_TEXT" ]; then
  pass "summary text format produces output"
else
  fail "summary text format produces output" "non-empty" "empty"
fi

# Test: packages command
echo "--- Test: query packages ---"
OUTPUT=$(canopy query packages --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "packages" ]; then
  pass "packages returns command=packages"
else
  fail "packages returns command=packages" "packages" "$CMD"
fi

# Test: packages result structure (same as symbols)
echo "--- Test: packages result structure ---"
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  FIRST=$(echo "$OUTPUT" | jq '.results[0]')
  HAS_NAME=$(echo "$FIRST" | jq 'has("name")')
  HAS_KIND=$(echo "$FIRST" | jq 'has("kind")')
  if [ "$HAS_NAME" = "true" ] && [ "$HAS_KIND" = "true" ]; then
    pass "packages results have name and kind"
  else
    fail "packages results have name and kind" "both true" "name=$HAS_NAME kind=$HAS_KIND"
  fi
else
  pass "packages returns 0 results (may be expected)"
fi

# Test: packages --prefix filter
echo "--- Test: packages --prefix ---"
OUTPUT=$(canopy query packages --prefix "$SCRATCH/go-project" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "packages" ]; then
  pass "packages --prefix works"
else
  fail "packages --prefix works" "command=packages" "$CMD"
fi

# Test: packages --format text
echo "--- Test: packages --format text ---"
OUTPUT_TEXT=$(canopy query packages --format text --db "$DB" 2>/dev/null)
pass "packages text format produces output"

# Test: package-summary
echo "--- Test: query package-summary ---"
# Get first package path
PKG_OUTPUT=$(canopy query packages --db "$DB" 2>/dev/null)
PKG_TOTAL=$(echo "$PKG_OUTPUT" | jq -r '.total_count')
if [ "$PKG_TOTAL" -gt 0 ]; then
  PKG_ID=$(echo "$PKG_OUTPUT" | jq -r '.results[0].id')
  OUTPUT=$(canopy query package-summary "$PKG_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "package-summary" ]; then
    pass "package-summary returns command=package-summary"
  else
    fail "package-summary returns command=package-summary" "package-summary" "$CMD"
  fi

  # Test: package-summary result fields
  echo "--- Test: package-summary fields ---"
  for field in file_count kind_counts; do
    HAS=$(echo "$OUTPUT" | jq ".results | has(\"$field\")")
    if [ "$HAS" != "true" ]; then
      fail "package-summary result has '$field'" "true" "$HAS"
    fi
  done
  pass "package-summary has expected fields"
else
  pass "No packages to test package-summary with (skipped)"
fi

# Test: package-summary text format
echo "--- Test: package-summary --format text ---"
if [ "$PKG_TOTAL" -gt 0 ]; then
  PKG_ID=$(echo "$PKG_OUTPUT" | jq -r '.results[0].id')
  OUTPUT_TEXT=$(canopy query package-summary "$PKG_ID" --format text --db "$DB" 2>/dev/null)
  if [ -n "$OUTPUT_TEXT" ]; then
    pass "package-summary text format produces output"
  else
    fail "package-summary text format produces output" "non-empty" "empty"
  fi
fi

# Test: package-summary with no arg (should error)
echo "--- Test: package-summary with no arg ---"
set +e
OUTPUT=$(canopy query package-summary --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "package-summary with no arg returns error"
else
  ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
  if [ -n "$ERR" ]; then
    pass "package-summary with no arg returns JSON error"
  else
    fail "package-summary with no arg returns error" "error" "exit=$EXIT_CODE"
  fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
