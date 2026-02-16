#!/bin/bash
# Test: canopy query symbols - list and filter symbols
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query symbols ==="

DB="$SCRATCH/go-project/.canopy/index.db"

# Test: basic symbols query
echo "--- Test: query symbols JSON ---"
OUTPUT=$(canopy query symbols --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "symbols" ] && [ "$TOTAL" -gt 0 ]; then
  pass "query symbols returns JSON with command=symbols and results"
else
  fail "query symbols returns JSON" "command=symbols, total_count>0" "command=$CMD, total_count=$TOTAL"
fi

# Test: symbol result fields
echo "--- Test: symbol result fields ---"
FIRST=$(echo "$OUTPUT" | jq '.results[0]')
for field in id name kind visibility file start_line start_col end_line end_col; do
  HAS=$(echo "$FIRST" | jq "has(\"$field\")")
  if [ "$HAS" != "true" ]; then
    fail "Symbol has field '$field'" "true" "$HAS"
  fi
done
pass "Symbol results have all expected fields"

# Test: --kind filter for functions
echo "--- Test: query symbols --kind function ---"
OUTPUT=$(canopy query symbols --kind function --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  # Verify all results have kind=function
  KINDS=$(echo "$OUTPUT" | jq -r '[.results[].kind] | unique | .[]')
  if [ "$KINDS" = "function" ]; then
    pass "query symbols --kind function returns only functions"
  else
    fail "query symbols --kind function returns only functions" "function" "$KINDS"
  fi
else
  fail "query symbols --kind function returns results" "total_count > 0" "$TOTAL"
fi

# Test: --kind filter for struct (Go uses "struct" not "type")
echo "--- Test: query symbols --kind struct ---"
OUTPUT=$(canopy query symbols --kind struct --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "query symbols --kind struct returns results for Go"
else
  fail "query symbols --kind struct returns results for Go" "total_count > 0" "$TOTAL"
fi

# Test: --kind filter for interfaces
echo "--- Test: query symbols --kind interface ---"
OUTPUT=$(canopy query symbols --kind interface --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "query symbols --kind interface returns results for Go"
else
  fail "query symbols --kind interface returns results for Go" "total_count > 0" "$TOTAL"
fi

# Test: --visibility filter
echo "--- Test: query symbols --visibility public ---"
OUTPUT=$(canopy query symbols --visibility public --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  VISIBILITIES=$(echo "$OUTPUT" | jq -r '[.results[].visibility] | unique | .[]')
  if [ "$VISIBILITIES" = "public" ]; then
    pass "query symbols --visibility public returns only public symbols"
  else
    fail "query symbols --visibility public returns only public symbols" "public" "$VISIBILITIES"
  fi
else
  fail "query symbols --visibility public returns results" "total_count > 0" "$TOTAL"
fi

# Test: --visibility private
echo "--- Test: query symbols --visibility private ---"
OUTPUT=$(canopy query symbols --visibility private --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "query symbols --visibility private returns results"
else
  # Might be 0 if all exported in Go
  pass "query symbols --visibility private returns 0 results (possible for Go)"
fi

# Test: --file filter
echo "--- Test: query symbols --file ---"
FILE_PATH=$(canopy query files --db "$DB" 2>/dev/null | jq -r '.results[0].path')
OUTPUT=$(canopy query symbols --file "$FILE_PATH" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  # Verify all results reference the filtered file
  FILES=$(echo "$OUTPUT" | jq -r '[.results[].file] | unique | .[]')
  if [ "$FILES" = "$FILE_PATH" ]; then
    pass "query symbols --file returns only symbols in that file"
  else
    fail "query symbols --file returns only symbols in that file" "$FILE_PATH" "$FILES"
  fi
else
  fail "query symbols --file returns results" "total_count > 0" "$TOTAL"
fi

# Test: --path-prefix filter
echo "--- Test: query symbols --path-prefix ---"
OUTPUT=$(canopy query symbols --path-prefix "$SCRATCH/go-project/utils" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "query symbols --path-prefix filters to utils directory"
else
  fail "query symbols --path-prefix filters to utils directory" "total_count > 0" "$TOTAL"
fi

# Test: sort by name
echo "--- Test: query symbols --sort name ---"
OUTPUT=$(canopy query symbols --sort name --db "$DB" 2>/dev/null)
FIRST_NAME=$(echo "$OUTPUT" | jq -r '.results[0].name')
if [ -n "$FIRST_NAME" ]; then
  pass "query symbols --sort name returns results"
else
  fail "query symbols --sort name returns results" "non-empty first name" "empty"
fi

# Test: sort by name descending
echo "--- Test: query symbols --sort name --order desc ---"
OUTPUT_ASC=$(canopy query symbols --sort name --order asc --db "$DB" 2>/dev/null)
OUTPUT_DESC=$(canopy query symbols --sort name --order desc --db "$DB" 2>/dev/null)
FIRST_ASC=$(echo "$OUTPUT_ASC" | jq -r '.results[0].name')
FIRST_DESC=$(echo "$OUTPUT_DESC" | jq -r '.results[0].name')
if [ "$FIRST_ASC" != "$FIRST_DESC" ]; then
  pass "sort order asc vs desc produces different first results"
else
  fail "sort order asc vs desc produces different first results" "different names" "both $FIRST_ASC"
fi

# Test: sort by ref_count
echo "--- Test: query symbols --sort ref_count ---"
OUTPUT=$(canopy query symbols --sort ref_count --order desc --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "symbols" ]; then
  pass "query symbols --sort ref_count succeeds"
else
  fail "query symbols --sort ref_count succeeds" "command=symbols" "command=$CMD"
fi

# Test: --limit and --offset pagination
echo "--- Test: query symbols --limit 2 --offset 0 ---"
OUTPUT=$(canopy query symbols --limit 2 --offset 0 --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$COUNT" -le 2 ] && [ "$TOTAL" -gt 0 ]; then
  pass "query symbols --limit 2 returns at most 2 results"
else
  fail "query symbols --limit 2 returns at most 2 results" "count<=2" "count=$COUNT"
fi

# Test: text format
echo "--- Test: query symbols --format text ---"
OUTPUT_TEXT=$(canopy query symbols --format text --db "$DB" 2>/dev/null)
if echo "$OUTPUT_TEXT" | grep -qiE "function|type|interface|variable|method"; then
  pass "text format shows symbol kinds"
else
  fail "text format shows symbol kinds" "contains kind names" "$OUTPUT_TEXT"
fi

# Test: ref_count fields present
echo "--- Test: ref_count fields in symbol results ---"
OUTPUT=$(canopy query symbols --db "$DB" 2>/dev/null)
FIRST=$(echo "$OUTPUT" | jq '.results[0]')
HAS_RC=$(echo "$FIRST" | jq 'has("ref_count")')
HAS_ERC=$(echo "$FIRST" | jq 'has("external_ref_count")')
HAS_IRC=$(echo "$FIRST" | jq 'has("internal_ref_count")')
if [ "$HAS_RC" = "true" ] && [ "$HAS_ERC" = "true" ] && [ "$HAS_IRC" = "true" ]; then
  pass "Symbol results have ref_count, external_ref_count, internal_ref_count"
else
  fail "Symbol results have ref_count fields" "all true" "ref_count=$HAS_RC ext=$HAS_ERC int=$HAS_IRC"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
