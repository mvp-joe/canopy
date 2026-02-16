#!/bin/bash
# Test: Pagination and sorting across all query types
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Pagination and Sorting ==="

DB="$SCRATCH/go-project/.canopy/index.db"

# Test: symbols --limit + --offset pagination
echo "--- Test: symbols pagination consistency ---"
ALL=$(canopy query symbols --sort name --order asc --db "$DB" 2>/dev/null)
TOTAL=$(echo "$ALL" | jq -r '.total_count')
PAGE1=$(canopy query symbols --sort name --order asc --limit 3 --offset 0 --db "$DB" 2>/dev/null)
PAGE2=$(canopy query symbols --sort name --order asc --limit 3 --offset 3 --db "$DB" 2>/dev/null)
NAME_P1_LAST=$(echo "$PAGE1" | jq -r '.results[-1].name')
NAME_P2_FIRST=$(echo "$PAGE2" | jq -r '.results[0].name')
# Pages should not overlap
if [ "$NAME_P1_LAST" != "$NAME_P2_FIRST" ]; then
  pass "pagination pages don't overlap"
else
  fail "pagination pages don't overlap" "different names" "both=$NAME_P1_LAST"
fi

# Test: symbols sort by name ascending is actually sorted
echo "--- Test: symbols --sort name --order asc ---"
NAMES=$(canopy query symbols --sort name --order asc --limit 5 --db "$DB" 2>/dev/null | jq -r '.results[].name')
SORTED=$(echo "$NAMES" | sort)
if [ "$NAMES" = "$SORTED" ]; then
  pass "symbols sorted by name asc are in alphabetical order"
else
  fail "symbols sorted by name asc are in alphabetical order" "$SORTED" "$NAMES"
fi

# Test: symbols sort by name descending is actually sorted
echo "--- Test: symbols --sort name --order desc ---"
NAMES=$(canopy query symbols --sort name --order desc --limit 5 --db "$DB" 2>/dev/null | jq -r '.results[].name')
SORTED=$(echo "$NAMES" | sort -r)
if [ "$NAMES" = "$SORTED" ]; then
  pass "symbols sorted by name desc are in reverse alphabetical order"
else
  fail "symbols sorted by name desc are in reverse alphabetical order" "$SORTED" "$NAMES"
fi

# Test: symbols sort by ref_count desc should have highest refs first
echo "--- Test: symbols --sort ref_count --order desc ---"
REFS=$(canopy query symbols --sort ref_count --order desc --limit 5 --db "$DB" 2>/dev/null | jq -r '.results[].ref_count')
FIRST_REF=$(echo "$REFS" | head -1)
LAST_REF=$(echo "$REFS" | tail -1)
if [ "$FIRST_REF" -ge "$LAST_REF" ]; then
  pass "symbols sorted by ref_count desc has highest refs first"
else
  fail "symbols sorted by ref_count desc" "first >= last" "first=$FIRST_REF last=$LAST_REF"
fi

# Test: search pagination
echo "--- Test: search pagination ---"
ALL=$(canopy query search "*" --db "$DB" 2>/dev/null)
ALL_TOTAL=$(echo "$ALL" | jq -r '.total_count')
PAGE=$(canopy query search "*" --limit 3 --offset 0 --db "$DB" 2>/dev/null)
PAGE_COUNT=$(echo "$PAGE" | jq '.results | length')
PAGE_TOTAL=$(echo "$PAGE" | jq -r '.total_count')
if [ "$PAGE_COUNT" -le 3 ] && [ "$PAGE_TOTAL" -eq "$ALL_TOTAL" ]; then
  pass "search pagination: limit respected, total_count consistent"
else
  fail "search pagination" "count<=3 total=$ALL_TOTAL" "count=$PAGE_COUNT total=$PAGE_TOTAL"
fi

# Test: files pagination
echo "--- Test: files pagination ---"
PAGE=$(canopy query files --limit 1 --offset 0 --db "$DB" 2>/dev/null)
PAGE_COUNT=$(echo "$PAGE" | jq '.results | length')
PAGE_TOTAL=$(echo "$PAGE" | jq -r '.total_count')
if [ "$PAGE_COUNT" -eq 1 ] && [ "$PAGE_TOTAL" -gt 1 ]; then
  pass "files pagination works"
else
  fail "files pagination works" "count=1, total>1" "count=$PAGE_COUNT, total=$PAGE_TOTAL"
fi

# Test: symbols default limit is 50
echo "--- Test: symbols default limit ---"
# We need a project with more than 50 symbols to test this
# For now, just verify that without --limit, we get results
OUTPUT=$(canopy query symbols --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$COUNT" -le 50 ] && [ "$COUNT" -eq "$TOTAL" ]; then
  pass "symbols default limit: returns up to 50 or all if less"
else
  if [ "$COUNT" -le 50 ]; then
    pass "symbols default limit: result count <= 50"
  else
    fail "symbols default limit is 50" "count <= 50" "count=$COUNT"
  fi
fi

# Test: total_count is independent of limit
echo "--- Test: total_count independent of limit ---"
OUT_ALL=$(canopy query symbols --db "$DB" 2>/dev/null)
OUT_LIM=$(canopy query symbols --limit 1 --db "$DB" 2>/dev/null)
TOTAL_ALL=$(echo "$OUT_ALL" | jq -r '.total_count')
TOTAL_LIM=$(echo "$OUT_LIM" | jq -r '.total_count')
if [ "$TOTAL_ALL" -eq "$TOTAL_LIM" ]; then
  pass "total_count is same regardless of limit"
else
  fail "total_count is same regardless of limit" "$TOTAL_ALL" "$TOTAL_LIM"
fi

# Test: packages sort
echo "--- Test: packages --sort name ---"
OUTPUT=$(canopy query packages --sort name --order asc --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "packages" ]; then
  pass "packages --sort name works"
else
  fail "packages --sort name works" "command=packages" "$CMD"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
