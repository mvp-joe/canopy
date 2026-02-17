#!/usr/bin/env bash
# Tests for: canopy query transitive-callees
# Spec: cli-reference.md — "Same interface and output structure as transitive-callers,
#        but traverses in the callee direction."
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Helper: get symbol ID by name and file suffix
get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
ADD_ID=$(get_id "Add" "util.go")
MAIN_GO_ID=$(canopy query symbols --db "$DB" --kind function --file "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go" 2>/dev/null | jq -r '.results[] | select(.name == "main") | .id' | head -1)

# Test 1: transitive-callees for main, which calls Greet and Add
echo "=== Test 1: transitive-callees for main ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --symbol "$MAIN_GO_ID" 2>&1)
CMD="canopy query transitive-callees --db $DB --symbol $MAIN_GO_ID"

if echo "$OUTPUT" | jq -e '.command == "transitive-callees"' > /dev/null 2>&1; then
  echo "PASS: command field is transitive-callees"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not transitive-callees"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: same structure as transitive-callers (root, nodes, edges, depth)
if echo "$OUTPUT" | jq -e '.results | has("root", "nodes", "edges", "depth")' > /dev/null 2>&1; then
  echo "PASS: results has required fields"
  PASS=$((PASS + 1))
else
  echo "FAIL: results missing required fields"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# root should be main
if echo "$OUTPUT" | jq -e --argjson id "$MAIN_GO_ID" '.results.root == $id' > /dev/null 2>&1; then
  echo "PASS: root is $MAIN_GO_ID (main)"
  PASS=$((PASS + 1))
else
  echo "FAIL: root is not $MAIN_GO_ID"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Should find Greet and Add as callees
CALLEE_IDS=$(echo "$OUTPUT" | jq -r '.results.nodes[] | select(.depth > 0) | .symbol.id')
if echo "$CALLEE_IDS" | grep -q "$GREET_ID"; then
  echo "PASS: Greet found as callee"
  PASS=$((PASS + 1))
else
  echo "FAIL: Greet not found as callee"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

if echo "$CALLEE_IDS" | grep -q "$ADD_ID"; then
  echo "PASS: Add found as callee"
  PASS=$((PASS + 1))
else
  echo "FAIL: Add not found as callee"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: transitive-callees with --max-depth 0 returns only root
echo ""
echo "=== Test 2: transitive-callees --max-depth 0 ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --symbol "$MAIN_GO_ID" --max-depth 0 2>&1)
CMD="canopy query transitive-callees --db $DB --symbol $MAIN_GO_ID --max-depth 0"
if echo "$OUTPUT" | jq -e '.results.nodes | length == 1' > /dev/null 2>&1; then
  echo "PASS: max-depth 0 returns only root"
  PASS=$((PASS + 1))
else
  echo "FAIL: max-depth 0 returned more than root"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: transitive-callees for a leaf function (no callees)
echo ""
echo "=== Test 3: transitive-callees for leaf function (Greet) ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --symbol "$GREET_ID" 2>&1)
CMD="canopy query transitive-callees --db $DB --symbol $GREET_ID"
# Should still have the root node, but depth 0 and no edges
if echo "$OUTPUT" | jq -e '.results.nodes | length == 1 and .[0].depth == 0' > /dev/null 2>&1; then
  echo "PASS: leaf function has only root node at depth 0"
  PASS=$((PASS + 1))
else
  NODE_COUNT=$(echo "$OUTPUT" | jq '.results.nodes | length')
  echo "FAIL: expected 1 node for leaf, got $NODE_COUNT"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: transitive-callees for non-existent symbol
echo ""
echo "=== Test 4: transitive-callees for non-existent symbol ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --symbol 99999 2>&1)
CMD="canopy query transitive-callees --db $DB --symbol 99999"
if echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: non-existent symbol returns null"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent symbol did not return null"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: transitive-callees with negative max-depth errors
echo ""
echo "=== Test 5: transitive-callees negative max-depth ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --symbol "$MAIN_GO_ID" --max-depth -1 2>&1) || true
CMD="canopy query transitive-callees --db $DB --symbol $MAIN_GO_ID --max-depth -1"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: negative max-depth returns error"
  PASS=$((PASS + 1))
else
  echo "FAIL: negative max-depth did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: transitive-callees text format
echo ""
echo "=== Test 6: transitive-callees text format ==="
OUTPUT=$(canopy query transitive-callees --db "$DB" --format text --symbol "$MAIN_GO_ID" 2>&1)
CMD="canopy query transitive-callees --db $DB --format text --symbol $MAIN_GO_ID"
if echo "$OUTPUT" | grep -qi "main\|Greet\|Add"; then
  echo "PASS: text format includes symbol names"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format doesn't include expected symbols"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
