#!/usr/bin/env bash
# Tests for: --ref-count-min and --ref-count-max flags on symbols and search
# Spec: cli-reference.md — symbols and search commands
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: symbols with --ref-count-min 1 (only symbols with >= 1 references)
echo "=== Test 1: symbols --ref-count-min 1 ==="
OUTPUT=$(canopy query symbols --db "$DB" --ref-count-min 1 2>&1)
CMD="canopy query symbols --db $DB --ref-count-min 1"

if echo "$OUTPUT" | jq -e '.command == "symbols"' > /dev/null 2>&1; then
  echo "PASS: command field is symbols"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not symbols"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# All results should have ref_count >= 1
BELOW_MIN=$(echo "$OUTPUT" | jq '[.results[] | select(.ref_count < 1)] | length')
if [ "$BELOW_MIN" = "0" ]; then
  echo "PASS: all symbols have ref_count >= 1"
  PASS=$((PASS + 1))
else
  echo "FAIL: $BELOW_MIN symbols have ref_count < 1"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Should have fewer results than unfiltered
FILTERED_COUNT=$(echo "$OUTPUT" | jq '.total_count')
TOTAL_OUTPUT=$(canopy query symbols --db "$DB" --limit 500 2>&1)
TOTAL_COUNT=$(echo "$TOTAL_OUTPUT" | jq '.total_count')
if [ "$FILTERED_COUNT" -lt "$TOTAL_COUNT" ]; then
  echo "PASS: filtered count ($FILTERED_COUNT) < total count ($TOTAL_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: filtered count ($FILTERED_COUNT) not less than total ($TOTAL_COUNT)"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# Test 2: symbols with --ref-count-max 0 (only unreferenced symbols)
echo ""
echo "=== Test 2: symbols --ref-count-max 0 ==="
OUTPUT=$(canopy query symbols --db "$DB" --ref-count-max 0 2>&1)
CMD="canopy query symbols --db $DB --ref-count-max 0"
ABOVE_MAX=$(echo "$OUTPUT" | jq '[.results[] | select(.ref_count > 0)] | length')
if [ "$ABOVE_MAX" = "0" ]; then
  echo "PASS: all symbols have ref_count <= 0"
  PASS=$((PASS + 1))
else
  echo "FAIL: $ABOVE_MAX symbols have ref_count > 0"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: symbols with both --ref-count-min and --ref-count-max
echo ""
echo "=== Test 3: symbols --ref-count-min 1 --ref-count-max 1 ==="
OUTPUT=$(canopy query symbols --db "$DB" --ref-count-min 1 --ref-count-max 1 2>&1)
CMD="canopy query symbols --db $DB --ref-count-min 1 --ref-count-max 1"
# All should have exactly ref_count == 1
OUTSIDE_RANGE=$(echo "$OUTPUT" | jq '[.results[] | select(.ref_count < 1 or .ref_count > 1)] | length')
if [ "$OUTSIDE_RANGE" = "0" ]; then
  echo "PASS: all symbols have ref_count == 1"
  PASS=$((PASS + 1))
else
  echo "FAIL: $OUTSIDE_RANGE symbols outside range"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: search with --ref-count-min
echo ""
echo "=== Test 4: search with --ref-count-min ==="
OUTPUT=$(canopy query search --db "$DB" "*add*" --ref-count-min 1 2>&1)
CMD="canopy query search --db $DB '*add*' --ref-count-min 1"
if echo "$OUTPUT" | jq -e '.command == "search"' > /dev/null 2>&1; then
  echo "PASS: search command returns search format"
  PASS=$((PASS + 1))
else
  echo "FAIL: search command format unexpected (expected command=search)"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

BELOW_MIN=$(echo "$OUTPUT" | jq '[.results[] | select(.ref_count < 1)] | length')
if [ "$BELOW_MIN" = "0" ]; then
  echo "PASS: search --ref-count-min filters correctly"
  PASS=$((PASS + 1))
else
  echo "FAIL: search --ref-count-min didn't filter: $BELOW_MIN below min"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: search with --ref-count-max
echo ""
echo "=== Test 5: search with --ref-count-max ==="
OUTPUT=$(canopy query search --db "$DB" "*" --ref-count-max 0 2>&1)
CMD="canopy query search --db $DB '*' --ref-count-max 0"
ABOVE_MAX=$(echo "$OUTPUT" | jq '[.results[] | select(.ref_count > 0)] | length')
if [ "$ABOVE_MAX" = "0" ]; then
  echo "PASS: search --ref-count-max 0 filters correctly"
  PASS=$((PASS + 1))
else
  echo "FAIL: search --ref-count-max 0 didn't filter: $ABOVE_MAX above max"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: symbols with --ref-count-min > --ref-count-max should return empty
echo ""
echo "=== Test 6: symbols with impossible range (min > max) ==="
OUTPUT=$(canopy query symbols --db "$DB" --ref-count-min 10 --ref-count-max 1 2>&1)
CMD="canopy query symbols --db $DB --ref-count-min 10 --ref-count-max 1"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
if [ "$RESULT_LEN" = "0" ]; then
  echo "PASS: impossible range returns empty results"
  PASS=$((PASS + 1))
else
  echo "FAIL: impossible range returned $RESULT_LEN results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
