#!/usr/bin/env bash
# Tests for: canopy query implements
# Spec: cli-reference.md — "Find interfaces/traits that a concrete type implements.
#        This is the inverse of implementations."
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Helper: get symbol ID by name and file suffix
get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
UTIL_CLASS_ID=$(get_id "Util" "Util.java")

# Test 1: implements by --symbol for Java class Util
echo "=== Test 1: implements for Java class ==="
OUTPUT=$(canopy query implements --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
CMD="canopy query implements --db $DB --symbol $UTIL_CLASS_ID"

if echo "$OUTPUT" | jq -e '.command == "implements"' > /dev/null 2>&1; then
  echo "PASS: command field is implements"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not implements"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: output same structure as definition (array of locations with symbol_id)
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: implements by positional args
echo ""
echo "=== Test 2: implements by position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/Util.java"
OUTPUT=$(canopy query implements --db "$DB" "$FILE" 0 14 2>&1)
CMD="canopy query implements --db $DB $FILE 0 14"
if echo "$OUTPUT" | jq -e '.command == "implements"' > /dev/null 2>&1; then
  echo "PASS: positional implements works"
  PASS=$((PASS + 1))
else
  echo "FAIL: positional implements failed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: implements for a function (no interfaces) — should return empty array
echo ""
echo "=== Test 3: implements for a non-type symbol ==="
OUTPUT=$(canopy query implements --db "$DB" --symbol "$GREET_ID" 2>&1)
CMD="canopy query implements --db $DB --symbol $GREET_ID"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: implements for function returns array"
  PASS=$((PASS + 1))
else
  echo "FAIL: implements for function did not return array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: implements for non-existent symbol
echo ""
echo "=== Test 4: implements for non-existent symbol ==="
OUTPUT=$(canopy query implements --db "$DB" --symbol 99999 2>&1)
CMD="canopy query implements --db $DB --symbol 99999"
# Spec says results should be array of locations, for non-existent could be empty array or null
if echo "$OUTPUT" | jq -e '.results == null or (.results | type == "array" and length == 0)' > /dev/null 2>&1; then
  echo "PASS: non-existent symbol returns null or empty array"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent symbol returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: implements text format
echo ""
echo "=== Test 5: implements text format ==="
OUTPUT=$(canopy query implements --db "$DB" --format text --symbol "$UTIL_CLASS_ID" 2>&1)
CMD="canopy query implements --db $DB --format text --symbol $UTIL_CLASS_ID"
# Spec says text format for locations: "One per line as file:line:col"
# Output could be empty if no interfaces, but should not error
if ! echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: text format works without error"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format produced error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
