#!/usr/bin/env bash
# Tests for: canopy query transitive-callers
# Spec: cli-reference.md "Call Graph Commands" — transitive-callers section
set -euo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Helper: get symbol ID by name and file suffix
get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
MAIN_GO_ID=$(canopy query symbols --db "$DB" --kind function --file "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go" 2>/dev/null | jq -r '.results[] | select(.name == "main") | .id' | head -1)

# Test 1: transitive-callers for Greet, which is called by main
echo "=== Test 1: transitive-callers for Greet ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_ID" 2>&1)
CMD="canopy query transitive-callers --db $DB --symbol $GREET_ID"

if echo "$OUTPUT" | jq -e '.command == "transitive-callers"' > /dev/null 2>&1; then
  echo "PASS: command field is transitive-callers"
  PASS=$((PASS + 1))
else
  echo "FAIL: command field is not transitive-callers"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: results should have root, nodes, edges, depth
if echo "$OUTPUT" | jq -e '.results | has("root", "nodes", "edges", "depth")' > /dev/null 2>&1; then
  echo "PASS: results has required fields (root, nodes, edges, depth)"
  PASS=$((PASS + 1))
else
  echo "FAIL: results missing required fields"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# root should be the symbol id (Greet)
if echo "$OUTPUT" | jq -e --argjson id "$GREET_ID" '.results.root == $id' > /dev/null 2>&1; then
  echo "PASS: root is $GREET_ID (Greet)"
  PASS=$((PASS + 1))
else
  echo "FAIL: root is not $GREET_ID"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Spec: depth 0 node is the root symbol itself
if echo "$OUTPUT" | jq -e --argjson id "$GREET_ID" '.results.nodes[] | select(.depth == 0) | .symbol.id == $id' > /dev/null 2>&1; then
  echo "PASS: depth-0 node is root symbol"
  PASS=$((PASS + 1))
else
  echo "FAIL: depth-0 node is not root symbol"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Should find main as a caller at depth 1
if echo "$OUTPUT" | jq -e --argjson id "$MAIN_GO_ID" '.results.nodes[] | select(.depth == 1 and .symbol.id == $id)' > /dev/null 2>&1; then
  echo "PASS: main found as depth-1 caller"
  PASS=$((PASS + 1))
else
  echo "FAIL: main not found as depth-1 caller"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Edges should have caller_id, callee_id, file, line, col
if echo "$OUTPUT" | jq -e '.results.edges | length > 0' > /dev/null 2>&1; then
  if echo "$OUTPUT" | jq -e '.results.edges[0] | has("caller_id", "callee_id", "file", "line", "col")' > /dev/null 2>&1; then
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

# Node symbol should have full symbol info including ref counts
if echo "$OUTPUT" | jq -e '.results.nodes[0].symbol | has("id", "name", "kind", "visibility", "file", "ref_count")' > /dev/null 2>&1; then
  echo "PASS: node symbol has full info"
  PASS=$((PASS + 1))
else
  echo "FAIL: node symbol missing fields"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: transitive-callers with --max-depth 0 returns only root
echo ""
echo "=== Test 2: transitive-callers --max-depth 0 ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_ID" --max-depth 0 2>&1)
CMD="canopy query transitive-callers --db $DB --symbol $GREET_ID --max-depth 0"
if echo "$OUTPUT" | jq -e '.results.nodes | length == 1' > /dev/null 2>&1; then
  echo "PASS: max-depth 0 returns only root node"
  PASS=$((PASS + 1))
else
  echo "FAIL: max-depth 0 returned more than 1 node"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: transitive-callers with --max-depth 1 limits traversal
echo ""
echo "=== Test 3: transitive-callers --max-depth 1 ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_ID" --max-depth 1 2>&1)
CMD="canopy query transitive-callers --db $DB --symbol $GREET_ID --max-depth 1"
if echo "$OUTPUT" | jq -e '.results.depth <= 1' > /dev/null 2>&1; then
  echo "PASS: depth <= 1"
  PASS=$((PASS + 1))
else
  echo "FAIL: depth > 1 with --max-depth 1"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: transitive-callers for non-existent symbol returns null
echo ""
echo "=== Test 4: transitive-callers for non-existent symbol ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol 99999 2>&1)
CMD="canopy query transitive-callers --db $DB --symbol 99999"
if echo "$OUTPUT" | jq -e '.results == null' > /dev/null 2>&1; then
  echo "PASS: non-existent symbol returns null"
  PASS=$((PASS + 1))
else
  echo "FAIL: non-existent symbol did not return null"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 5: transitive-callers with negative --max-depth returns error
echo ""
echo "=== Test 5: transitive-callers with negative max-depth ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_ID" --max-depth -1 2>&1) || true
CMD="canopy query transitive-callers --db $DB --symbol $GREET_ID --max-depth -1"
if echo "$OUTPUT" | jq -e '.error' > /dev/null 2>&1 || echo "$OUTPUT" | grep -qi "error"; then
  echo "PASS: negative max-depth returns error"
  PASS=$((PASS + 1))
else
  echo "FAIL: negative max-depth did not error"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 6: transitive-callers by positional args
echo ""
echo "=== Test 6: transitive-callers by position ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/util.go"
OUTPUT=$(canopy query transitive-callers --db "$DB" "$FILE" 2 5 2>&1)
CMD="canopy query transitive-callers --db $DB $FILE 2 5"
if echo "$OUTPUT" | jq -e --argjson id "$GREET_ID" '.results.root == $id' > /dev/null 2>&1; then
  echo "PASS: positional lookup finds Greet"
  PASS=$((PASS + 1))
else
  echo "FAIL: positional lookup didn't find Greet"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 7: transitive-callers text format
echo ""
echo "=== Test 7: transitive-callers text format ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --format text --symbol "$GREET_ID" 2>&1)
CMD="canopy query transitive-callers --db $DB --format text --symbol $GREET_ID"
if echo "$OUTPUT" | grep -qi "Greet\|main"; then
  echo "PASS: text format includes symbol names"
  PASS=$((PASS + 1))
else
  echo "FAIL: text format doesn't include expected symbols"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 8: transitive-callers with --max-depth > 100 caps at 100
echo ""
echo "=== Test 8: transitive-callers with max-depth > 100 ==="
OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_ID" --max-depth 200 2>&1)
CMD="canopy query transitive-callers --db $DB --symbol $GREET_ID --max-depth 200"
# Spec says "Values > 100 are capped at 100" — should still work, not error
if echo "$OUTPUT" | jq -e '.results != null' > /dev/null 2>&1; then
  echo "PASS: max-depth > 100 accepted (capped)"
  PASS=$((PASS + 1))
else
  echo "FAIL: max-depth > 100 failed"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
