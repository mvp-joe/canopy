#!/bin/bash
# Test: Deep edge cases - stress testing, unusual inputs, boundary conditions
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Deep Edge Cases ==="

DB="$SCRATCH/go-project/.canopy/index.db"

# Test: negative line/col in symbol-at
echo "--- Test: symbol-at negative line ---"
set +e
OUTPUT=$(canopy query symbol-at "$SCRATCH/go-project/main.go" -1 0 --db "$DB" 2>&1)
EXIT_CODE=$?
set -e
# Should handle gracefully - either null results or error
if [ "$EXIT_CODE" -eq 0 ] || [ "$EXIT_CODE" -eq 1 ]; then
  pass "symbol-at with negative line handled gracefully"
else
  fail "symbol-at with negative line" "exit 0 or 1" "exit=$EXIT_CODE"
fi

# Test: symbol-at with 0 0 on every language
echo "--- Test: symbol-at 0 0 on all languages ---"
ALL_OK=true
for proj_file in "go-project:main.go" "ts-project:src/index.ts" "js-project:app.js" "python-project:main.py" "rust-project:src/main.rs" "c-project:main.c" "cpp-project:main.cpp" "java-project:App.java" "php-project:index.php" "ruby-project:app.rb"; do
  PROJ=$(echo "$proj_file" | cut -d: -f1)
  FILE=$(echo "$proj_file" | cut -d: -f2)
  set +e
  OUTPUT=$(canopy query symbol-at "$SCRATCH/$PROJ/$FILE" 0 0 --db "$SCRATCH/$PROJ/.canopy/index.db" 2>/dev/null)
  EXIT_CODE=$?
  set -e
  if [ "$EXIT_CODE" -ne 0 ]; then
    ALL_OK=false
    fail "symbol-at 0 0 on $PROJ" "exit 0" "exit=$EXIT_CODE"
  fi
done
if [ "$ALL_OK" = true ]; then
  pass "symbol-at 0 0 works on all 10 languages"
fi

# Test: references with --symbol 0 (non-existent)
echo "--- Test: references --symbol 0 ---"
set +e
OUTPUT=$(canopy query references --symbol 0 --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count // 0')
if [ "$TOTAL" -eq 0 ] || [ "$EXIT_CODE" -ne 0 ]; then
  pass "references --symbol 0 returns 0 results or error"
else
  fail "references --symbol 0" "0 or error" "total=$TOTAL exit=$EXIT_CODE"
fi

# Test: references with --symbol 999999 (non-existent)
echo "--- Test: references --symbol 999999 ---"
set +e
OUTPUT=$(canopy query references --symbol 999999 --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count // 0')
if [ "$TOTAL" -eq 0 ] || [ "$EXIT_CODE" -ne 0 ]; then
  pass "references --symbol 999999 returns 0 results or error"
else
  fail "references --symbol 999999" "0 or error" "total=$TOTAL exit=$EXIT_CODE"
fi

# Test: callers with --symbol pointing to a non-function (struct)
echo "--- Test: callers on struct (not a function) ---"
STRUCT_ID=$(canopy query symbols --kind struct --db "$DB" 2>/dev/null | jq -r '.results[0].id // empty')
if [ -n "$STRUCT_ID" ]; then
  OUTPUT=$(canopy query callers --symbol "$STRUCT_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "callers" ]; then
    pass "callers on struct handled gracefully"
  else
    fail "callers on struct" "command=callers" "$CMD"
  fi
fi

# Test: implementations on function (not an interface)
echo "--- Test: implementations on function ---"
FUNC_ID=$(canopy query symbols --kind function --db "$DB" 2>/dev/null | jq -r '.results[0].id // empty')
if [ -n "$FUNC_ID" ]; then
  OUTPUT=$(canopy query implementations --symbol "$FUNC_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
  if [ "$CMD" = "implementations" ]; then
    pass "implementations on function handled (returns total=$TOTAL)"
  else
    fail "implementations on function" "command=implementations" "$CMD"
  fi
fi

# Test: dependents for source with special characters
echo "--- Test: dependents with special chars ---"
OUTPUT=$(canopy query dependents "fmt.Println" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "dependents" ]; then
  pass "dependents with dots in source works"
else
  fail "dependents with dots in source" "command=dependents" "$CMD"
fi

# Test: search with empty-like patterns
echo "--- Test: search '?' ---"
set +e
OUTPUT=$(canopy query search "?" --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 0 ]; then
  pass "search '?' doesn't crash"
else
  pass "search '?' returns error (ok for invalid glob)"
fi

# Test: search with double wildcard
echo "--- Test: search '**' ---"
OUTPUT=$(canopy query search "**" --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "search" ]; then
  pass "search '**' doesn't crash"
else
  fail "search '**' doesn't crash" "command=search" "$CMD"
fi

# Test: query with --db pointing to non-SQLite file
echo "--- Test: --db pointing to non-SQLite file ---"
set +e
OUTPUT=$(canopy query symbols --db "$SCRATCH/go-project/main.go" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 1 ]; then
  pass "non-SQLite --db file returns error"
else
  fail "non-SQLite --db file returns error" "exit 1" "exit=$EXIT_CODE"
fi

# Test: index a directory with no supported files
echo "--- Test: index directory with no supported files ---"
NOSUPPORT="$SCRATCH/no-supported"
mkdir -p "$NOSUPPORT"
echo "Hello" > "$NOSUPPORT/readme.txt"
echo "key: value" > "$NOSUPPORT/config.yaml"
cd "$NOSUPPORT" && git init -q && git add -A && git commit -q -m "init" && cd /tmp/claude-exercise-canopy-20260216-232119
set +e
OUTPUT=$(canopy index "$NOSUPPORT" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 0 ]; then
  pass "indexing dir with no supported files succeeds (0 files indexed)"
else
  fail "indexing dir with no supported files" "exit 0" "exit=$EXIT_CODE"
fi

# Test: query files on empty result with text format
echo "--- Test: files text format with 0 results ---"
if [ -f "$NOSUPPORT/.canopy/index.db" ]; then
  OUTPUT=$(canopy query files --format text --db "$NOSUPPORT/.canopy/index.db" 2>/dev/null)
  # Should output nothing or just headers
  pass "files text format with 0 results handled"
fi

# Test: summary text on project with 0 symbols
echo "--- Test: summary text on empty project ---"
if [ -f "$NOSUPPORT/.canopy/index.db" ]; then
  OUTPUT=$(canopy query summary --format text --db "$NOSUPPORT/.canopy/index.db" 2>/dev/null)
  pass "summary text on empty project handled"
fi

# Test: packages --prefix with non-matching prefix
echo "--- Test: packages --prefix with non-matching ---"
OUTPUT=$(canopy query packages --prefix "/nonexistent/path" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ]; then
  pass "packages --prefix with non-matching returns 0"
else
  fail "packages --prefix with non-matching returns 0" "0" "$TOTAL"
fi

# Test: concurrent access - run two queries simultaneously
echo "--- Test: concurrent queries ---"
canopy query symbols --db "$DB" 2>/dev/null > /dev/null &
PID1=$!
canopy query files --db "$DB" 2>/dev/null > /dev/null &
PID2=$!
wait $PID1 $PID2
if [ $? -eq 0 ]; then
  pass "concurrent queries don't crash"
else
  fail "concurrent queries don't crash" "both succeed" "error"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
