#!/usr/bin/env bash
# Tests for: canopy query symbol-detail
# Spec: cli-reference.md "Symbol Detail & Structural Queries" section
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Helper: get symbol ID by name and file suffix
get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
MAIN_GO_ID=$(get_id "main" "main.go" | head -1)
UTIL_CLASS_ID=$(get_id "Util" "Util.java")
GREET_METHOD_ID=$(get_id "greet" "Util.java")

# Test 1: symbol-detail by --symbol flag for a function (Go Greet)
echo "=== Test 1: symbol-detail by --symbol (function) ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_ID" 2>&1)
CMD="canopy query symbol-detail --db $DB --symbol $GREET_ID"
if echo "$OUTPUT" | jq -e '.command == "symbol-detail"' > /dev/null 2>&1; then
  echo "PASS: command field is symbol-detail"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not symbol-detail"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

if echo "$OUTPUT" | jq -e '.results.symbol.name == "Greet"' > /dev/null 2>&1; then
  echo "PASS: symbol name is Greet"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol name is not Greet"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

if echo "$OUTPUT" | jq -e '.results.symbol.kind == "function"' > /dev/null 2>&1; then
  echo "PASS: symbol kind is function"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol kind is not function"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

for field in parameters members type_params annotations; do
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

if echo "$OUTPUT" | jq -e '.results.symbol | has("ref_count", "external_ref_count", "internal_ref_count")' > /dev/null 2>&1; then
  echo "PASS: symbol has ref_count fields"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol missing ref_count fields"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

if echo "$OUTPUT" | jq -e '.total_count == 1' > /dev/null 2>&1; then
  echo "PASS: total_count is 1"
  PASS=$((PASS + 1))
else
  echo "FAIL: total_count is not 1"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: symbol-detail by positional args
echo ""
echo "=== Test 2: symbol-detail by position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query symbol-detail --db "$DB" "$FILE" 2 5 2>&1)
CMD="canopy query symbol-detail --db $DB $FILE 2 5"
if echo "$OUTPUT" | jq -e '.results.symbol.name == "main"' > /dev/null 2>&1; then
  echo "PASS: positional lookup finds main"
  PASS=$((PASS + 1))
else
  echo "FAIL: positional lookup didn't find main"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: symbol-detail for a class (Java Util)
echo ""
echo "=== Test 3: symbol-detail for a class ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
CMD="canopy query symbol-detail --db $DB --symbol $UTIL_CLASS_ID"
if echo "$OUTPUT" | jq -e '.results.symbol.kind == "class"' > /dev/null 2>&1; then
  echo "PASS: Util is a class"
  PASS=$((PASS + 1))
else
  echo "FAIL: Util kind is not class"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

if echo "$OUTPUT" | jq -e '.results.members | length > 0' > /dev/null 2>&1; then
  echo "PASS: class has members"
  PASS=$((PASS + 1))
else
  echo "FAIL: class has no members"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: symbol-detail for a method
echo ""
echo "=== Test 4: symbol-detail for a method ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_METHOD_ID" 2>&1)
CMD="canopy query symbol-detail --db $DB --symbol $GREET_METHOD_ID"
if echo "$OUTPUT" | jq -e '.results.symbol.kind == "method"' > /dev/null 2>&1; then
  echo "PASS: greet is a method"
  PASS=$((PASS + 1))
else
  echo "FAIL: greet kind is not method"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: symbol-detail for non-existent symbol returns null
echo ""
echo "=== Test 5: symbol-detail for non-existent symbol ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol 99999 2>&1)
CMD="canopy query symbol-detail --db $DB --symbol 99999"
if echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: non-existent symbol returns null results"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent symbol did not return null results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: symbol-detail text format
echo ""
echo "=== Test 6: symbol-detail text format ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --format text --symbol "$GREET_ID" 2>&1)
CMD="canopy query symbol-detail --db $DB --format text --symbol $GREET_ID"
if echo "$OUTPUT" | grep -qi "Greet"; then
  echo "PASS: text format contains symbol name"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format does not contain symbol name"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
