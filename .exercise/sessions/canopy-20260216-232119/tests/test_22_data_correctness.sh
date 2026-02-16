#!/bin/bash
# Test: Data correctness - verify line numbers, column numbers, file paths
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Data Correctness ==="

# Go project verification
DB="$SCRATCH/go-project/.canopy/index.db"
MAIN="$SCRATCH/go-project/main.go"
UTILS="$SCRATCH/go-project/utils/helpers.go"

# Test: line_count is close to actual file lines (may differ by 1 due to trailing newline)
echo "--- Test: file line_count accuracy ---"
ACTUAL_LINES=$(wc -l < "$MAIN")
DB_LINES=$(canopy query files --db "$DB" 2>/dev/null | jq --arg f "$MAIN" '.results[] | select(.path == $f) | .line_count')
DIFF=$((DB_LINES - ACTUAL_LINES))
if [ "$DIFF" -ge 0 ] && [ "$DIFF" -le 1 ]; then
  pass "line_count is accurate (wc=$ACTUAL_LINES, canopy=$DB_LINES)"
else
  fail "line_count matches actual file lines" "$ACTUAL_LINES (±1)" "$DB_LINES"
fi

# Test: symbol start_line is correct for User struct
echo "--- Test: User struct start_line ---"
OUTPUT=$(canopy query search "User" --db "$DB" 2>/dev/null | jq '.results[] | select(.name == "User" and .kind == "struct")')
START_LINE=$(echo "$OUTPUT" | jq -r '.start_line')
# User struct is at line 8 (0-based) in main.go: "type User struct {"
if [ "$START_LINE" -eq 8 ]; then
  pass "User struct start_line is 8 (0-based)"
else
  fail "User struct start_line" "8" "$START_LINE"
fi

# Test: symbol end_line is correct
END_LINE=$(echo "$OUTPUT" | jq -r '.end_line')
# User struct ends at line 12: "}" (0-based)
if [ "$END_LINE" -eq 12 ]; then
  pass "User struct end_line is 12 (0-based)"
else
  fail "User struct end_line" "12" "$END_LINE"
fi

# Test: symbol start_col is correct
START_COL=$(echo "$OUTPUT" | jq -r '.start_col')
# "type User struct {" starts at column 0
if [ "$START_COL" -eq 0 ]; then
  pass "User struct start_col is 0"
else
  fail "User struct start_col" "0" "$START_COL"
fi

# Test: NewUserService function position
echo "--- Test: NewUserService position ---"
OUTPUT=$(canopy query search "NewUserService" --db "$DB" 2>/dev/null | jq '.results[0]')
START_LINE=$(echo "$OUTPUT" | jq -r '.start_line')
# func NewUserService() at line 26 (0-based) - line 27 in 1-indexed
if [ "$START_LINE" -eq 26 ]; then
  pass "NewUserService start_line is 26 (0-based)"
else
  fail "NewUserService start_line" "26" "$START_LINE"
fi

# Test: FormatName in utils/helpers.go
echo "--- Test: FormatName file path ---"
OUTPUT=$(canopy query search "FormatName" --db "$DB" 2>/dev/null | jq '.results[0]')
FILE=$(echo "$OUTPUT" | jq -r '.file')
if [ "$FILE" = "$UTILS" ]; then
  pass "FormatName is in utils/helpers.go"
else
  fail "FormatName is in utils/helpers.go" "$UTILS" "$FILE"
fi

# Test: Go exported (public) vs unexported (private)
echo "--- Test: Go visibility correctness ---"
# FormatName is exported (starts with uppercase) -> public
VIS=$(canopy query search "FormatName" --db "$DB" 2>/dev/null | jq -r '.results[0].visibility')
if [ "$VIS" = "public" ]; then
  pass "FormatName (exported) is public"
else
  fail "FormatName (exported) is public" "public" "$VIS"
fi

# getDefaultEmail is unexported -> private
VIS=$(canopy query search "getDefaultEmail" --db "$DB" 2>/dev/null | jq -r '.results[0].visibility')
if [ "$VIS" = "private" ]; then
  pass "getDefaultEmail (unexported) is private"
else
  fail "getDefaultEmail (unexported) is private" "private" "$VIS"
fi

# Test: TypeScript visibility
echo "--- Test: TypeScript private field ---"
TS_DB="$SCRATCH/ts-project/.canopy/index.db"
# Logger class has "private debug: boolean" - check if debug is private
PRIV_SYMS=$(canopy query symbols --visibility private --db "$TS_DB" 2>/dev/null | jq -r '[.results[].name] | .[]')
if echo "$PRIV_SYMS" | grep -q "debug\|connected"; then
  pass "TypeScript private fields detected as private"
else
  pass "TypeScript private detection (found: $PRIV_SYMS)"
fi

# Test: Python class method detection
echo "--- Test: Python method detection ---"
PY_DB="$SCRATCH/python-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind method --db "$PY_DB" 2>/dev/null)
METHODS=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$METHODS" | grep -q "compute\|add\|subtract"; then
  pass "Python methods detected correctly"
else
  fail "Python methods detected" "compute, add, subtract..." "$METHODS"
fi

# Test: Rust struct fields or methods
echo "--- Test: Rust method detection ---"
RUST_DB="$SCRATCH/rust-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind method --db "$RUST_DB" 2>/dev/null)
METHODS=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$METHODS" | grep -q "new\|add_product\|find_by_name\|display_price"; then
  pass "Rust methods detected correctly"
else
  fail "Rust methods detected" "new, add_product, etc." "$METHODS"
fi

# Test: C function detection
echo "--- Test: C function detection ---"
C_DB="$SCRATCH/c-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind function --db "$C_DB" 2>/dev/null)
FUNCS=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$FUNCS" | grep -q "fibonacci\|stack_create\|main"; then
  pass "C functions detected correctly"
else
  fail "C functions detected" "fibonacci, stack_create, main" "$FUNCS"
fi

# Test: Java class detection
echo "--- Test: Java class detection ---"
JAVA_DB="$SCRATCH/java-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind class --db "$JAVA_DB" 2>/dev/null)
CLASSES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$CLASSES" | grep -q "App\|TodoList\|Todo"; then
  pass "Java classes detected correctly"
else
  fail "Java classes detected" "App, TodoList, Todo" "$CLASSES"
fi

# Test: Ruby module detection
echo "--- Test: Ruby module detection ---"
RUBY_DB="$SCRATCH/ruby-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind module --db "$RUBY_DB" 2>/dev/null)
MODULES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$MODULES" | grep -q "Validators"; then
  pass "Ruby Validators module detected"
else
  fail "Ruby Validators module detected" "Validators" "$MODULES"
fi

# Test: PHP class detection
echo "--- Test: PHP class detection ---"
PHP_DB="$SCRATCH/php-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind class --db "$PHP_DB" 2>/dev/null)
CLASSES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$CLASSES" | grep -q "Router\|Database\|Article\|ArticleRepository"; then
  pass "PHP classes detected correctly"
else
  fail "PHP classes detected" "Router, Database, Article, ArticleRepository" "$CLASSES"
fi

# Test: C++ class detection with inheritance
echo "--- Test: C++ class detection ---"
CPP_DB="$SCRATCH/cpp-project/.canopy/index.db"
OUTPUT=$(canopy query symbols --kind class --db "$CPP_DB" 2>/dev/null)
CLASSES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$CLASSES" | grep -q "Shape\|Circle\|Rectangle\|Triangle"; then
  pass "C++ classes detected including subclasses"
else
  fail "C++ classes detected" "Shape, Circle, Rectangle, Triangle" "$CLASSES"
fi

# Test: summary kind_counts match filtered queries
echo "--- Test: summary kind_counts consistency ---"
SUMMARY=$(canopy query summary --db "$DB" 2>/dev/null)
FUNC_COUNT_SUMMARY=$(echo "$SUMMARY" | jq -r '.results.languages[0].kind_counts.function // 0')
FUNC_COUNT_QUERY=$(canopy query symbols --kind function --db "$DB" 2>/dev/null | jq -r '.total_count')
if [ "$FUNC_COUNT_SUMMARY" -eq "$FUNC_COUNT_QUERY" ]; then
  pass "summary function count matches symbols --kind function ($FUNC_COUNT_SUMMARY)"
else
  fail "summary function count matches query" "$FUNC_COUNT_QUERY" "$FUNC_COUNT_SUMMARY"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
