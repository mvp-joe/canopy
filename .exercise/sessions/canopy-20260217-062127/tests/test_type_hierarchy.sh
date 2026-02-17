#!/usr/bin/env bash
# Tests for: canopy query type-hierarchy
# Spec: cli-reference.md "Type Hierarchy Commands" section
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Helper: get symbol ID by name and file suffix
get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
UTIL_CLASS_ID=$(get_id "Util" "Util.java")

# Test 1: type-hierarchy by --symbol for Java class Util
echo "=== Test 1: type-hierarchy for Java class ==="
OUTPUT=$(canopy query type-hierarchy --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
CMD="canopy query type-hierarchy --db $DB --symbol $UTIL_CLASS_ID"

if echo "$OUTPUT" | jq -e '.command == "type-hierarchy"' > /dev/null 2>&1; then
  echo "PASS: command field is type-hierarchy"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not type-hierarchy"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: results should have symbol, implements, implemented_by, composes, composed_by, extensions
if echo "$OUTPUT" | jq -e '.results | has("symbol", "implements", "implemented_by", "composes", "composed_by", "extensions")' > /dev/null 2>&1; then
  echo "PASS: results has all required fields"
  PASS=$((PASS + 1))
else
  echo "FAIL: results missing required fields"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# symbol should match
if echo "$OUTPUT" | jq -e '.results.symbol.name == "Util"' > /dev/null 2>&1; then
  echo "PASS: symbol name is Util"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol name is not Util"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Arrays should be arrays
for field in implements implemented_by composes composed_by extensions; do
  if echo "$OUTPUT" | jq -e ".results.${field} | type == \"array\"" > /dev/null 2>&1; then
    echo "PASS: $field is an array"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $field is not an array"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
done

# total_count should be 1
if echo "$OUTPUT" | jq -e '.total_count == 1' > /dev/null 2>&1; then
  echo "PASS: total_count is 1"
  PASS=$((PASS + 1))
else
  echo "FAIL: total_count is not 1"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: type-hierarchy by positional args
echo ""
echo "=== Test 2: type-hierarchy by position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/Util.java"
OUTPUT=$(canopy query type-hierarchy --db "$DB" "$FILE" 0 14 2>&1)
CMD="canopy query type-hierarchy --db $DB $FILE 0 14"
if echo "$OUTPUT" | jq -e '.results.symbol.name == "Util"' > /dev/null 2>&1; then
  echo "PASS: positional lookup finds Util"
  PASS=$((PASS + 1))
else
  echo "FAIL: positional lookup didn't find Util"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: type-hierarchy for non-existent symbol returns null
echo ""
echo "=== Test 3: type-hierarchy for non-existent symbol ==="
OUTPUT=$(canopy query type-hierarchy --db "$DB" --symbol 99999 2>&1)
CMD="canopy query type-hierarchy --db $DB --symbol 99999"
if echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: non-existent symbol returns null"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent symbol did not return null"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: type-hierarchy for a function (non-type) - should still work but have empty arrays
echo ""
echo "=== Test 4: type-hierarchy for a function ==="
OUTPUT=$(canopy query type-hierarchy --db "$DB" --symbol "$GREET_ID" 2>&1)
CMD="canopy query type-hierarchy --db $DB --symbol $GREET_ID"
if echo "$OUTPUT" | jq -e '.results != null' > /dev/null 2>&1; then
  echo "PASS: type-hierarchy works for function symbol"
  PASS=$((PASS + 1))
else
  echo "FAIL: type-hierarchy failed for function symbol"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: type-hierarchy text format
echo ""
echo "=== Test 5: type-hierarchy text format ==="
OUTPUT=$(canopy query type-hierarchy --db "$DB" --format text --symbol "$UTIL_CLASS_ID" 2>&1)
CMD="canopy query type-hierarchy --db $DB --format text --symbol $UTIL_CLASS_ID"
if echo "$OUTPUT" | grep -qi "Util"; then
  echo "PASS: text format contains symbol name"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format doesn't contain symbol name"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
