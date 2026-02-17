#!/usr/bin/env bash
# Tests for: canopy query reexports
# Spec: cli-reference.md — "Find re-exported symbols from a file."
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: reexports for a TypeScript file (likely has re-exports)
echo "=== Test 1: reexports for TypeScript file ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.ts"
OUTPUT=$(canopy query reexports --db "$DB" "$FILE" 2>&1)
CMD="canopy query reexports --db $DB $FILE"

if echo "$OUTPUT" | jq -e '.command == "reexports"' > /dev/null 2>&1; then
  echo "PASS: command field is reexports"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not reexports"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: results should be an array
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# If there are results, check fields: file_id, original_name, alias, source, kind
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results[0] | has("file_id", "original_name", "alias", "source", "kind")' > /dev/null 2>&1; then
    echo "PASS: reexport entry has required fields"
    PASS=$((PASS + 1))
  else
    echo "FAIL: reexport entry missing required fields"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no reexports found in this file (may be expected)"
fi

# Test 2: reexports for a Go file (usually no re-exports)
echo ""
echo "=== Test 2: reexports for Go file ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query reexports --db "$DB" "$FILE" 2>&1)
CMD="canopy query reexports --db $DB $FILE"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: Go file reexports returns array"
  PASS=$((PASS + 1))
else
  echo "FAIL: Go file reexports not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: reexports for non-existent file
echo ""
echo "=== Test 3: reexports for non-existent file ==="
OUTPUT=$(canopy query reexports --db "$DB" "/nonexistent/file.ts" 2>&1) || true
CMD="canopy query reexports --db $DB /nonexistent/file.ts"
if echo "$OUTPUT" | jq -e '.results | type == "array" and length == 0' > /dev/null 2>&1 || echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1; then
  echo "PASS: non-existent file returns empty or error"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent file returned unexpected result"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: reexports for JS file
echo ""
echo "=== Test 4: reexports for JavaScript file ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.js"
OUTPUT=$(canopy query reexports --db "$DB" "$FILE" 2>&1)
CMD="canopy query reexports --db $DB $FILE"
if echo "$OUTPUT" | jq -e '.command == "reexports"' > /dev/null 2>&1; then
  echo "PASS: reexports works for JS file"
  PASS=$((PASS + 1))
else
  echo "FAIL: reexports failed for JS file"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
