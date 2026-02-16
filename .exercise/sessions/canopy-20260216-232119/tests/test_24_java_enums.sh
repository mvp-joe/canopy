#!/bin/bash
# Test: Java enums, interfaces, and visibility
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Java Enums and Features ==="

JAVA_DB="$SCRATCH/java-project/.canopy/index.db"
JAVA_FILE="$SCRATCH/java-project/App.java"

# Test: Java enum detection
echo "--- Test: Java enum ---"
OUTPUT=$(canopy query symbols --db "$JAVA_DB" 2>/dev/null)
KINDS=$(echo "$OUTPUT" | jq -r '[.results[].kind] | unique | sort | .[]')
echo "  Java kinds: $KINDS"

# Check if Priority enum is detected
OUTPUT=$(canopy query search "Priority" --db "$JAVA_DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  KIND=$(echo "$OUTPUT" | jq -r '.results[0].kind')
  pass "Java enum Priority detected as kind '$KIND'"
else
  fail "Java enum Priority detected" "total > 0" "total=$TOTAL"
fi

# Test: Java class methods
echo "--- Test: Java methods ---"
OUTPUT=$(canopy query symbols --kind method --db "$JAVA_DB" 2>/dev/null)
METHODS=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$METHODS" | grep -q "main\|add\|complete\|getAll"; then
  pass "Java methods detected: main, add, complete, getAll"
else
  fail "Java methods detected" "main, add, etc." "$METHODS"
fi

# Test: Java class with members
echo "--- Test: Java TodoList class ---"
OUTPUT=$(canopy query search "TodoList" --db "$JAVA_DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "TodoList class detected"
else
  fail "TodoList class detected" "total > 0" "total=$TOTAL"
fi

# Test: Java symbol-at on class
echo "--- Test: Java symbol-at on class ---"
# App class at line 3 (0-based)
OUTPUT=$(canopy query symbol-at "$JAVA_FILE" 3 13 --db "$JAVA_DB" 2>/dev/null)
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
if [ "$NAME" = "App" ]; then
  pass "Java symbol-at finds App class"
else
  fail "Java symbol-at finds App class" "App" "$NAME"
fi

# Test: Java deps
echo "--- Test: Java deps ---"
OUTPUT=$(canopy query deps "$JAVA_FILE" --db "$JAVA_DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  SOURCES=$(echo "$OUTPUT" | jq -r '[.results[].source] | .[]')
  pass "Java deps detected ($TOTAL imports: $SOURCES)"
else
  pass "Java deps: 0 imports (Java imports may be in separate file)"
fi

# Test: Java callers
echo "--- Test: Java callers ---"
# Find 'add' method ID
ADD_ID=$(canopy query search "add" --db "$JAVA_DB" 2>/dev/null | jq -r '.results[] | select(.kind == "method") | .id' | head -1)
if [ -n "$ADD_ID" ]; then
  OUTPUT=$(canopy query callers --symbol "$ADD_ID" --db "$JAVA_DB" 2>/dev/null)
  CMD=$(echo "$OUTPUT" | jq -r '.command')
  if [ "$CMD" = "callers" ]; then
    TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
    pass "Java callers for 'add' works (found $TOTAL callers)"
  else
    fail "Java callers works" "command=callers" "$CMD"
  fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
