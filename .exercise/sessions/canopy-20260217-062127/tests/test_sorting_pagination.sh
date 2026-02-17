#!/usr/bin/env bash
# Tests for sorting and pagination on new commands (unused, hotspots)
# Spec: cli-reference.md shared pagination/sort flags
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: unused --sort name --order asc
echo "=== Test 1: unused --sort name --order asc ==="
OUTPUT=$(canopy query unused --db "$DB" --sort name --order asc 2>&1)
CMD="canopy query unused --db $DB --sort name --order asc"
NAMES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
SORTED_NAMES=$(echo "$NAMES" | sort)
if [ "$NAMES" = "$SORTED_NAMES" ]; then
  echo "PASS: unused sorted by name ascending"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused not sorted by name ascending"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# Test 2: unused --sort name --order desc
echo ""
echo "=== Test 2: unused --sort name --order desc ==="
OUTPUT=$(canopy query unused --db "$DB" --sort name --order desc 2>&1)
CMD="canopy query unused --db $DB --sort name --order desc"
NAMES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
SORTED_NAMES=$(echo "$NAMES" | sort -r)
if [ "$NAMES" = "$SORTED_NAMES" ]; then
  echo "PASS: unused sorted by name descending"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused not sorted by name descending"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# Test 3: unused --sort kind
echo ""
echo "=== Test 3: unused --sort kind ==="
OUTPUT=$(canopy query unused --db "$DB" --sort kind --order asc 2>&1)
CMD="canopy query unused --db $DB --sort kind --order asc"
KINDS=$(echo "$OUTPUT" | jq -r '[.results[].kind] | .[]')
SORTED_KINDS=$(echo "$KINDS" | sort)
if [ "$KINDS" = "$SORTED_KINDS" ]; then
  echo "PASS: unused sorted by kind ascending"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused not sorted by kind ascending"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# Test 4: unused pagination with offset
echo ""
echo "=== Test 4: unused pagination with offset ==="
PAGE1=$(canopy query unused --db "$DB" --limit 3 --offset 0 2>&1)
PAGE2=$(canopy query unused --db "$DB" --limit 3 --offset 3 2>&1)
CMD="canopy query unused --db $DB --limit 3 --offset 0/3"
P1_IDS=$(echo "$PAGE1" | jq -r '[.results[].id] | .[]')
P2_IDS=$(echo "$PAGE2" | jq -r '[.results[].id] | .[]')
# Pages should not overlap
OVERLAP=0
for id in $P1_IDS; do
  if echo "$P2_IDS" | grep -q "^${id}$"; then
    OVERLAP=$((OVERLAP + 1))
  fi
done
if [ "$OVERLAP" -eq 0 ]; then
  echo "PASS: page 1 and page 2 have no overlapping IDs"
  PASS=$((PASS + 1))
else
  echo "FAIL: $OVERLAP IDs overlap between pages"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# total_count should be the same across pages
P1_TOTAL=$(echo "$PAGE1" | jq '.total_count')
P2_TOTAL=$(echo "$PAGE2" | jq '.total_count')
if [ "$P1_TOTAL" = "$P2_TOTAL" ]; then
  echo "PASS: total_count consistent across pages ($P1_TOTAL)"
  PASS=$((PASS + 1))
else
  echo "FAIL: total_count differs: page1=$P1_TOTAL, page2=$P2_TOTAL"
  FAIL=$((FAIL + 1))
fi

# Test 5: unused --limit 0 returns empty or error
echo ""
echo "=== Test 5: unused --limit 0 ==="
OUTPUT=$(canopy query unused --db "$DB" --limit 0 2>&1) || true
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length' 2>/dev/null || echo "-1")
# Spec says limit range is [0,500]; --limit 0 should logically return 0 results
if [ "$RESULT_LEN" = "0" ]; then
  echo "PASS: --limit 0 returns empty results"
  PASS=$((PASS + 1))
else
  echo "FAIL: --limit 0 returned $RESULT_LEN results (expected 0 per pagination semantics)"
  echo "  Spec: --limit int, default 50, max 500"
  FAIL=$((FAIL + 1))
fi

# Test 6: unused --sort file
echo ""
echo "=== Test 6: unused --sort file ==="
OUTPUT=$(canopy query unused --db "$DB" --sort file --order asc 2>&1)
CMD="canopy query unused --db $DB --sort file --order asc"
FILES=$(echo "$OUTPUT" | jq -r '[.results[].file] | .[]')
SORTED_FILES=$(echo "$FILES" | sort)
if [ "$FILES" = "$SORTED_FILES" ]; then
  echo "PASS: unused sorted by file ascending"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused not sorted by file ascending"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# Test 7: hotspots --top with large value
echo ""
echo "=== Test 7: hotspots --top 100 ==="
OUTPUT=$(canopy query hotspots --db "$DB" --top 100 2>&1)
CMD="canopy query hotspots --db $DB --top 100"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
TOTAL=$(echo "$OUTPUT" | jq '.total_count')
if [ "$RESULT_LEN" -le 100 ]; then
  echo "PASS: --top 100 returns <= 100 results (got $RESULT_LEN, total=$TOTAL)"
  PASS=$((PASS + 1))
else
  echo "FAIL: --top 100 returned $RESULT_LEN results"
  FAIL=$((FAIL + 1))
fi

# Test 8: unused --limit 500 (max per spec)
echo ""
echo "=== Test 8: unused --limit 500 ==="
OUTPUT=$(canopy query unused --db "$DB" --limit 500 2>&1)
CMD="canopy query unused --db $DB --limit 500"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
if [ "$RESULT_LEN" -le 500 ]; then
  echo "PASS: --limit 500 returns <= 500 results (got $RESULT_LEN)"
  PASS=$((PASS + 1))
else
  echo "FAIL: --limit 500 returned $RESULT_LEN results"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
