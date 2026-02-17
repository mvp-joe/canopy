#!/usr/bin/env bash
# Tests for new commands across multiple languages
# Exercises taxonomy categories 3, 6, 13 across Go, Java, TypeScript, Python, C, etc.
set -uo pipefail
DB=/tmp/exercise-test.db
PASS=0; FAIL=0

# Test 1: scope-at works for Python
echo "=== Test 1: scope-at in Python ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/util.py"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 1 4 2>&1)
CMD="canopy query scope-at --db $DB $FILE 1 4"
if echo "$OUTPUT" | jq -e '.command == "scope-at"' > /dev/null 2>&1; then
  echo "PASS: scope-at works for Python"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope-at failed for Python"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 2: scope-at works for Rust
echo ""
echo "=== Test 2: scope-at in Rust ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.rs"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 5 4 2>&1)
CMD="canopy query scope-at --db $DB $FILE 5 4"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: scope-at returns array for Rust"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope-at failed for Rust"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 3: scope-at works for C
echo ""
echo "=== Test 3: scope-at in C ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/main.c"
OUTPUT=$(canopy query scope-at --db "$DB" "$FILE" 15 4 2>&1)
CMD="canopy query scope-at --db $DB $FILE 15 4"
if echo "$OUTPUT" | jq -e '.results | type == "array"' > /dev/null 2>&1; then
  echo "PASS: scope-at returns array for C"
  PASS=$((PASS + 1))
else
  echo "FAIL: scope-at failed for C"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

# Test 4: symbol-detail for TypeScript function
echo ""
echo "=== Test 4: symbol-detail for TypeScript function ==="
TS_GREET_ID=$(canopy query search --db "$DB" "greet" 2>/dev/null | jq -r '.results[] | select(.file | endswith("util.ts")) | .id' | head -1)
if [ -n "$TS_GREET_ID" ]; then
  OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol "$TS_GREET_ID" 2>&1)
  CMD="canopy query symbol-detail --db $DB --symbol $TS_GREET_ID"
  if echo "$OUTPUT" | jq -e '.results.symbol.name == "greet"' > /dev/null 2>&1; then
    echo "PASS: TypeScript greet found via symbol-detail"
    PASS=$((PASS + 1))
  else
    echo "FAIL: TypeScript greet not found"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "FAIL: could not find TypeScript greet symbol ID"
  FAIL=$((FAIL + 1))
fi

# Test 5: symbol-detail for Python function
echo ""
echo "=== Test 5: symbol-detail for Python function ==="
PY_ADD_ID=$(canopy query search --db "$DB" "add" 2>/dev/null | jq -r '.results[] | select(.file | endswith("util.py")) | .id' | head -1)
if [ -n "$PY_ADD_ID" ]; then
  OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol "$PY_ADD_ID" 2>&1)
  CMD="canopy query symbol-detail --db $DB --symbol $PY_ADD_ID"
  if echo "$OUTPUT" | jq -e '.results.symbol.name == "add"' > /dev/null 2>&1; then
    echo "PASS: Python add found via symbol-detail"
    PASS=$((PASS + 1))
  else
    echo "FAIL: Python add not found"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "FAIL: could not find Python add symbol ID"
  FAIL=$((FAIL + 1))
fi

# Test 6: symbol-detail for C function
echo ""
echo "=== Test 6: symbol-detail for C function ==="
C_ADD_ID=$(canopy query search --db "$DB" "add" 2>/dev/null | jq -r '.results[] | select(.file | endswith("main.c")) | .id' | head -1)
if [ -n "$C_ADD_ID" ]; then
  OUTPUT=$(canopy query symbol-detail --db "$DB" --symbol "$C_ADD_ID" 2>&1)
  CMD="canopy query symbol-detail --db $DB --symbol $C_ADD_ID"
  if echo "$OUTPUT" | jq -e '.results.symbol.name == "add"' > /dev/null 2>&1; then
    echo "PASS: C add found via symbol-detail"
    PASS=$((PASS + 1))
  else
    echo "FAIL: C add not found"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
else
  echo "FAIL: could not find C add symbol ID"
  FAIL=$((FAIL + 1))
fi

# Test 7: transitive-callers works cross-language (C main -> add)
echo ""
echo "=== Test 7: transitive-callers for C add ==="
if [ -n "$C_ADD_ID" ]; then
  OUTPUT=$(canopy query transitive-callers --db "$DB" --symbol "$C_ADD_ID" 2>&1)
  CMD="canopy query transitive-callers --db $DB --symbol $C_ADD_ID"
  if echo "$OUTPUT" | jq -e '.results.nodes | length >= 1' > /dev/null 2>&1; then
    echo "PASS: transitive-callers works for C function"
    PASS=$((PASS + 1))
  else
    echo "FAIL: transitive-callers failed for C function"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 8: unused filters by language via path-prefix
echo ""
echo "=== Test 8: unused --path-prefix for JavaScript files ==="
OUTPUT=$(canopy query unused --db "$DB" --path-prefix "/home/joe/code/canopy/.exercise/scratch/multi-lang/main.js" 2>&1)
CMD="canopy query unused --db $DB --path-prefix .../main.js"
# All results should be from main.js
NON_JS=$(echo "$OUTPUT" | jq '[.results[] | select(.file | endswith("main.js") | not)] | length')
if [ "$NON_JS" = "0" ]; then
  echo "PASS: path-prefix filters to JS files only"
  PASS=$((PASS + 1))
else
  echo "FAIL: path-prefix returned non-JS files"
  echo "  Command: $CMD"
  echo "  Non-JS count: $NON_JS"
  FAIL=$((FAIL + 1))
fi

# Test 9: type-hierarchy for Java Main class
echo ""
echo "=== Test 9: type-hierarchy for Java Main class ==="
MAIN_JAVA_ID=$(canopy query search --db "$DB" "Main" 2>/dev/null | jq -r '.results[] | select((.file | endswith("Main.java")) and .kind == "class") | .id' | head -1)
if [ -n "$MAIN_JAVA_ID" ]; then
  OUTPUT=$(canopy query type-hierarchy --db "$DB" --symbol "$MAIN_JAVA_ID" 2>&1)
  CMD="canopy query type-hierarchy --db $DB --symbol $MAIN_JAVA_ID"
  if echo "$OUTPUT" | jq -e '.results.symbol.name == "Main"' > /dev/null 2>&1; then
    echo "PASS: type-hierarchy works for Java Main class"
    PASS=$((PASS + 1))
  else
    echo "FAIL: type-hierarchy failed for Java Main class"
    echo "  Command: $CMD"
    echo "  Output: $OUTPUT"
    FAIL=$((FAIL + 1))
  fi
fi

# Test 10: reexports for util.ts
echo ""
echo "=== Test 10: reexports for util.ts ==="
FILE="/home/joe/code/canopy/.exercise/scratch/multi-lang/util.ts"
OUTPUT=$(canopy query reexports --db "$DB" "$FILE" 2>&1)
CMD="canopy query reexports --db $DB $FILE"
if echo "$OUTPUT" | jq -e '.command == "reexports"' > /dev/null 2>&1; then
  echo "PASS: reexports works for util.ts"
  PASS=$((PASS + 1))
else
  echo "FAIL: reexports failed for util.ts"
  echo "  Command: $CMD"
  echo "  Output: $OUTPUT"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
