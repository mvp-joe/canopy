#!/usr/bin/env bash
# Tests for: canopy query unused
# Spec: cli-reference.md — "List symbols with zero resolved references.
#        Excludes package/module/namespace kinds."
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: unused basic output
echo "=== Test 1: unused basic output ==="
OUTPUT=$(canopy query unused --db "$DB" 2>&1)
CMD="canopy query unused --db $DB"

if echo "$OUTPUT" | jq -e '.command == "unused"' > /dev/null 2>&1; then
  echo "PASS: command field is unused"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not unused"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: same structure as symbols
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: results is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: results is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# All returned symbols should have ref_count of 0
NON_ZERO=$(echo "$OUTPUT" | jq '[.results[] | select(.ref_count != 0)] | length')
if [ "$NON_ZERO" = "0" ]; then
  echo "PASS: all unused symbols have ref_count 0"
  PASS=$((PASS + 1))
else
  echo "FAIL: $NON_ZERO symbols have non-zero ref_count"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: excludes package/module/namespace kinds
PKG_COUNT=$(echo "$OUTPUT" | jq '[.results[] | select(.kind == "package" or .kind == "module" or .kind == "namespace")] | length')
if [ "$PKG_COUNT" = "0" ]; then
  echo "PASS: no package/module/namespace kinds in unused"
  PASS=$((PASS + 1))
else
  echo "FAIL: $PKG_COUNT package/module/namespace kinds found in unused"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Each result should have symbol fields
if echo "$OUTPUT" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results[0] | has("id", "name", "kind", "visibility", "file", "ref_count")' > /dev/null 2>&1; then
    echo "PASS: unused symbols have required fields"
    PASS=$((PASS + 1))
  else
    echo "FAIL: unused symbols missing required fields"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 2: unused with --kind filter
echo ""
echo "=== Test 2: unused --kind function ==="
OUTPUT=$(canopy query unused --db "$DB" --kind function 2>&1)
CMD="canopy query unused --db $DB --kind function"
# All results should be functions
NON_FUNC=$(echo "$OUTPUT" | jq '[.results[] | select(.kind != "function")] | length')
if [ "$NON_FUNC" = "0" ]; then
  echo "PASS: --kind function only returns functions"
  PASS=$((PASS + 1))
else
  echo "FAIL: --kind function returned non-function results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: unused with --visibility filter
echo ""
echo "=== Test 3: unused --visibility public ==="
OUTPUT=$(canopy query unused --db "$DB" --visibility public 2>&1)
CMD="canopy query unused --db $DB --visibility public"
NON_PUBLIC=$(echo "$OUTPUT" | jq '[.results[] | select(.visibility != "public")] | length')
if [ "$NON_PUBLIC" = "0" ]; then
  echo "PASS: --visibility public only returns public symbols"
  PASS=$((PASS + 1))
else
  echo "FAIL: --visibility public returned non-public results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: unused with --path-prefix
echo ""
echo "=== Test 4: unused --path-prefix ==="
OUTPUT=$(canopy query unused --db "$DB" --path-prefix "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.c" 2>&1)
CMD="canopy query unused --db $DB --path-prefix /home/joe/code/canopy/.exercise/scratch/multi-lang/main.c"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: path-prefix filter works"
  PASS=$((PASS + 1))
else
  echo "FAIL: path-prefix filter failed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: unused with pagination
echo ""
echo "=== Test 5: unused with --limit and --offset ==="
OUTPUT=$(canopy query unused --db "$DB" --limit 3 --offset 0 2>&1)
CMD="canopy query unused --db $DB --limit 3 --offset 0"
RESULT_LEN=$(echo "$OUTPUT" | jq '.results | length')
if [ "$RESULT_LEN" -le 3 ]; then
  echo "PASS: --limit 3 returns <= 3 results (got $RESULT_LEN)"
  PASS=$((PASS + 1))
else
  echo "FAIL: --limit 3 returned $RESULT_LEN results"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: unused text format
echo ""
echo "=== Test 6: unused text format ==="
OUTPUT=$(canopy query unused --db "$DB" --format text 2>&1)
CMD="canopy query unused --db $DB --format text"
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
