#!/usr/bin/env bash
# Tests for: canopy query scope-at
# Spec: cli-reference.md "scope-at" section
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: scope-at inside a function (Go main.go, line 3 col 5 = inside main func)
echo "=== Test 1: scope-at inside a Go function ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 3 5 2>&1)
CMD="canopy query scope-at --db $DB $FILE 3 5"

if echo "$OUTPUT" | jq -e '.command == "scope-at"' > /dev/null 2>&1; then
  echo "PASS: command field is scope-at"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not scope-at"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Should return an array ordered from innermost to outermost
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Should have at least one scope
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  echo "PASS: has at least one scope"
  PASS=$((PASS + 1))
else
  echo "FAIL: no scopes found"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Each scope should have required fields: id, kind, start_line, start_col, end_line, end_col
if echo "$OUTPUT" | jq -e '.results[0] | has("id", "kind", "start_line", "start_col", "end_line", "end_col")' > /dev/null 2>&1; then
  echo "PASS: scope has required fields"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope missing required fields"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Should have a file-level scope (outermost)
if echo "$OUTPUT" | jq -e '.results[-1].kind == "file"' > /dev/null 2>&1; then
  echo "PASS: outermost scope is file"
  PASS=$((PASS + 1))
else
  echo "FAIL: outermost scope is not file"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# total_count should match results length
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
TOTAL_COUNT=$(echo "$OUTPUT" | jq '.total_count')
if [ "$RESULT_LEN" = "$TOTAL_COUNT" ]; then
  echo "PASS: total_count matches results length ($TOTAL_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: total_count ($TOTAL_COUNT) does not match results length ($RESULT_LEN)"
  echo "  Command: $CMD"
  FAIL=$((FAIL + 1))
fi

# Test 2: scope-at for Java method (inside a class method)
echo ""
echo "=== Test 2: scope-at inside a Java method ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/Util.java"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 2 8 2>&1)
CMD="canopy query scope-at --db $DB $FILE 2 8"
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  echo "PASS: found scopes in Java method"
  PASS=$((PASS + 1))
else
  echo "FAIL: no scopes found in Java method"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: scope-at outside any function (file level only)
echo ""
echo "=== Test 3: scope-at at file level ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/util.go"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 0 0 2>&1)
CMD="canopy query scope-at --db $DB $FILE 0 0"
# At file level, we should get at least one scope (the file scope)
if echo "$OUTPUT" | jq -e '.results != null' > /dev/null 2>&1; then
  echo "PASS: scope-at at file level returns results"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope-at at file level returned null"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: scope-at with non-existent file
echo ""
echo "=== Test 4: scope-at with non-existent file ==="
OUTPUT=$(canopy query scope-at --db "$DB" "/nonexistent/file.go" 0 0 2>&1)
CMD="canopy query scope-at --db $DB /nonexistent/file.go 0 0"
# Should return null results, empty array, or error
if echo "$OUTPUT" | jq -e '.results == null or (.results | type == "array" and length == 0)' > /dev/null 2>&1 || echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1; then
  echo "PASS: non-existent file returns null/empty or error"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent file returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: scope-at text format
echo ""
echo "=== Test 5: scope-at text format ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query scope-at --db "$DB" --format text "$FILE" 3 5 2>&1)
CMD="canopy query scope-at --db $DB --format text $FILE 3 5"
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
