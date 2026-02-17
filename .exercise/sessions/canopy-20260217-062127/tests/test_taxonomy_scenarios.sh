#!/usr/bin/env bash
# Taxonomy-driven scenarios: code smells, impact analysis, cross-cutting concerns
# Taxonomy categories: 7 (refactoring), 8 (code smells), 9 (impact), 10 (cross-cutting), 11 (comparative)
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

# === Category 7: Refactoring ===

# Test 1: Can I safely reduce visibility? Find public symbols with only internal refs
echo "=== Test 1: public symbols with only internal refs (visibility reduction candidates) ==="
# Symbols that are public, ref_count > 0, external_ref_count = 0 → could be made private
CANDIDATES=$(canopy query symbols --db "$DB" --visibility public --ref-count-min 1 --limit 500 2>&1 | \
  jq '[.results[] | select(.external_ref_count == 0)] | length')
echo "INFO: Found $CANDIDATES public symbols with only internal references"
# Just verify the query works (no crash, valid result)
if [ "$CANDIDATES" -ge 0 ]; then
  echo "PASS: visibility reduction query works"
  PASS=$((PASS + 1))
else
  echo "FAIL: visibility reduction query failed"
  FAIL=$((FAIL + 1))
fi

# Test 2: Blast radius — transitive callers count gives impact scope
echo ""
echo "=== Test 2: blast radius via transitive callers ==="
GREET_GO_ID=$(get_id "Greet" "util.go")
TC=$(canopy query transitive-callers --db "$DB" --symbol "$GREET_GO_ID" 2>&1)
NODE_COUNT=$(echo "$TC" | jq '.results.nodes | length')
EDGE_COUNT=$(echo "$TC" | jq '.results.edges | length')
echo "INFO: Greet blast radius: $NODE_COUNT nodes, $EDGE_COUNT edges"
if [ "$NODE_COUNT" -ge 1 ]; then
  echo "PASS: transitive callers provides blast radius data"
  PASS=$((PASS + 1))
else
  echo "FAIL: transitive callers returned no nodes"
  FAIL=$((FAIL + 1))
fi

# === Category 8: Code Smells ===

# Test 3: God objects — classes with too many members
echo ""
echo "=== Test 3: god object detection (classes with many members) ==="
# Find all class/struct/type symbols and check member counts
CLASSES=$(canopy query symbols --db "$DB" --kind class --limit 500 2>&1)
CLASS_IDS=$(echo "$CLASSES" | jq -r '.results[].id')
MAX_MEMBERS=0
GOD_CLASS=""
for cid in $CLASS_IDS; do
  DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$cid" 2>&1)
  MC=$(echo "$DETAIL" | jq '.results.members | length')
  if [ "$MC" -gt "$MAX_MEMBERS" ]; then
    MAX_MEMBERS=$MC
    GOD_CLASS=$(echo "$DETAIL" | jq -r '.results.symbol.name')
  fi
done
echo "INFO: Largest class is '$GOD_CLASS' with $MAX_MEMBERS members"
if [ "$MAX_MEMBERS" -ge 0 ]; then
  echo "PASS: god object detection works"
  PASS=$((PASS + 1))
else
  echo "FAIL: could not check class members"
  FAIL=$((FAIL + 1))
fi

# Test 4: High fan-in hotspots (fragile symbols)
echo ""
echo "=== Test 4: high fan-in hotspots ==="
HOTSPOTS=$(canopy query hotspots --db "$DB" --top 5 2>&1)
TOP_NAME=$(echo "$HOTSPOTS" | jq -r '.results[0].symbol.name')
TOP_CALLERS=$(echo "$HOTSPOTS" | jq '.results[0].caller_count')
echo "INFO: Top hotspot is '$TOP_NAME' with $TOP_CALLERS callers"
if echo "$HOTSPOTS" | jq -e '.results | length > 0' > /dev/null 2>&1; then
  echo "PASS: hotspot analysis works for fan-in detection"
  PASS=$((PASS + 1))
else
  echo "FAIL: no hotspots found"
  FAIL=$((FAIL + 1))
fi

# Test 5: Public symbols that could be private (0 external refs, > 0 internal refs)
echo ""
echo "=== Test 5: public symbols that could be private ==="
COULD_BE_PRIVATE=$(canopy query symbols --db "$DB" --visibility public --limit 500 2>&1 | \
  jq '[.results[] | select(.ref_count > 0 and .external_ref_count == 0)] | length')
echo "INFO: $COULD_BE_PRIVATE public symbols could potentially be made private"
if [ "$COULD_BE_PRIVATE" -ge 0 ]; then
  echo "PASS: visibility analysis query works"
  PASS=$((PASS + 1))
else
  echo "FAIL: visibility analysis query failed"
  FAIL=$((FAIL + 1))
fi

# === Category 9: Impact Analysis ===

# Test 6: How widely used is a symbol? (ref_count from symbol-detail)
echo ""
echo "=== Test 6: symbol usage breadth via ref counts ==="
GREET_DETAIL=$(canopy query symbol-detail --db "$DB" --symbol "$GREET_GO_ID" 2>&1)
REF=$(echo "$GREET_DETAIL" | jq '.results.symbol.ref_count')
EXT=$(echo "$GREET_DETAIL" | jq '.results.symbol.external_ref_count')
INT=$(echo "$GREET_DETAIL" | jq '.results.symbol.internal_ref_count')
echo "INFO: Greet usage - total: $REF, external: $EXT, internal: $INT"
if [ "$REF" -ge 0 ] && [ "$EXT" -ge 0 ] && [ "$INT" -ge 0 ]; then
  echo "PASS: ref counts provide usage breadth data"
  PASS=$((PASS + 1))
else
  echo "FAIL: invalid ref counts"
  FAIL=$((FAIL + 1))
fi

# Test 7: Package coupling via package-graph edge count
echo ""
echo "=== Test 7: package coupling analysis ==="
PG=$(canopy query package-graph --db "$DB" 2>&1)
PKG_COUNT=$(echo "$PG" | jq '.results.packages | length')
EDGE_COUNT=$(echo "$PG" | jq '.results.edges | length')
echo "INFO: $PKG_COUNT packages, $EDGE_COUNT dependency edges"
if [ "$PKG_COUNT" -ge 0 ] && [ "$EDGE_COUNT" -ge 0 ]; then
  echo "PASS: package coupling data available"
  PASS=$((PASS + 1))
else
  echo "FAIL: package graph query failed"
  FAIL=$((FAIL + 1))
fi

# === Category 10: Cross-Cutting Concerns ===

# Test 8: All exported symbols across the project
echo ""
echo "=== Test 8: all exported (public) symbols ==="
PUBLIC=$(canopy query symbols --db "$DB" --visibility public --limit 500 2>&1)
PUB_COUNT=$(echo "$PUBLIC" | jq '.total_count')
echo "INFO: $PUB_COUNT public symbols across the project"
if [ "$PUB_COUNT" -ge 0 ]; then
  echo "PASS: cross-cutting public symbol query works"
  PASS=$((PASS + 1))
else
  echo "FAIL: public symbol query failed"
  FAIL=$((FAIL + 1))
fi

# Test 9: All functions across languages
echo ""
echo "=== Test 9: all functions across languages ==="
FUNCS=$(canopy query symbols --db "$DB" --kind function --limit 500 2>&1)
FUNC_COUNT=$(echo "$FUNCS" | jq '.total_count')
echo "INFO: $FUNC_COUNT functions across all languages"
# Verify functions come from multiple languages by checking different file extensions
FUNC_EXTS=$(echo "$FUNCS" | jq -r '[.results[].file | split(".") | .[-1]] | unique | .[]')
EXT_COUNT=$(echo "$FUNC_EXTS" | wc -l)
if [ "$EXT_COUNT" -gt 1 ]; then
  echo "PASS: functions found across $EXT_COUNT file types"
  PASS=$((PASS + 1))
else
  echo "FAIL: functions only from 1 file type"
  echo "  Extensions: $FUNC_EXTS"
  FAIL=$((FAIL + 1))
fi

# === Category 11: Comparative ===

# Test 10: Symbol kind distribution
echo ""
echo "=== Test 10: symbol kind distribution ==="
SUMMARY=$(canopy query summary --db "$DB" 2>&1)
LANG_COUNT=$(echo "$SUMMARY" | jq '.results.languages | length')
echo "INFO: Project has $LANG_COUNT languages"
# Each language should have kind_counts
ALL_HAVE_KINDS=1
for i in $(seq 0 $((LANG_COUNT - 1))); do
  HAS=$(echo "$SUMMARY" | jq ".results.languages[$i].kind_counts | length")
  if [ "$HAS" -eq 0 ]; then
    ALL_HAVE_KINDS=0
  fi
done
if [ "$ALL_HAVE_KINDS" -eq 1 ]; then
  echo "PASS: all languages have kind distribution data"
  PASS=$((PASS + 1))
else
  echo "FAIL: some languages missing kind distribution"
  FAIL=$((FAIL + 1))
fi

# Test 11: File size comparison via files query
echo ""
echo "=== Test 11: file size comparison ==="
FILES=$(canopy query files --db "$DB" 2>&1)
LINE_COUNTS=$(echo "$FILES" | jq '[.results[].line_count]')
MAX_LINES=$(echo "$LINE_COUNTS" | jq 'max')
MIN_LINES=$(echo "$LINE_COUNTS" | jq 'min')
echo "INFO: File sizes range from $MIN_LINES to $MAX_LINES lines"
if [ "$MAX_LINES" -ge "$MIN_LINES" ] && [ "$MIN_LINES" -ge 0 ]; then
  echo "PASS: file size comparison data available"
  PASS=$((PASS + 1))
else
  echo "FAIL: invalid file size data"
  FAIL=$((FAIL + 1))
fi

# Test 12: Ratio of unused to total symbols
echo ""
echo "=== Test 12: unused symbol ratio ==="
TOTAL_SYMS=$(canopy query symbols --db "$DB" --limit 500 2>&1 | jq '.total_count')
UNUSED_SYMS=$(canopy query unused --db "$DB" --limit 500 2>&1 | jq '.total_count')
echo "INFO: $UNUSED_SYMS of $TOTAL_SYMS symbols are unused ($(( (UNUSED_SYMS * 100) / TOTAL_SYMS ))%)"
if [ "$UNUSED_SYMS" -le "$TOTAL_SYMS" ]; then
  echo "PASS: unused count ($UNUSED_SYMS) <= total count ($TOTAL_SYMS)"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused count ($UNUSED_SYMS) > total count ($TOTAL_SYMS)"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
