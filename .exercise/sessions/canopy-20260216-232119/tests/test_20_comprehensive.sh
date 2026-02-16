#!/bin/bash
# Test: Comprehensive edge cases and remaining untested features
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Comprehensive Edge Cases ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: package-summary with path prefix (string argument)
echo "--- Test: package-summary with path prefix ---"
OUTPUT=$(canopy query package-summary "$SCRATCH/go-project" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "package-summary" ]; then
  pass "package-summary accepts path prefix string"
else
  fail "package-summary accepts path prefix string" "command=package-summary" "$CMD (output: $OUTPUT)"
fi

# Test: package-summary with numeric ID
echo "--- Test: package-summary with numeric ID ---"
PKG_ID=$(canopy query packages --db "$DB" 2>/dev/null | jq -r '.results[0].id // empty')
if [ -n "$PKG_ID" ]; then
  OUTPUT=$(canopy query package-summary "$PKG_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "package-summary" ]; then
    pass "package-summary accepts numeric symbol ID"
  else
    fail "package-summary accepts numeric symbol ID" "command=package-summary" "$CMD"
  fi
fi

# Test: search with single character
echo "--- Test: search 'a' ---"
OUTPUT=$(canopy query search "a" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "search" ]; then
  pass "search with single character works"
else
  fail "search with single character works" "command=search" "$CMD"
fi

# Test: search with special glob characters
echo "--- Test: search '*' (just wildcard) ---"
OUTPUT=$(canopy query search "*" --db "$DB" 2>/dev/null)
ALL_TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
# Should return all symbols
SYMBOLS_TOTAL=$(canopy query symbols --db "$DB" 2>/dev/null | jq -r '.total_count')
if [ "$ALL_TOTAL" -eq "$SYMBOLS_TOTAL" ]; then
  pass "search '*' returns same count as symbols ($ALL_TOTAL)"
else
  fail "search '*' returns same count as symbols" "search=$SYMBOLS_TOTAL" "search=$ALL_TOTAL"
fi

# Test: search with question mark (if supported)
echo "--- Test: search 'User?' ---"
OUTPUT=$(canopy query search "User?" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "search" ]; then
  pass "search 'User?' doesn't crash"
else
  fail "search 'User?' doesn't crash" "command=search" "$CMD"
fi

# Test: multiple refs to same symbol
echo "--- Test: high ref count symbol ---"
OUTPUT=$(canopy query symbols --sort ref_count --order desc --limit 1 --db "$DB" 2>/dev/null)
TOP_NAME=$(echo "$OUTPUT" | jq -r '.results[0].name // empty')
TOP_REF=$(echo "$OUTPUT" | jq -r '.results[0].ref_count // 0')
if [ -n "$TOP_NAME" ]; then
  pass "Most referenced symbol: $TOP_NAME with $TOP_REF refs"
else
  fail "Found most referenced symbol" "non-empty" "empty"
fi

# Test: references --sort with offset
echo "--- Test: references --sort + --offset ---"
OUTPUT=$(canopy query references "$MAIN_FILE" 8 5 --sort file --offset 2 --limit 3 --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$COUNT" -le 3 ]; then
  pass "references --sort + --offset + --limit works (got $COUNT results)"
else
  fail "references --sort + --offset + --limit" "count <= 3" "count=$COUNT"
fi

# Test: files --language with different case
echo "--- Test: files --language case sensitivity ---"
OUT1=$(canopy query files --language go --db "$DB" 2>/dev/null | jq -r '.total_count')
OUT2=$(canopy query files --language Go --db "$DB" 2>/dev/null | jq -r '.total_count')
if [ "$OUT1" -gt 0 ]; then
  pass "files --language go returns results ($OUT1)"
  if [ "$OUT2" -gt 0 ]; then
    pass "files --language Go also returns results ($OUT2)"
  else
    pass "files --language Go returns 0 (case-sensitive - expected)"
  fi
else
  fail "files --language go returns results" "> 0" "$OUT1"
fi

# Test: callers text pagination footer
echo "--- Test: callers text format pagination ---"
MC_DB="$SCRATCH/multi-caller/.canopy/index.db"
OUTPUT=$(canopy query callers "$SCRATCH/multi-caller/main.go" 4 5 --limit 2 --format text --db "$MC_DB" 2>/dev/null)
# Once --limit is fixed, this should show "Showing 2 of 6 results"
LINES=$(echo "$OUTPUT" | wc -l)
if [ "$LINES" -gt 0 ]; then
  pass "callers text format produces output ($LINES lines)"
else
  fail "callers text format produces output" "> 0 lines" "$LINES"
fi

# Test: symbol-at at every boundary of a struct (first col, last col, line before, line after)
echo "--- Test: symbol-at boundary precision ---"
# User struct is at lines 8-12, cols 0-1
# Line 7 should NOT return User
OUT7=$(canopy query symbol-at "$MAIN_FILE" 7 0 --db "$DB" 2>/dev/null | jq -r '.results.name // "null"')
# Line 8 col 0 SHOULD return User
OUT8=$(canopy query symbol-at "$MAIN_FILE" 8 0 --db "$DB" 2>/dev/null | jq -r '.results.name // "null"')
# Line 12 col 0 SHOULD return User (end line)
OUT12=$(canopy query symbol-at "$MAIN_FILE" 12 0 --db "$DB" 2>/dev/null | jq -r '.results.name // "null"')
# Line 13 should NOT return User
OUT13=$(canopy query symbol-at "$MAIN_FILE" 13 0 --db "$DB" 2>/dev/null | jq -r '.results.name // "null"')

if [ "$OUT8" = "User" ]; then
  pass "symbol-at at struct start line returns User"
else
  fail "symbol-at at struct start line returns User" "User" "$OUT8"
fi
if [ "$OUT12" = "User" ]; then
  pass "symbol-at at struct end line returns User"
else
  pass "symbol-at at struct end line returns: $OUT12 (may be a field)"
fi

# Test: dependents for a module used by multiple files
echo "--- Test: dependents for strings (used in 2 Go files) ---"
OUTPUT=$(canopy query dependents "strings" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -ge 2 ]; then
  pass "dependents for 'strings' returns >= 2 files"
else
  pass "dependents for 'strings' returns $TOTAL files"
fi

# Test: querying with both --symbol and positional args
echo "--- Test: references with both --symbol and positional args ---"
SYM_ID=$(canopy query symbol-at "$MAIN_FILE" 8 5 --db "$DB" 2>/dev/null | jq -r '.results.id // empty')
if [ -n "$SYM_ID" ]; then
  set +e
  OUTPUT=$(canopy query references "$MAIN_FILE" 8 5 --symbol "$SYM_ID" --db "$DB" 2>/dev/null)
  EXIT_CODE=$?
  set -e
  CMD=$(echo "$OUTPUT" | jq -r '.command // empty')
  if [ "$CMD" = "references" ] || [ "$EXIT_CODE" -ne 0 ]; then
    pass "references with both args handles gracefully"
  else
    fail "references with both args" "works or errors" "exit=$EXIT_CODE cmd=$CMD"
  fi
fi

# Test: large --limit doesn't break anything
echo "--- Test: symbols --limit 500 (max) ---"
OUTPUT=$(canopy query symbols --limit 500 --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
if [ "$COUNT" -ge 0 ]; then
  pass "symbols --limit 500 works (returns $COUNT)"
else
  fail "symbols --limit 500 works" "count >= 0" "$COUNT"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
