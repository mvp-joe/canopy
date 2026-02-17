#!/usr/bin/env bash
# Deep tests for symbol-detail: parameters, members, type_params, annotations
# Taxonomy categories: 3 (understanding symbols), 12 (type queries)
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

# Get IDs for various symbol types
GREET_GO_ID=$(get_id "Greet" "util.go")
ADD_GO_ID=$(get_id "Add" "util.go")
UTIL_CLASS_ID=$(get_id "Util" "Util.java")
GREET_JAVA_ID=$(get_id "greet" "Util.java")
ADD_JAVA_ID=$(get_id "add" "Util.java")
MAIN_GO_ID=$(canopy query symbols --db "$DB" --kind function --file "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go" 2>/dev/null | jq -r '.results[] | select(.name == "main") | .id' | head -1)

# Test 1: Go function Greet has parameters
echo "=== Test 1: Go Greet function has parameters ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_GO_ID" 2>&1)
PARAM_COUNT=$(echo "$DETAIL" | jq '.results.parameters | length')
if [ "$PARAM_COUNT" -gt 0 ]; then
  echo "PASS: Greet has $PARAM_COUNT parameter(s)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Greet has no parameters"
  echo "  Detail: $(echo "$DETAIL" | jq '.results.parameters')"
  FAIL=$((FAIL + 1))
fi

# Test 2: Parameter has required fields (name, type, ordinal, is_receiver, is_return)
echo ""
echo "=== Test 2: parameter has required fields ==="
if echo "$DETAIL" | jq -e '.results.parameters[0] | has("name", "ordinal")' > /dev/null 2>&1; then
  echo "PASS: parameter has required fields"
  PASS=$((PASS + 1))
else
  echo "FAIL: parameter missing fields"
  echo "  First param: $(echo "$DETAIL" | jq '.results.parameters[0]')"
  FAIL=$((FAIL + 1))
fi

# Test 3: Go Add function has 2 parameters (a, b)
echo ""
echo "=== Test 3: Go Add function parameters ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$ADD_GO_ID" 2>&1)
PARAM_NAMES=$(echo "$DETAIL" | jq -r '[.results.parameters[] | select(.is_return != true and .is_receiver != true) | .name] | sort | .[]')
if echo "$PARAM_NAMES" | grep -q "a"; then
  echo "PASS: Add function has parameter 'a'"
  PASS=$((PASS + 1))
else
  echo "FAIL: Add function missing parameter 'a'"
  echo "  Params: $PARAM_NAMES"
  FAIL=$((FAIL + 1))
fi

if echo "$PARAM_NAMES" | grep -q "b"; then
  echo "PASS: Add function has parameter 'b'"
  PASS=$((PASS + 1))
else
  echo "FAIL: Add function missing parameter 'b'"
  echo "  Params: $PARAM_NAMES"
  FAIL=$((FAIL + 1))
fi

# Test 4: Java Util class has members
echo ""
echo "=== Test 4: Java Util class has members ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
MEMBER_COUNT=$(echo "$DETAIL" | jq '.results.members | length')
if [ "$MEMBER_COUNT" -gt 0 ]; then
  echo "PASS: Util class has $MEMBER_COUNT member(s)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Util class has no members"
  FAIL=$((FAIL + 1))
fi

# Test 5: Non-class symbol (function) has empty members
echo ""
echo "=== Test 5: function has empty members array ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_GO_ID" 2>&1)
MEMBER_COUNT=$(echo "$DETAIL" | jq '.results.members | length')
if [ "$MEMBER_COUNT" -eq 0 ]; then
  echo "PASS: function has empty members array"
  PASS=$((PASS + 1))
else
  echo "FAIL: function has $MEMBER_COUNT members"
  FAIL=$((FAIL + 1))
fi

# Test 6: symbol-detail returns all required top-level fields
echo ""
echo "=== Test 6: symbol-detail has all required result fields ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_GO_ID" 2>&1)
if echo "$DETAIL" | jq -e '.results | has("symbol", "parameters", "members", "type_params", "annotations")' > /dev/null 2>&1; then
  echo "PASS: results has all required fields (symbol, parameters, members, type_params, annotations)"
  PASS=$((PASS + 1))
else
  FIELDS=$(echo "$DETAIL" | jq '.results | keys')
  echo "FAIL: results missing fields. Has: $FIELDS"
  FAIL=$((FAIL + 1))
fi

# Test 7: symbol-detail symbol has ref_count fields
echo ""
echo "=== Test 7: symbol in detail has ref_count fields ==="
if echo "$DETAIL" | jq -e '.results.symbol | has("ref_count", "external_ref_count", "internal_ref_count")' > /dev/null 2>&1; then
  echo "PASS: symbol has ref_count, external_ref_count, internal_ref_count"
  PASS=$((PASS + 1))
else
  echo "FAIL: symbol missing ref count fields"
  FAIL=$((FAIL + 1))
fi

# Test 8: Java method (Util.add) has parameters
echo ""
echo "=== Test 8: Java Util.add method has parameters ==="
if [ -n "$ADD_JAVA_ID" ]; then
  DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$ADD_JAVA_ID" 2>&1)
  PARAM_COUNT=$(echo "$DETAIL" | jq '.results.parameters | length')
  if [ "$PARAM_COUNT" -gt 0 ]; then
    echo "PASS: Java add method has $PARAM_COUNT parameter(s)"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Java add method has no parameters"
    FAIL=$((FAIL + 1))
  fi
else
  echo "FAIL: could not find Java add method ID"
  FAIL=$((FAIL + 1))
fi

# Test 9: type_params and annotations are arrays (even if empty)
echo ""
echo "=== Test 9: type_params and annotations are arrays ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_GO_ID" 2>&1)
TP_TYPE=$(echo "$DETAIL" | jq '.results.type_params | type')
AN_TYPE=$(echo "$DETAIL" | jq '.results.annotations | type')
if [ "$TP_TYPE" = '"array"' ] && [ "$AN_TYPE" = '"array"' ]; then
  echo "PASS: type_params and annotations are arrays"
  PASS=$((PASS + 1))
else
  echo "FAIL: type_params type=$TP_TYPE, annotations type=$AN_TYPE"
  FAIL=$((FAIL + 1))
fi

# Test 10: member has name, kind, visibility fields
echo ""
echo "=== Test 10: class members have required fields ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
if echo "$DETAIL" | jq -e '.results.members[0] | has("name", "kind", "visibility")' > /dev/null 2>&1; then
  echo "PASS: member has name, kind, visibility"
  PASS=$((PASS + 1))
else
  FIRST_MEMBER=$(echo "$DETAIL" | jq '.results.members[0]')
  echo "FAIL: member missing fields: $FIRST_MEMBER"
  FAIL=$((FAIL + 1))
fi

# Test 11: parameter ordinals are sequential starting from 0
echo ""
echo "=== Test 11: parameter ordinals are sequential ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$ADD_GO_ID" 2>&1)
ORDINALS=$(echo "$DETAIL" | jq -r '[.results.parameters[] | select(.is_return != true) | .ordinal] | sort | .[]')
EXPECTED=0
SEQUENTIAL=1
for o in $ORDINALS; do
  if [ "$o" != "$EXPECTED" ]; then
    SEQUENTIAL=0
    break
  fi
  EXPECTED=$((EXPECTED + 1))
done
if [ "$SEQUENTIAL" -eq 1 ] && [ "$EXPECTED" -gt 0 ]; then
  echo "PASS: ordinals are sequential 0..$((EXPECTED-1))"
  PASS=$((PASS + 1))
else
  echo "FAIL: ordinals not sequential: $ORDINALS"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
