#!/usr/bin/env bash
# Tests for: canopy query circular-deps
# Spec: cli-reference.md — "Detect circular dependencies in the package dependency graph.
#        Uses Tarjan's SCC algorithm."
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: circular-deps basic structure
echo "=== Test 1: circular-deps basic structure ==="
OUTPUT=$(canopy query circular-deps --db "$DB" 2>&1)
CMD="canopy query circular-deps --db $DB"

if echo "$OUTPUT" | jq -e '.command == "circular-deps"' > /dev/null 2>&1; then
  echo "PASS: command field is circular-deps"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not circular-deps"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: results should be an array (empty if no cycles)
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# If there are results, each should have a "cycle" field which is an array
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results[0].cycle | type == "array"' > /dev/null 2>&1; then
    echo "PASS: cycle is an array"
    PASS=$((PASS + 1))
  else
    echo "FAIL: cycle is not an array"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
  # A cycle should start and end with the same package
  FIRST=$(echo "$OUTPUT" | jq -r '.results[0].cycle[0]')
  LAST=$(echo "$OUTPUT" | jq -r '.results[0].cycle[-1]')
  if [ "$FIRST" = "$LAST" ]; then
    echo "PASS: cycle starts and ends with same package"
    PASS=$((PASS + 1))
  else
    echo "FAIL: cycle does not start and end with same package (first=$FIRST, last=$LAST)"
    echo "  Command: $CMD"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no circular dependencies found (empty array)"
fi

# total_count should be a number
if echo "$OUTPUT" | jq -e '.total_count | type == "number"' > /dev/null 2>&1; then
  echo "PASS: total_count is a number"
  PASS=$((PASS + 1))
else
  echo "FAIL: total_count is not a number"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: circular-deps text format
echo ""
echo "=== Test 2: circular-deps text format ==="
OUTPUT=$(canopy query circular-deps --db "$DB" --format text 2>&1)
CMD="canopy query circular-deps --db $DB --format text"
# Should work without errors (might produce no output if no cycles)
EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 0 ]; then
  echo "PASS: text format works"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format errored"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
