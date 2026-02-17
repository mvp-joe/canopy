#!/usr/bin/env bash
# Tests for: canopy query extensions
# Spec: cli-reference.md — "Returns extension bindings for a type"
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Helper: get symbol ID by name and file suffix
get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
UTIL_CLASS_ID=$(get_id "Util" "Util.java")

# Test 1: extensions by --symbol for Java class Util
echo "=== Test 1: extensions for Java class ==="
OUTPUT=$(canopy query extensions --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
CMD="canopy query extensions --db $DB --symbol $UTIL_CLASS_ID"

if echo "$OUTPUT" | jq -e '.command == "extensions"' > /dev/null 2>&1; then
  echo "PASS: command field is extensions"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not extensions"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: results is array of extension bindings
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: each should have type_symbol_id, member_symbol_id, kind, source_file_id
# BUG CHECK: implementation may use different field names
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results[0] | has("type_symbol_id", "member_symbol_id", "kind", "source_file_id")' > /dev/null 2>&1; then
    echo "PASS: extension entry has spec-defined fields (type_symbol_id, member_symbol_id, kind, source_file_id)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: extension entry missing spec-defined fields. Spec requires: type_symbol_id, member_symbol_id, kind, source_file_id"
    echo "  Command: $CMD"
    echo "  Actual fields: $(echo "$OUTPUT" | jq '.results[0] | keys')"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no extensions found (empty array is valid for this type)"
fi

# Test 2: extensions by positional args
echo ""
echo "=== Test 2: extensions by position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/Util.java"
OUTPUT=$(canopy query extensions --db "$DB" "$FILE" 0 14 2>&1)
CMD="canopy query extensions --db $DB $FILE 0 14"
if echo "$OUTPUT" | jq -e '.command == "extensions"' > /dev/null 2>&1; then
  echo "PASS: positional extensions works"
  PASS=$((PASS + 1))
else
  echo "FAIL: positional extensions failed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: extensions for a non-type (function) — should return empty
echo ""
echo "=== Test 3: extensions for a function ==="
OUTPUT=$(canopy query extensions --db "$DB" --symbol "$GREET_ID" 2>&1)
CMD="canopy query extensions --db $DB --symbol $GREET_ID"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: extensions for function returns array"
  PASS=$((PASS + 1))
else
  echo "FAIL: extensions for function did not return array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: extensions for non-existent symbol
echo ""
echo "=== Test 4: extensions for non-existent symbol ==="
OUTPUT=$(canopy query extensions --db "$DB" --symbol 99999 2>&1)
CMD="canopy query extensions --db $DB --symbol 99999"
if echo "$OUTPUT" | jq -e '.results | type == "array" and length == 0' > /dev/null 2>&1 || echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: non-existent symbol returns empty or null"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent symbol returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
