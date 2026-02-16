#!/bin/bash
# Test: canopy index command - happy path and error cases
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy index ==="

# Test 1: Index a project (already done, but verify DB exists)
echo "--- Test: DB created after index ---"
if [ -f "$SCRATCH/go-project/.canopy/index.db" ]; then
  pass "DB file exists after indexing go-project"
else
  fail "DB file exists after indexing go-project" "file exists" "file missing"
fi

# Test 2: Index with --force (should succeed and recreate DB)
echo "--- Test: Index with --force ---"
OUTPUT=$(canopy index --force "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -q "Indexed"; then
  pass "Index with --force prints timing summary"
else
  fail "Index with --force prints timing summary" "output contains 'Indexed'" "$OUTPUT"
fi

# Test 3: Index with --languages filter
echo "--- Test: Index with --languages filter ---"
OUTPUT=$(canopy index --force --languages go "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -q "Indexed"; then
  pass "Index with --languages go succeeds"
else
  fail "Index with --languages go succeeds" "output contains 'Indexed'" "$OUTPUT"
fi

# Test 4: Index with --parallel
echo "--- Test: Index with --parallel ---"
OUTPUT=$(canopy index --force --parallel "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -q "Indexed"; then
  pass "Index with --parallel succeeds"
else
  fail "Index with --parallel succeeds" "output contains 'Indexed'" "$OUTPUT"
fi

# Test 5: Index prints database path to stderr
echo "--- Test: Index prints database path ---"
OUTPUT=$(canopy index --force "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -q "Database:"; then
  pass "Index prints 'Database:' path"
else
  fail "Index prints 'Database:' path" "output contains 'Database:'" "$OUTPUT"
fi

# Test 6: Index invalid path should fail with exit code 1
echo "--- Test: Index invalid path ---"
set +e
OUTPUT=$(canopy index "/nonexistent/path" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 1 ]; then
  pass "Index invalid path exits with code 1"
else
  fail "Index invalid path exits with code 1" "exit code 1" "exit code $EXIT_CODE"
fi

# Test 7: Index with custom --db path
echo "--- Test: Index with --db flag ---"
CUSTOM_DB="/tmp/claude-exercise-canopy-20260216-232119/scratch/custom-test.db"
rm -f "$CUSTOM_DB"
OUTPUT=$(canopy index --db "$CUSTOM_DB" --force "$SCRATCH/go-project" 2>&1)
if [ -f "$CUSTOM_DB" ]; then
  pass "Index with --db creates DB at custom path"
  rm -f "$CUSTOM_DB"
else
  fail "Index with --db creates DB at custom path" "file exists at $CUSTOM_DB" "file missing"
fi

# Test 8: Index with --languages filtering multiple languages
echo "--- Test: Index with multiple --languages ---"
OUTPUT=$(canopy index --force --languages go,python "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -q "Indexed"; then
  pass "Index with --languages go,python succeeds"
else
  fail "Index with --languages go,python succeeds" "success" "$OUTPUT"
fi

# Test 9: Re-index without --force (should still work, using incremental)
echo "--- Test: Re-index without --force ---"
OUTPUT=$(canopy index "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -q "Indexed\|already"; then
  pass "Re-index without --force succeeds"
else
  fail "Re-index without --force succeeds" "success output" "$OUTPUT"
fi

# Test 10: Index timing summary format
echo "--- Test: Timing summary format ---"
OUTPUT=$(canopy index --force "$SCRATCH/go-project" 2>&1)
if echo "$OUTPUT" | grep -qE "Indexed .* in .* \(extract: .*, resolve: .*\)"; then
  pass "Timing summary matches expected format"
else
  fail "Timing summary matches expected format" "Indexed <path> in <total> (extract: <time>, resolve: <time>)" "$OUTPUT"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
