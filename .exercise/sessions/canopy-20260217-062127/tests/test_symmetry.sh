#!/usr/bin/env bash
# Tests for symmetry: callers↔callees, deps↔dependents, ref_count decomposition
# Taxonomy categories: 4 (call graph), 5 (dependencies), 7 (refactoring)
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

get_id() {
  canopy query search --db "$DB" "$1" 2>/dev/null | jq -r ".results[] | select(.file | endswith(\"$2\")) | .id" | head -1
}

GREET_ID=$(get_id "Greet" "util.go")
ADD_ID=$(get_id "Add" "util.go")
MAIN_GO_ID=$(canopy query symbols --db "$DB" --kind function --file "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go" 2>/dev/null | jq -r '.results[] | select(.name == "main") | .id' | head -1)

# Test 1: callers/callees symmetry — if main calls Greet, then Greet's callers include main
echo "=== Test 1: callers/callees symmetry (main→Greet) ==="
CALLEES_OF_MAIN=$(canopy query callees --db "$DB" --symbol "$MAIN_GO_ID" 2>&1)
CALLERS_OF_GREET=$(canopy query callers --db "$DB" --symbol "$GREET_ID" 2>&1)
MAIN_CALLS_GREET=$(echo "$CALLEES_OF_MAIN" | jq --argjson gid "$GREET_ID" '[.results[] | select(.callee_id == $gid)] | length')
GREET_CALLED_BY_MAIN=$(echo "$CALLERS_OF_GREET" | jq --argjson mid "$MAIN_GO_ID" '[.results[] | select(.caller_id == $mid)] | length')
if [ "$MAIN_CALLS_GREET" -gt 0 ] && [ "$GREET_CALLED_BY_MAIN" -gt 0 ]; then
  echo "PASS: main→Greet appears in both callees(main) and callers(Greet)"
  PASS=$((PASS + 1))
elif [ "$MAIN_CALLS_GREET" -eq 0 ] && [ "$GREET_CALLED_BY_MAIN" -eq 0 ]; then
  echo "PASS: main does not call Greet (consistent in both directions)"
  PASS=$((PASS + 1))
else
  echo "FAIL: asymmetry: callees(main) has Greet=$MAIN_CALLS_GREET, callers(Greet) has main=$GREET_CALLED_BY_MAIN"
  FAIL=$((FAIL + 1))
fi

# Test 2: callers/callees symmetry — main→Add
echo ""
echo "=== Test 2: callers/callees symmetry (main→Add) ==="
MAIN_CALLS_ADD=$(echo "$CALLEES_OF_MAIN" | jq --argjson aid "$ADD_ID" '[.results[] | select(.callee_id == $aid)] | length')
CALLERS_OF_ADD=$(canopy query callers --db "$DB" --symbol "$ADD_ID" 2>&1)
ADD_CALLED_BY_MAIN=$(echo "$CALLERS_OF_ADD" | jq --argjson mid "$MAIN_GO_ID" '[.results[] | select(.caller_id == $mid)] | length')
if [ "$MAIN_CALLS_ADD" -gt 0 ] && [ "$ADD_CALLED_BY_MAIN" -gt 0 ]; then
  echo "PASS: main→Add appears in both callees(main) and callers(Add)"
  PASS=$((PASS + 1))
elif [ "$MAIN_CALLS_ADD" -eq 0 ] && [ "$ADD_CALLED_BY_MAIN" -eq 0 ]; then
  echo "PASS: main does not call Add (consistent)"
  PASS=$((PASS + 1))
else
  echo "FAIL: asymmetry: callees(main) has Add=$MAIN_CALLS_ADD, callers(Add) has main=$ADD_CALLED_BY_MAIN"
  FAIL=$((FAIL + 1))
fi

# Test 3: ref_count = external_ref_count + internal_ref_count for all symbols
echo ""
echo "=== Test 3: ref_count decomposition consistency ==="
ALL_SYMS=$(canopy query symbols --db "$DB" --limit 500 2>&1)
INCONSISTENT=$(echo "$ALL_SYMS" | jq '[.results[] | select(.ref_count != (.external_ref_count + .internal_ref_count))] | length')
if [ "$INCONSISTENT" = "0" ]; then
  echo "PASS: ref_count == external + internal for all symbols"
  PASS=$((PASS + 1))
else
  echo "FAIL: $INCONSISTENT symbols have ref_count != external + internal"
  echo "$ALL_SYMS" | jq '[.results[] | select(.ref_count != (.external_ref_count + .internal_ref_count)) | {name, ref_count, external_ref_count, internal_ref_count}]'
  FAIL=$((FAIL + 1))
fi

# Test 4: deps/dependents symmetry — if main.go imports from util.go source, dependents(source) includes main.go
echo ""
echo "=== Test 4: deps/dependents symmetry ==="
MAIN_GO_FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.go"
DEPS=$(canopy query deps --db "$DB" "$MAIN_GO_FILE" 2>&1)
DEP_SOURCES=$(echo "$DEPS" | jq -r '.results[].source' 2>/dev/null)
if [ -n "$DEP_SOURCES" ]; then
  FIRST_SOURCE=$(echo "$DEP_SOURCES" | head -1)
  DEPENDENTS=$(canopy query dependents --db "$DB" "$FIRST_SOURCE" 2>&1)
  # Check if main.go appears as a dependent of the source it imports
  HAS_MAIN=$(echo "$DEPENDENTS" | jq --arg f "$MAIN_GO_FILE" '[.results[] | select(.file_path == $f)] | length')
  if [ "$HAS_MAIN" -gt 0 ]; then
    echo "PASS: main.go found in dependents of '$FIRST_SOURCE'"
    PASS=$((PASS + 1))
  else
    echo "FAIL: main.go not found in dependents of '$FIRST_SOURCE'"
    echo "  Dependents: $(echo "$DEPENDENTS" | jq -r '.results[].file_path')"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: main.go has no deps, skipping symmetry check"
fi

# Test 5: symbols total_count matches actual results length (no pagination truncation at default limit)
echo ""
echo "=== Test 5: total_count matches results length ==="
SYMS=$(canopy query symbols --db "$DB" --limit 500 2>&1)
RESULT_LEN=$(echo "$SYMS" | jq '.results | length')
TOTAL=$(echo "$SYMS" | jq '.total_count')
if [ "$RESULT_LEN" = "$TOTAL" ]; then
  echo "PASS: results length ($RESULT_LEN) matches total_count ($TOTAL)"
  PASS=$((PASS + 1))
else
  echo "FAIL: results length ($RESULT_LEN) != total_count ($TOTAL)"
  FAIL=$((FAIL + 1))
fi

# Test 6: unused total_count matches results length
echo ""
echo "=== Test 6: unused total_count matches results length ==="
UNUSED=$(canopy query unused --db "$DB" --limit 500 2>&1)
RESULT_LEN=$(echo "$UNUSED" | jq '.results | length')
TOTAL=$(echo "$UNUSED" | jq '.total_count')
if [ "$RESULT_LEN" = "$TOTAL" ]; then
  echo "PASS: unused results length ($RESULT_LEN) matches total_count ($TOTAL)"
  PASS=$((PASS + 1))
else
  echo "FAIL: unused results length ($RESULT_LEN) != total_count ($TOTAL)"
  FAIL=$((FAIL + 1))
fi

# Test 7: hotspots total_count matches results length
echo ""
echo "=== Test 7: hotspots total_count matches results length ==="
HOTSPOTS=$(canopy query hotspots --db "$DB" --top 100 2>&1)
RESULT_LEN=$(echo "$HOTSPOTS" | jq '.results | length')
TOTAL=$(echo "$HOTSPOTS" | jq '.total_count')
if [ "$RESULT_LEN" = "$TOTAL" ]; then
  echo "PASS: hotspots results length ($RESULT_LEN) matches total_count ($TOTAL)"
  PASS=$((PASS + 1))
else
  echo "FAIL: hotspots results length ($RESULT_LEN) != total_count ($TOTAL)"
  FAIL=$((FAIL + 1))
fi

# Test 8: summary file_count matches files query
echo ""
echo "=== Test 8: summary file_count matches files query ==="
SUMMARY=$(canopy query summary --db "$DB" 2>&1)
SUMMARY_FILE_COUNT=$(echo "$SUMMARY" | jq '[.results.languages[].file_count] | add')
FILES=$(canopy query files --db "$DB" 2>&1)
FILES_COUNT=$(echo "$FILES" | jq '.total_count')
if [ "$SUMMARY_FILE_COUNT" = "$FILES_COUNT" ]; then
  echo "PASS: summary file_count ($SUMMARY_FILE_COUNT) matches files total ($FILES_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: summary file_count ($SUMMARY_FILE_COUNT) != files total ($FILES_COUNT)"
  FAIL=$((FAIL + 1))
fi

# Test 9: summary symbol_count matches symbols query
echo ""
echo "=== Test 9: summary symbol_count matches symbols query ==="
SUMMARY_SYM_COUNT=$(echo "$SUMMARY" | jq '[.results.languages[].symbol_count] | add')
ALL_SYM_COUNT=$(echo "$SYMS" | jq '.total_count')
if [ "$SUMMARY_SYM_COUNT" = "$ALL_SYM_COUNT" ]; then
  echo "PASS: summary symbol_count ($SUMMARY_SYM_COUNT) matches symbols total ($ALL_SYM_COUNT)"
  PASS=$((PASS + 1))
else
  echo "FAIL: summary symbol_count ($SUMMARY_SYM_COUNT) != symbols total ($ALL_SYM_COUNT)"
  FAIL=$((FAIL + 1))
fi

# Test 10: callers edge file/line matches actual source file position
echo ""
echo "=== Test 10: callers edge file exists in indexed files ==="
FIRST_EDGE_FILE=$(echo "$CALLERS_OF_GREET" | jq -r '.results[0].file' 2>/dev/null)
if [ -n "$FIRST_EDGE_FILE" ] && [ "$FIRST_EDGE_FILE" != "null" ]; then
  FILE_EXISTS=$(echo "$FILES" | jq --arg f "$FIRST_EDGE_FILE" '[.results[] | select(.path == $f)] | length')
  if [ "$FILE_EXISTS" -gt 0 ]; then
    echo "PASS: caller edge file ($FIRST_EDGE_FILE) exists in indexed files"
    PASS=$((PASS + 1))
  else
    echo "FAIL: caller edge file ($FIRST_EDGE_FILE) not found in indexed files"
    FAIL=$((FAIL + 1))
  fi
else
  echo "INFO: no caller edges to check"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
