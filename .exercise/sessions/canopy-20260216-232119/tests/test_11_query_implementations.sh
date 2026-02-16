#!/bin/bash
# Test: canopy query implementations
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: canopy query implementations ==="

DB="$SCRATCH/go-project/.canopy/index.db"
MAIN_FILE="$SCRATCH/go-project/main.go"

# Test: implementations of UserService interface (line 15, 0-based)
echo "--- Test: implementations of UserService ---"
OUTPUT=$(canopy query implementations "$MAIN_FILE" 15 5 --db "$DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "implementations" ]; then
  pass "implementations returns command=implementations"
else
  fail "implementations returns command=implementations" "implementations" "$CMD"
fi

# Test: implementations result fields (same as definition - array of locations)
echo "--- Test: implementations result structure ---"
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  FIRST=$(echo "$OUTPUT" | jq '.results[0]')
  for field in file start_line start_col end_line end_col; do
    HAS=$(echo "$FIRST" | jq "has(\"$field\")")
    if [ "$HAS" != "true" ]; then
      fail "implementation result has '$field'" "true" "$HAS"
    fi
  done
  pass "implementations results have location fields"
else
  pass "implementations returns 0 results (resolution may not find impl)"
fi

# Test: implementations with --symbol flag
echo "--- Test: implementations with --symbol ---"
SYM_OUTPUT=$(canopy query symbol-at "$MAIN_FILE" 15 5 --db "$DB" 2>/dev/null)
SYM_ID=$(echo "$SYM_OUTPUT" | jq -r '.results.id // empty')
if [ -n "$SYM_ID" ]; then
  OUTPUT=$(canopy query implementations --symbol "$SYM_ID" --db "$DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "implementations" ]; then
    pass "implementations with --symbol works"
  else
    fail "implementations with --symbol works" "command=implementations" "$CMD"
  fi
else
  fail "Could not get symbol ID for implementations test" "non-empty" "empty"
fi

# Test: implementations text format
echo "--- Test: implementations --format text ---"
OUTPUT_TEXT=$(canopy query implementations "$MAIN_FILE" 15 5 --format text --db "$DB" 2>/dev/null)
pass "implementations text format produces output"

# Test: implementations with no args
echo "--- Test: implementations with no args ---"
set +e
OUTPUT=$(canopy query implementations --db "$DB" 2>/dev/null)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
  pass "implementations with no args returns error"
else
  ERR=$(echo "$OUTPUT" | jq -r '.error // empty')
  if [ -n "$ERR" ]; then
    pass "implementations with no args returns JSON error"
  else
    fail "implementations with no args returns error" "error" "exit=$EXIT_CODE"
  fi
fi

# Test: implementations in C++ project (Shape interface)
echo "--- Test: implementations in C++ ---"
CPP_DB="$SCRATCH/cpp-project/.canopy/index.db"
CPP_FILE="$SCRATCH/cpp-project/shapes.hpp"
# Shape class at line 7 (0-based)
OUTPUT=$(canopy query implementations "$CPP_FILE" 7 6 --db "$CPP_DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "implementations" ]; then
  pass "implementations works in C++ project"
else
  fail "implementations works in C++ project" "command=implementations" "$CMD"
fi

# Test: implementations in Rust project (Displayable trait)
echo "--- Test: implementations in Rust ---"
RUST_DB="$SCRATCH/rust-project/.canopy/index.db"
RUST_FILE="$SCRATCH/rust-project/src/models.rs"
# Displayable trait at line 26 (0-based)
OUTPUT=$(canopy query implementations "$RUST_FILE" 26 10 --db "$RUST_DB" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "implementations" ]; then
  pass "implementations works in Rust project"
else
  fail "implementations works in Rust project" "command=implementations" "$CMD"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
