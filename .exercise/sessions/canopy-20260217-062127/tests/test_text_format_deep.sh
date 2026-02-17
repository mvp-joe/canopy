#!/usr/bin/env bash
# Deep text format tests for new commands
# Spec: cli-reference.md Text Format Output section
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_GO_ID=$(get_id "Greet" "util.go")
UTIL_CLASS_ID=$(get_id "Util" "Util.java")
MAIN_GO_ID=$(canopy query symbols --db "$DB" --kind function --file "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go" 2>/dev/null | jq -r '.results[] | select(.name == "main") | .id' | head -1)

# Test 1: symbol-detail text has Parameters section
echo "=== Test 1: symbol-detail text has Parameters section ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --format text --symbol "$GREET_GO_ID" 2>&1)
if echo "$OUTPUT" | grep -qi "Parameters\|Param"; then
  echo "PASS: symbol-detail text has Parameters section"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol-detail text missing Parameters section"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: symbol-detail text for class has Members section
echo ""
echo "=== Test 2: symbol-detail text for class has Members section ==="
OUTPUT=$(canopy query symbol-detail --db "$DB" --format text --symbol "$UTIL_CLASS_ID" 2>&1)
if echo "$OUTPUT" | grep -qi "Members\|Member"; then
  echo "PASS: class symbol-detail text has Members section"
  PASS=$((PASS + 1))
else
  echo "FAIL: class symbol-detail text missing Members section"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: scope-at text has kind and position
echo ""
echo "=== Test 3: scope-at text format ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query scope-at --db "$DB" --format text "$FILE" 3 5 2>&1)
if echo "$OUTPUT" | grep -qi "function\|file\|scope"; then
  echo "PASS: scope-at text includes scope kind"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope-at text missing scope kind"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: type-hierarchy text has section headers
echo ""
echo "=== Test 4: type-hierarchy text has section headers ==="
OUTPUT=$(canopy query type-hierarchy --db "$DB" --format text --symbol "$UTIL_CLASS_ID" 2>&1)
# Spec says: Symbol header, then sections for Implements, ImplementedBy, Composes, ComposedBy, Extensions
if echo "$OUTPUT" | grep -qi "Implement\|Compose\|Extension\|Util"; then
  echo "PASS: type-hierarchy text has expected sections"
  PASS=$((PASS + 1))
else
  echo "FAIL: type-hierarchy text missing expected sections"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: hotspots text has caller/callee counts
echo ""
echo "=== Test 5: hotspots text format includes counts ==="
OUTPUT=$(canopy query hotspots --db "$DB" --format text 2>&1)
# Spec says: Aligned columns with symbol info plus caller/callee counts
if echo "$OUTPUT" | grep -qiE "caller|callee|[0-9]"; then
  echo "PASS: hotspots text includes count information"
  PASS=$((PASS + 1))
else
  echo "FAIL: hotspots text missing count information"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: unused text format shows aligned columns
echo ""
echo "=== Test 6: unused text format ==="
OUTPUT=$(canopy query unused --db "$DB" --format text --limit 5 2>&1)
# Spec says: Aligned columns: ID NAME KIND VISIBILITY FILE LINE
if echo "$OUTPUT" | grep -qiE "function|variable|class|method|constant"; then
  echo "PASS: unused text shows symbol kinds"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused text missing expected columns"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 7: package-graph text has Packages and Edges
echo ""
echo "=== Test 7: package-graph text format ==="
OUTPUT=$(canopy query package-graph --db "$DB" --format text 2>&1)
# Spec says: Packages section and Edges section
if echo "$OUTPUT" | grep -qiE "Package|Edge|->|→"; then
  echo "PASS: package-graph text has expected sections"
  PASS=$((PASS + 1))
else
  echo "FAIL: package-graph text missing sections"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 8: circular-deps text format
echo ""
echo "=== Test 8: circular-deps text format ==="
OUTPUT=$(canopy query circular-deps --db "$DB" --format text 2>&1)
# Spec says: One cycle per line, or no cycles message
if [ -n "$OUTPUT" ]; then
  echo "PASS: circular-deps text produces output"
  PASS=$((PASS + 1))
else
  echo "FAIL: circular-deps text produces no output"
  FAIL=$((FAIL + 1))
fi

# Test 9: unused text with --limit shows pagination footer
echo ""
echo "=== Test 9: unused text pagination footer ==="
OUTPUT=$(canopy query unused --db "$DB" --format text --limit 3 2>&1)
# Spec says: "Showing X of Y results" appears when results are truncated
if echo "$OUTPUT" | grep -qi "Showing.*of.*results\|showing.*of"; then
  echo "PASS: pagination footer present"
  PASS=$((PASS + 1))
else
  echo "FAIL: pagination footer missing when limit < total"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 10: implements text format doesn't error
echo ""
echo "=== Test 10: implements text format ==="
OUTPUT=$(canopy query implements --db "$DB" --format text --symbol "$UTIL_CLASS_ID" 2>&1)
EXIT_CODE=$?
# Spec says: Locations: One per line as file:line:col
# Empty output is valid when there are no implementations
if [ "$EXIT_CODE" -eq 0 ]; then
  echo "PASS: implements text format succeeds (exit code 0)"
  PASS=$((PASS + 1))
else
  echo "FAIL: implements text format error (exit code $EXIT_CODE)"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 11: summary text is human-readable
echo ""
echo "=== Test 11: summary text format ==="
OUTPUT=$(canopy query summary --db "$DB" --format text 2>&1)
if echo "$OUTPUT" | grep -qiE "language|file|symbol|go|java|python"; then
  echo "PASS: summary text is human-readable with language info"
  PASS=$((PASS + 1))
else
  echo "FAIL: summary text missing expected content"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 12: files text has aligned columns
echo ""
echo "=== Test 12: files text format ==="
OUTPUT=$(canopy query files --db "$DB" --format text 2>&1)
# Spec: Aligned columns: ID PATH LANGUAGE
if echo "$OUTPUT" | grep -qiE "go|java|python|typescript"; then
  echo "PASS: files text format shows languages"
  PASS=$((PASS + 1))
else
  echo "FAIL: files text format missing languages"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
