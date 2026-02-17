#!/usr/bin/env bash
# Tests for: canopy query hotspots
# Spec: cli-reference.md — "Show the most-referenced symbols with fan-in and fan-out
#        call graph metrics. Sorted by external reference count descending."
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: hotspots basic output
echo "=== Test 1: hotspots basic output ==="
OUTPUT=$(canopy query hotspots --db "$DB" 2>&1)
CMD="canopy query hotspots --db $DB"

if echo "$OUTPUT" | jq -e '.command == "hotspots"' > /dev/null 2>&1; then
  echo "PASS: command field is hotspots"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not hotspots"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# results should be an array
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Each result should have symbol, caller_count, callee_count
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results[0] | has("symbol", "caller_count", "callee_count")' > /dev/null 2>&1; then
    echo "PASS: hotspot has required fields (symbol, caller_count, callee_count)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: hotspot missing required fields"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi

  # symbol should have full symbol info
  if echo "$OUTPUT" | jq -e '.results[0].symbol | has("id", "name", "kind", "ref_count", "external_ref_count", "internal_ref_count")' > /dev/null 2>&1; then
    echo "PASS: hotspot symbol has full info"
    PASS=$((PASS + 1))
  else
    echo "FAIL: hotspot symbol missing fields"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi

  # Spec: sorted by external_ref_count descending
  SORTED=$(echo "$OUTPUT" | jq '[.results[].symbol.external_ref_count] | . as $arr | [range(0;length-1)] | all(. as $i | $arr[$i] >= $arr[$i+1])')
  if [ "$SORTED" = "true" ]; then
    echo "PASS: sorted by external_ref_count descending"
    PASS=$((PASS + 1))
  else
    echo "FAIL: not sorted by external_ref_count descending"
    echo "  Command: $CMD"
    echo "  Ref counts: $(echo "$OUTPUT" | jq '[.results[].symbol.external_ref_count]')"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no hotspots found"
fi

# Test 2: hotspots with --top flag
echo ""
echo "=== Test 2: hotspots --top 3 ==="
OUTPUT=$(canopy query hotspots --db "$DB" --top 3 2>&1)
CMD="canopy query hotspots --db $DB --top 3"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
if [ "$RESULT_LEN" -le 3 ]; then
  echo "PASS: --top 3 returns <= 3 results (got $RESULT_LEN)"
  PASS=$((PASS + 1))
else
  echo "FAIL: --top 3 returned $RESULT_LEN results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: hotspots default --top 10
echo ""
echo "=== Test 3: hotspots default --top ==="
OUTPUT=$(canopy query hotspots --db "$DB" 2>&1)
CMD="canopy query hotspots --db $DB"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
if [ "$RESULT_LEN" -le 10 ]; then
  echo "PASS: default returns <= 10 results (got $RESULT_LEN)"
  PASS=$((PASS + 1))
else
  echo "FAIL: default returned $RESULT_LEN results (expected <= 10)"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: hotspots --top 1 returns exactly 1
echo ""
echo "=== Test 4: hotspots --top 1 ==="
OUTPUT=$(canopy query hotspots --db "$DB" --top 1 2>&1)
CMD="canopy query hotspots --db $DB --top 1"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
if [ "$RESULT_LEN" -eq 1 ]; then
  echo "PASS: --top 1 returns exactly 1 result"
  PASS=$((PASS + 1))
else
  echo "FAIL: --top 1 returned $RESULT_LEN results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: hotspots text format
echo ""
echo "=== Test 5: hotspots text format ==="
OUTPUT=$(canopy query hotspots --db "$DB" --format text 2>&1)
CMD="canopy query hotspots --db $DB --format text"
if [ -n "$OUTPUT" ]; then
  echo "PASS: text format produces output"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format produces no output"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
