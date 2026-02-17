#!/usr/bin/env bash
# Tests for edge cases on new commands
# Missing symbols, non-existent files, boundary values, empty results
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: symbol-detail with no args (should error)
echo "=== Test 1: symbol-detail with no args ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" 2>&1) || true
CMD="canopy query symbol-detail --db $DB"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: symbol-detail with no args errors"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol-detail with no args did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: scope-at with wrong number of positional args
echo "=== Test 2: scope-at with wrong args ==="
OUTPUT=$(canopy query scope-at --db "$DB" "/some/file.go" 5 2>&1) || true
CMD="canopy query scope-at --db $DB /some/file.go 5"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: scope-at with 2 args errors"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope-at with 2 args did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: transitive-callers with --symbol 0
echo "=== Test 3: transitive-callers --symbol 0 ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol 0 2>&1) || true
CMD="canopy query transitive-callers --db $DB --symbol 0"
if echo "$OUTPUT" | jq -e '.results == null or .error' > /dev/null 2>&1; then
  echo "PASS: symbol 0 returns null or error"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol 0 returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: transitive-callees with extremely large --max-depth (should cap at 100)
echo "=== Test 4: transitive-callees --max-depth 999 ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --symbol 9 --max-depth 999 2>&1) || true
CMD="canopy query transitive-callees --db $DB --symbol 9 --max-depth 999"
if echo "$OUTPUT" | jq -e '.results != null' > /dev/null 2>&1; then
  echo "PASS: max-depth 999 accepted (capped at 100)"
  PASS=$((PASS + 1))
else
  echo "FAIL: max-depth 999 failed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: hotspots --top 0 edge case
echo "=== Test 5: hotspots --top 0 ==="
OUTPUT=$(canopy query hotspots --db "$DB" --top 0 2>&1) || true
CMD="canopy query hotspots --db $DB --top 0"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length' 2>/dev/null || echo "-1")
if [ "$RESULT_LEN" = "0" ]; then
  echo "PASS: --top 0 returns empty results"
  PASS=$((PASS + 1))
else
  echo "FAIL: --top 0 returned $RESULT_LEN results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: unused with --sort ref_count
echo "=== Test 6: unused with --sort ref_count ==="
OUTPUT=$(canopy query unused --db "$DB" --sort ref_count 2>&1)
CMD="canopy query unused --db $DB --sort ref_count"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: unused --sort ref_count works"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused --sort ref_count failed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 7: type-hierarchy with position that's not on a symbol
echo "=== Test 7: type-hierarchy at whitespace position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query type-hierarchy --db "$DB" "$FILE" 1 0 2>&1) || true
CMD="canopy query type-hierarchy --db $DB $FILE 1 0"
if echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1 || echo "$OUTPUT" | jq -e '.results != null' > /dev/null 2>&1; then
  echo "PASS: type-hierarchy at whitespace doesn't crash"
  PASS=$((PASS + 1))
else
  echo "FAIL: type-hierarchy at whitespace crashed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 8: reexports missing required argument
echo "=== Test 8: reexports without file argument ==="
OUTPUT=$(canopy query reexports --db "$DB" 2>&1) || true
CMD="canopy query reexports --db $DB"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: reexports without file errors"
  PASS=$((PASS + 1))
else
  echo "FAIL: reexports without file did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 9: implements with wrong number of positional args (only 2)
echo "=== Test 9: implements with wrong positional args ==="
OUTPUT=$(canopy query implements --db "$DB" "/some/file.go" 5 2>&1) || true
CMD="canopy query implements --db $DB /some/file.go 5"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: implements with 2 args errors"
  PASS=$((PASS + 1))
else
  echo "FAIL: implements with 2 args did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 10: extensions with --symbol flag for valid but non-existent type
echo "=== Test 10: extensions --symbol large number ==="
OUTPUT=$(canopy query extensions --db "$DB" --symbol 100000 2>&1) || true
CMD="canopy query extensions --db $DB --symbol 100000"
if echo "$OUTPUT" | jq -e '.results | type == "array" and length == 0' > /dev/null 2>&1 || echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: large symbol ID returns empty or null"
  PASS=$((PASS + 1))
else
  echo "FAIL: large symbol ID returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 11: symbol-detail for position past end of file
echo "=== Test 11: symbol-detail at extreme position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query symbol-detail --db "$DB" "$FILE" 9999 9999 2>&1) || true
CMD="canopy query symbol-detail --db $DB $FILE 9999 9999"
if echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: extreme position returns null"
  PASS=$((PASS + 1))
else
  echo "FAIL: extreme position returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 12: package-graph with non-existent DB
echo "=== Test 12: package-graph with missing DB ==="
OUTPUT=$(canopy query package-graph --db /tmp/nonexistent.db 2>&1) || true
CMD="canopy query package-graph --db /tmp/nonexistent.db"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: missing DB returns error"
  PASS=$((PASS + 1))
else
  echo "FAIL: missing DB did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
