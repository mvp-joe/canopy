#!/usr/bin/env bash
# Tests for: canopy query package-graph
# Spec: cli-reference.md — "Shows the package-to-package dependency graph"
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: package-graph basic structure
echo "=== Test 1: package-graph basic structure ==="
OUTPUT=$(canopy query package-graph --db "$DB" 2>&1)
CMD="canopy query package-graph --db $DB"

if echo "$OUTPUT" | jq -e '.command == "package-graph"' > /dev/null 2>&1; then
  echo "PASS: command field is package-graph"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not package-graph"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: results should have packages and edges
if echo "$OUTPUT" | jq -e '.results | has("packages", "edges")' > /dev/null 2>&1; then
  echo "PASS: results has packages and edges"
  PASS=$((PASS + 1))
else
  echo "FAIL: results missing packages or edges"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# packages should be an array
if echo "$OUTPUT" | jq -e '.results.packages | type == "array"' > /dev/null 2>&1; then
  echo "PASS: packages is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: packages is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# edges should be an array
if echo "$OUTPUT" | jq -e '.results.edges | type == "array"' > /dev/null 2>&1; then
  echo "PASS: edges is an array"
  PASS=$((PASS + 1))
else
  echo "FAIL: edges is not an array"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Each package should have name, file_count, line_count
if echo "$OUTPUT" | jq -e '.results.packages | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results.packages[0] | has("name", "file_count", "line_count")' > /dev/null 2>&1; then
    echo "PASS: package has required fields"
    PASS=$((PASS + 1))
  else
    echo "FAIL: package missing required fields"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no packages found"
fi

# If there are edges, each should have from_package, to_package, import_count
if echo "$OUTPUT" | jq -e '.results.edges | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results.edges[0] | has("from_package", "to_package", "import_count")' > /dev/null 2>&1; then
    echo "PASS: edge has required fields"
    PASS=$((PASS + 1))
  else
    echo "FAIL: edge missing required fields"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no edges found"
fi

# total_count should be 1 (per spec)
if echo "$OUTPUT" | jq -e '.total_count == 1' > /dev/null 2>&1; then
  echo "PASS: total_count is 1"
  PASS=$((PASS + 1))
else
  echo "FAIL: total_count is not 1"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: package-graph text format
echo ""
echo "=== Test 2: package-graph text format ==="
OUTPUT=$(canopy query package-graph --db "$DB" --format text 2>&1)
CMD="canopy query package-graph --db $DB --format text"
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
