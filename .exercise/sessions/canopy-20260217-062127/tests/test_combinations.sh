#!/usr/bin/env bash
# Tests for combining multiple commands and cross-checking results
# Taxonomy categories: 3 (understanding symbols), 7 (refactoring), 9 (impact)
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
ADD_ID=$(get_id "Add" "util.go")
UTIL_CLASS_ID=$(get_id "Util" "Util.java")
MAIN_GO_ID=$(canopy query symbols --db "$DB" --kind function --file "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go" 2>/dev/null | jq -r '.results[] | select(.name == "main") | .id' | head -1)

# Test 1: symbol-detail + callers consistency
# If symbol-detail shows ref_count > 0, callers should find at least one caller
echo "=== Test 1: symbol-detail ref_count matches callers existence ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_ID" 2>&1)
REF_COUNT=$(echo "$DETAIL" | jq '.results.symbol.ref_count')
CALLERS=$(canopy query callers --db "$DB" --symbol "$GREET_ID" 2>&1)
CALLER_COUNT=$(echo "$CALLERS" | jq '.total_count')
if [ "$REF_COUNT" -gt 0 ] && [ "$CALLER_COUNT" -gt 0 ]; then
  echo "PASS: ref_count > 0 and callers found (ref=$REF_COUNT, callers=$CALLER_COUNT)"
  PASS=$((PASS + 1))
elif [ "$REF_COUNT" -eq 0 ] && [ "$CALLER_COUNT" -eq 0 ]; then
  echo "PASS: ref_count == 0 and no callers"
  PASS=$((PASS + 1))
else
  echo "FAIL: ref_count ($REF_COUNT) inconsistent with caller count ($CALLER_COUNT)"
  FAIL=$((FAIL + 1))
fi

# Test 2: transitive-callers depth=1 should match direct callers
echo ""
echo "=== Test 2: transitive-callers depth=1 should match direct callers ==="
TC=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_ID" --max-depth 1 2>&1)
TC_CALLER_IDS=$(echo "$TC" | jq -r '[.results.nodes[] | select(.depth == 1) | .symbol.id] | sort | .[]')
DIRECT_CALLER_IDS=$(echo "$CALLERS" | jq -r '[.results[].caller_id] | sort | .[]')
if [ "$TC_CALLER_IDS" = "$DIRECT_CALLER_IDS" ]; then
  echo "PASS: transitive depth-1 callers match direct callers"
  PASS=$((PASS + 1))
else
  echo "FAIL: transitive depth-1 callers don't match direct callers"
  echo "  Transitive: $TC_CALLER_IDS"
  echo "  Direct: $DIRECT_CALLER_IDS"
  FAIL=$((FAIL + 1))
fi

# Test 3: transitive-callees depth=1 should match direct callees
echo ""
echo "=== Test 3: transitive-callees depth=1 matches direct callees ==="
TC2=$(canopy query transitive-callees --db "$DB" --symbol "$MAIN_GO_ID" --max-depth 1 2>&1)
TC2_CALLEE_IDS=$(echo "$TC2" | jq -r '[.results.nodes[] | select(.depth == 1) | .symbol.id] | sort | .[]')
DIRECT_CALLEES=$(canopy query callees --db "$DB" --symbol "$MAIN_GO_ID" 2>&1)
DIRECT_CALLEE_IDS=$(echo "$DIRECT_CALLEES" | jq -r '[.results[].callee_id] | sort | .[]')
if [ "$TC2_CALLEE_IDS" = "$DIRECT_CALLEE_IDS" ]; then
  echo "PASS: transitive depth-1 callees match direct callees"
  PASS=$((PASS + 1))
else
  echo "FAIL: transitive depth-1 callees don't match direct callees"
  echo "  Transitive: $TC2_CALLEE_IDS"
  echo "  Direct: $DIRECT_CALLEE_IDS"
  FAIL=$((FAIL + 1))
fi

# Test 4: unused symbols should not appear in hotspots
echo ""
echo "=== Test 4: unused symbols should not be in hotspots ==="
UNUSED_IDS=$(canopy query unused --db "$DB" --limit 500 2>&1 | jq -r '[.results[].id] | sort | .[]')
HOTSPOT_IDS=$(canopy query hotspots --db "$DB" 2>&1 | jq -r '[.results[].symbol.id] | sort | .[]')
OVERLAP=0
for hid in $HOTSPOT_IDS; do
  if echo "$UNUSED_IDS" | grep -q "^${hid}$"; then
    OVERLAP=$((OVERLAP + 1))
  fi
done
if [ "$OVERLAP" -eq 0 ]; then
  echo "PASS: no overlap between unused and hotspots"
  PASS=$((PASS + 1))
else
  echo "FAIL: $OVERLAP symbols appear in both unused and hotspots"
  FAIL=$((FAIL + 1))
fi

# Test 5: symbols --ref-count-min 1 should exclude unused symbols
echo ""
echo "=== Test 5: ref-count-min 1 excludes unused ==="
REF_MIN1=$(canopy query symbols --db "$DB" --ref-count-min 1 --limit 500 2>&1)
REF_MIN1_IDS=$(echo "$REF_MIN1" | jq -r '[.results[].id] | sort | .[]')
OVERLAP2=0
for uid in $UNUSED_IDS; do
  if echo "$REF_MIN1_IDS" | grep -q "^${uid}$"; then
    OVERLAP2=$((OVERLAP2 + 1))
  fi
done
if [ "$OVERLAP2" -eq 0 ]; then
  echo "PASS: ref-count-min 1 excludes all unused symbols"
  PASS=$((PASS + 1))
else
  echo "FAIL: $OVERLAP2 unused symbols found in ref-count-min 1 results"
  FAIL=$((FAIL + 1))
fi

# Test 6: type-hierarchy implements should match implements command
echo ""
echo "=== Test 6: type-hierarchy.implements matches implements command ==="
TH=$(canopy query type-hierarchy --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
TH_IMPL_COUNT=$(echo "$TH" | jq '.results.implements | length')
IMPL=$(canopy query implements --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
IMPL_COUNT=$(echo "$IMPL" | jq '.results | length')
if [ "$TH_IMPL_COUNT" = "$IMPL_COUNT" ]; then
  echo "PASS: type-hierarchy implements count ($TH_IMPL_COUNT) matches implements command ($IMPL_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: type-hierarchy implements count ($TH_IMPL_COUNT) != implements command ($IMPL_COUNT)"
  FAIL=$((FAIL + 1))
fi

# Test 7: hotspots caller_count matches callers command for top symbol
echo ""
echo "=== Test 7: hotspots caller_count matches callers query ==="
TOP_HOTSPOT=$(canopy query hotspots --db "$DB" --top 1 2>&1)
TOP_ID=$(echo "$TOP_HOTSPOT" | jq '.results[0].symbol.id')
TOP_CALLER_COUNT=$(echo "$TOP_HOTSPOT" | jq '.results[0].caller_count')
ACTUAL_CALLERS=$(canopy query callers --db "$DB" --symbol "$TOP_ID" 2>&1)
ACTUAL_CALLER_COUNT=$(echo "$ACTUAL_CALLERS" | jq '.total_count')
if [ "$TOP_CALLER_COUNT" = "$ACTUAL_CALLER_COUNT" ]; then
  echo "PASS: hotspot caller_count ($TOP_CALLER_COUNT) matches callers query ($ACTUAL_CALLER_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: hotspot caller_count ($TOP_CALLER_COUNT) != callers query ($ACTUAL_CALLER_COUNT)"
  FAIL=$((FAIL + 1))
fi

# Test 8: hotspots callee_count matches callees command for top symbol
echo ""
echo "=== Test 8: hotspots callee_count matches callees query ==="
TOP_CALLEE_COUNT=$(echo "$TOP_HOTSPOT" | jq '.results[0].callee_count')
ACTUAL_CALLEES=$(canopy query callees --db "$DB" --symbol "$TOP_ID" 2>&1)
ACTUAL_CALLEE_COUNT=$(echo "$ACTUAL_CALLEES" | jq '.total_count')
if [ "$TOP_CALLEE_COUNT" = "$ACTUAL_CALLEE_COUNT" ]; then
  echo "PASS: hotspot callee_count ($TOP_CALLEE_COUNT) matches callees query ($ACTUAL_CALLEE_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: hotspot callee_count ($TOP_CALLEE_COUNT) != callees query ($ACTUAL_CALLEE_COUNT)"
  FAIL=$((FAIL + 1))
fi

# Test 9: symbol-detail members for Java class should include known methods
echo ""
echo "=== Test 9: Util class members include greet, add, isPositive ==="
DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$UTIL_CLASS_ID" 2>&1)
MEMBER_NAMES=$(echo "$DETAIL" | jq -r '[.results.members[].name] | sort | .[]')
for mname in greet add isPositive; do
  if echo "$MEMBER_NAMES" | grep -q "$mname"; then
    echo "PASS: member $mname found in Util class"
    PASS=$((PASS + 1))
  else
    echo "FAIL: member $mname not found in Util class"
    echo "  Members: $MEMBER_NAMES"
    FAIL=$((FAIL + 1))
  fi
done

# Test 10: scope-at position inside function should include function scope
echo ""
echo "=== Test 10: scope-at includes function scope kind ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 3 5 2>&1)
SCOPE_KINDS=$(echo "$OUTPUT" | jq -r '[.results[].kind] | .[]')
if echo "$SCOPE_KINDS" | grep -q "function"; then
  echo "PASS: function scope found in scope chain"
  PASS=$((PASS + 1))
else
  echo "FAIL: no function scope in scope chain"
  echo "  Scope kinds: $SCOPE_KINDS"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
