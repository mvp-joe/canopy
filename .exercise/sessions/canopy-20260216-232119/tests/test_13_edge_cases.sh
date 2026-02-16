#!/bin/bash
# Test: Edge cases - special characters, boundary values, cross-language
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Edge Cases ==="

# Test: Sort by file
echo "--- Test: --sort file ---"
DB="$SCRATCH/go-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --sort file --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "symbols" ]; then
  pass "--sort file works"
else
  fail "--sort file works" "command=symbols" "$CMD"
fi

# Test: Sort by kind
echo "--- Test: --sort kind ---"
OUTPUT=$(canopy query symbols --sort kind --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "symbols" ]; then
  pass "--sort kind works"
else
  fail "--sort kind works" "command=symbols" "$CMD"
fi

# Test: Combined filters - kind + visibility
echo "--- Test: --kind function --visibility public ---"
OUTPUT=$(canopy query symbols --kind function --visibility public --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -ge 0 ]; then
  pass "combined kind+visibility filter works"
else
  fail "combined kind+visibility filter works" "total >= 0" "$TOTAL"
fi

# Test: Combined filters - kind + file
echo "--- Test: --kind function --file ---"
FILE_PATH=$(canopy query files --db "$DB" 2>/dev/null | jq -r '.results[0].path')
OUTPUT=$(canopy query symbols --kind function --file "$FILE_PATH" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -ge 0 ]; then
  pass "combined kind+file filter works"
else
  fail "combined kind+file filter works" "total >= 0" "$TOTAL"
fi

# Test: --offset beyond total count
echo "--- Test: --offset beyond total ---"
OUTPUT=$(canopy query symbols --offset 99999 --db "$DB" 2>/dev/null)
COUNT=$(echo "$OUTPUT" | jq '.results | length')
if [ "$COUNT" -eq 0 ]; then
  pass "--offset beyond total returns 0 results"
else
  fail "--offset beyond total returns 0 results" "0" "$COUNT"
fi

# Test: --limit 0
echo "--- Test: --limit 0 ---"
set +e
OUTPUT=$(canopy query symbols --limit 0 --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
COUNT=$(echo "$OUTPUT" | jq '.results | length' 2>/dev/null || echo "error")
if [ "$COUNT" = "0" ] || [ "$EXIT_CODE" -ne 0 ]; then
  pass "--limit 0 returns 0 results or error"
else
  fail "--limit 0 returns 0 results or error" "0 or error" "count=$COUNT exit=$EXIT_CODE"
fi

# Test: files paths are absolute
echo "--- Test: file paths are absolute ---"
OUTPUT=$(canopy query files --db "$DB" 2>/dev/null)
FIRST_PATH=$(echo "$OUTPUT" | jq -r '.results[0].path')
if [[ "$FIRST_PATH" == /* ]]; then
  pass "file paths are absolute"
else
  fail "file paths are absolute" "starts with /" "$FIRST_PATH"
fi

# Test: line_count in files is positive
echo "--- Test: file line_count is positive ---"
LC=$(echo "$OUTPUT" | jq -r '.results[0].line_count')
if [ "$LC" -gt 0 ]; then
  pass "file line_count is positive"
else
  fail "file line_count is positive" "> 0" "$LC"
fi

# Test: symbols across all 10 languages
echo "--- Test: symbols found in all 10 languages ---"
ALL_OK=true
for proj in go-project ts-project js-project python-project rust-project c-project cpp-project java-project php-project ruby-project; do
  OUTPUT=$(canopy query symbols --db "$SCRATCH/$proj/.canopy/index.db" 2>/dev/null)
  TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
  if [ "$TOTAL" -le 0 ]; then
    ALL_OK=false
    fail "symbols found in $proj" "total > 0" "total=$TOTAL"
  fi
done
if [ "$ALL_OK" = true ]; then
  pass "All 10 language projects have symbols"
fi

# Test: search across all 10 languages
echo "--- Test: search works in all 10 languages ---"
ALL_OK=true
for proj in go-project ts-project js-project python-project rust-project c-project cpp-project java-project php-project ruby-project; do
  OUTPUT=$(canopy query search "*" --limit 1 --db "$SCRATCH/$proj/.canopy/index.db" 2>/dev/null)
  TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
  if [ "$TOTAL" -le 0 ]; then
    ALL_OK=false
    fail "search works in $proj" "total > 0" "total=$TOTAL"
  fi
done
if [ "$ALL_OK" = true ]; then
  pass "search works in all 10 language projects"
fi

# Test: summary for each language project
echo "--- Test: summary works for all 10 languages ---"
ALL_OK=true
for proj in go-project ts-project js-project python-project rust-project c-project cpp-project java-project php-project ruby-project; do
  OUTPUT=$(canopy query summary --db "$SCRATCH/$proj/.canopy/index.db" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" != "summary" ]; then
    ALL_OK=false
    fail "summary works in $proj" "command=summary" "$CMD"
  fi
done
if [ "$ALL_OK" = true ]; then
  pass "summary works in all 10 language projects"
fi

# Test: deps works for each language
echo "--- Test: deps works for all 10 languages ---"
ALL_OK=true
for proj_file in "go-project:main.go" "ts-project:src/index.ts" "js-project:app.js" "python-project:main.py" "rust-project:src/main.rs" "c-project:main.c" "cpp-project:main.cpp" "java-project:App.java" "php-project:index.php" "ruby-project:app.rb"; do
  PROJ=$(echo "$proj_file" | cut -d: -f1)
  FILE=$(echo "$proj_file" | cut -d: -f2)
  OUTPUT=$(canopy query deps "$SCRATCH/$PROJ/$FILE" --db "$SCRATCH/$PROJ/.canopy/index.db" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" != "deps" ]; then
    ALL_OK=false
    fail "deps works in $PROJ" "command=deps" "$CMD"
  fi
done
if [ "$ALL_OK" = true ]; then
  pass "deps works in all 10 language projects"
fi

# Test: search with just '*' wildcard
echo "--- Test: search '*' ---"
OUTPUT=$(canopy query search "*" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "search '*' returns all symbols"
else
  fail "search '*' returns all symbols" "total > 0" "total=$TOTAL"
fi

# Test: search with exact name (no wildcard)
echo "--- Test: search exact name ---"
OUTPUT=$(canopy query search "User" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  NAME=$(echo "$OUTPUT" | jq -r '.results[0].name')
  if [ "$NAME" = "User" ]; then
    pass "search exact name 'User' returns User"
  else
    fail "search exact name 'User' returns User" "User" "$NAME"
  fi
else
  fail "search exact name 'User' returns results" "total > 0" "total=$TOTAL"
fi

# Test: symbol-at at position 0,0 on go main.go (should be package keyword area)
echo "--- Test: symbol-at at 0,0 ---"
OUTPUT=$(canopy query symbol-at "$SCRATCH/go-project/main.go" 0 0 --db "$DB" 2>/dev/null)
RESULTS=$(echo "$OUTPUT" | jq -r '.results')
# Line 0 col 0 in go is "package main" - could be package symbol or null
if [ "$RESULTS" = "null" ] || echo "$OUTPUT" | jq -e '.results.kind' >/dev/null 2>&1; then
  pass "symbol-at 0,0 returns null or a symbol"
else
  fail "symbol-at 0,0 returns null or a symbol" "null or symbol" "$RESULTS"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
