#!/bin/bash
# Test: canopy query files - list indexed files
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query files ==="

# Test: JSON format - list all files for go-project
echo "--- Test: query files JSON output ---"
OUTPUT=$(canopy query files --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$CMD" = "files" ] && [ "$TOTAL" -gt 0 ]; then
  pass "query files returns JSON with command=files and total_count > 0"
else
  fail "query files returns JSON with command=files and total_count > 0" "command=files, total_count>0" "command=$CMD, total_count=$TOTAL"
fi

# Test: files have expected fields
echo "--- Test: file result fields ---"
FIRST=$(echo "$OUTPUT" | jq '.results[0]')
HAS_ID=$(echo "$FIRST" | jq 'has("id")')
HAS_PATH=$(echo "$FIRST" | jq 'has("path")')
HAS_LANG=$(echo "$FIRST" | jq 'has("language")')
HAS_LC=$(echo "$FIRST" | jq 'has("line_count")')
if [ "$HAS_ID" = "true" ] && [ "$HAS_PATH" = "true" ] && [ "$HAS_LANG" = "true" ] && [ "$HAS_LC" = "true" ]; then
  pass "File result has id, path, language, line_count fields"
else
  fail "File result has id, path, language, line_count fields" "all true" "id=$HAS_ID path=$HAS_PATH language=$HAS_LANG line_count=$HAS_LC"
fi

# Test: go-project files should have language "go"
echo "--- Test: go-project files have language go ---"
LANG=$(echo "$OUTPUT" | jq -r '.results[0].language')
if [ "$LANG" = "go" ]; then
  pass "go-project file has language 'go'"
else
  fail "go-project file has language 'go'" "go" "$LANG"
fi

# Test: text format
echo "--- Test: query files --format text ---"
OUTPUT_TEXT=$(canopy query files --format text --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
if echo "$OUTPUT_TEXT" | grep -q "\.go"; then
  pass "text format lists .go files"
else
  fail "text format lists .go files" "contains .go" "$OUTPUT_TEXT"
fi

# Test: --language filter
echo "--- Test: query files --language go ---"
OUTPUT=$(canopy query files --language go --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "query files --language go returns results"
else
  fail "query files --language go returns results" "total_count > 0" "total_count=$TOTAL"
fi

# Test: --language filter with non-existent language
echo "--- Test: query files --language python on go project ---"
OUTPUT=$(canopy query files --language python --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ]; then
  pass "query files --language python on go project returns 0 results"
else
  fail "query files --language python on go project returns 0 results" "total_count=0" "total_count=$TOTAL"
fi

# Test: --prefix filter
echo "--- Test: query files --prefix ---"
GOPATH=$(echo "$OUTPUT" | jq -r '.results[0].path // empty' 2>/dev/null || true)
OUTPUT=$(canopy query files --prefix "$SCRATCH/go-project/utils" --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -ge 1 ]; then
  pass "query files --prefix filters to utils directory"
else
  fail "query files --prefix filters to utils directory" "total_count >= 1" "total_count=$TOTAL"
fi

# Test: Each supported language project returns correct file language
echo "--- Test: all 10 languages indexed correctly ---"
ALL_LANGS_OK=true
for proj_lang in "go-project:go" "ts-project:typescript" "js-project:javascript" "python-project:python" "rust-project:rust" "c-project:c" "cpp-project:cpp" "java-project:java" "php-project:php" "ruby-project:ruby"; do
  PROJ=$(echo "$proj_lang" | cut -d: -f1)
  LANG=$(echo "$proj_lang" | cut -d: -f2)
  OUTPUT=$(canopy query files --db "$SCRATCH/$PROJ/.canopy/index.db" 2>/dev/null)
  FOUND_LANG=$(echo "$OUTPUT" | jq -r '.results[0].language // "none"')
  if [ "$FOUND_LANG" != "$LANG" ]; then
    ALL_LANGS_OK=false
    fail "$PROJ has language $LANG" "$LANG" "$FOUND_LANG"
  fi
done
if [ "$ALL_LANGS_OK" = true ]; then
  pass "All 10 projects have correct language in file results"
fi

# Test: pagination --limit
echo "--- Test: query files --limit 1 ---"
OUTPUT=$(canopy query files --limit 1 --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
RESULT_COUNT=$(echo "$OUTPUT" | jq '.results | length')
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$RESULT_COUNT" -eq 1 ] && [ "$TOTAL" -gt 1 ]; then
  pass "query files --limit 1 returns 1 result but total_count > 1"
else
  fail "query files --limit 1 returns 1 result but total_count > 1" "results=1, total>1" "results=$RESULT_COUNT, total=$TOTAL"
fi

# Test: pagination --offset
echo "--- Test: query files --offset ---"
OUTPUT_ALL=$(canopy query files --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
FIRST_ID=$(echo "$OUTPUT_ALL" | jq -r '.results[0].id')
OUTPUT_OFF=$(canopy query files --offset 1 --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
OFF_FIRST_ID=$(echo "$OUTPUT_OFF" | jq -r '.results[0].id')
if [ "$FIRST_ID" != "$OFF_FIRST_ID" ]; then
  pass "query files --offset 1 skips first result"
else
  fail "query files --offset 1 skips first result" "different first IDs" "both $FIRST_ID"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
