#!/bin/bash
# Test: canopy query search - glob pattern search for symbols
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query search ==="

DB="$SCRATCH/go-project/.canopy/index.db"

# Test: search with prefix wildcard
echo "--- Test: search 'User*' ---"
OUTPUT=$(canopy query search "User*" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "search" ] && [ "$TOTAL" -gt 0 ]; then
  pass "search 'User*' finds symbols starting with User"
else
  fail "search 'User*' finds symbols" "command=search, total>0" "command=$CMD, total=$TOTAL"
fi

# Test: search result has same fields as symbols
echo "--- Test: search result fields ---"
FIRST=$(echo "$OUTPUT" | jq '.results[0]')
for field in id name kind visibility file start_line start_col end_line end_col; do
  HAS=$(echo "$FIRST" | jq "has(\"$field\")")
  if [ "$HAS" != "true" ]; then
    fail "search result has '$field'" "true" "$HAS"
  fi
done
pass "search results have symbol fields"

# Test: search with contains wildcard
echo "--- Test: search '*User*' ---"
OUTPUT=$(canopy query search "*User*" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "search '*User*' finds results"
else
  fail "search '*User*' finds results" "total > 0" "total=$TOTAL"
fi

# Test: search with suffix wildcard
echo "--- Test: search '*Service' ---"
OUTPUT=$(canopy query search "*Service" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  # Verify results end with Service
  NAMES=$(echo "$OUTPUT" | jq -r '.results[].name')
  ALL_MATCH=true
  while IFS= read -r name; do
    if [[ ! "$name" == *Service ]]; then
      ALL_MATCH=false
    fi
  done <<< "$NAMES"
  if [ "$ALL_MATCH" = true ]; then
    pass "search '*Service' returns only names ending with Service"
  else
    fail "search '*Service' returns only names ending with Service" "all end with Service" "$NAMES"
  fi
else
  fail "search '*Service' finds results" "total > 0" "total=$TOTAL"
fi

# Test: search with no results
echo "--- Test: search for non-existent pattern ---"
OUTPUT=$(canopy query search "ZZZZZ_nonexistent*" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ]; then
  pass "search for non-existent pattern returns 0 results"
else
  fail "search for non-existent pattern returns 0 results" "total=0" "total=$TOTAL"
fi

# Test: search with pagination
echo "--- Test: search with --limit ---"
OUTPUT=$(canopy query search "*" --limit 2 --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
if [ "$COUNT" -le 2 ]; then
  pass "search '*' --limit 2 returns at most 2 results"
else
  fail "search '*' --limit 2 returns at most 2 results" "count<=2" "count=$COUNT"
fi

# Test: search text format
echo "--- Test: search --format text ---"
OUTPUT_TEXT=$(canopy query search "User*" --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT_TEXT" | grep -qi "user"; then
  pass "search text format shows matching symbols"
else
  fail "search text format shows matching symbols" "contains user" "$OUTPUT_TEXT"
fi

# Test: search with no pattern (should error - exactly 1 arg required)
echo "--- Test: search with no pattern ---"
set +e
OUTPUT=$(canopy query search --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "search with no pattern returns error"
else
  fail "search with no pattern returns error" "non-zero exit" "exit=$EXIT_CODE"
fi

# Test: search in TypeScript project
echo "--- Test: search in TypeScript ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
OUTPUT=$(canopy query search "*Repository*" --db "$TS_DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "search finds Repository in TypeScript project"
else
  fail "search finds Repository in TypeScript project" "total > 0" "total=$TOTAL"
fi

# Test: search sort order
echo "--- Test: search with --sort name ---"
OUTPUT=$(canopy query search "*" --sort name --order asc --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "search" ]; then
  pass "search with --sort name works"
else
  fail "search with --sort name works" "command=search" "$CMD"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
