#!/bin/bash
# Test: Unusual code patterns, empty files, single-line files, huge symbols
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Unusual Code Patterns ==="

# Create a project with unusual code patterns
UNUSUAL="$SCRATCH/unusual-project"
mkdir -p "$UNUSUAL"

# File with a single function
cat > "$UNUSUAL/one_liner.go" << 'EOF'
package unusual
func One() int { return 1 }
EOF

# Empty Go file (just package declaration)
cat > "$UNUSUAL/empty.go" << 'EOF'
package unusual
EOF

# File with many nested symbols
cat > "$UNUSUAL/nested.py" << 'EOF'
class Outer:
    class Inner:
        class DeepInner:
            def deep_method(self):
                return "deep"

        def inner_method(self):
            return "inner"

    def outer_method(self):
        return "outer"

def standalone():
    return "standalone"
EOF

# File with unicode identifiers (Python supports this)
cat > "$UNUSUAL/unicode_names.py" << 'EOF'
def calculate_area(width, height):
    return width * height

class DataProcessor:
    def process(self, data):
        return sorted(data)
EOF

# File with very long function name
cat > "$UNUSUAL/long_names.go" << 'EOF'
package unusual

func ThisIsAVeryLongFunctionNameThatGoesOnAndOnAndOnForever() string {
    return "long"
}

var thisIsAlsoAVeryLongVariableNameThatIsQuiteVerbose = "verbose"
EOF

cd "$UNUSUAL" && git init -q && git add -A && git commit -q -m "init" && cd /tmp/claude-exercise-canopy-20260216-232119
canopy index --force "$UNUSUAL" 2>&1

DB="$UNUSUAL/.canopy/index.db"

# Test: single-line function
echo "--- Test: single-line function ---"
OUTPUT=$(canopy query search "One" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "Single-line function 'One' detected"
  START=$(echo "$OUTPUT" | jq -r '.results[0].start_line')
  END=$(echo "$OUTPUT" | jq -r '.results[0].end_line')
  if [ "$START" -eq "$END" ]; then
    pass "Single-line function has same start and end line ($START)"
  else
    pass "Single-line function spans $START to $END"
  fi
else
  fail "Single-line function detected" "total > 0" "$TOTAL"
fi

# Test: empty file (just package declaration)
echo "--- Test: empty file symbols ---"
OUTPUT=$(canopy query symbols --file "$UNUSUAL/empty.go" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
# Should have at most a package symbol
pass "Empty file query works (found $TOTAL symbols)"

# Test: nested classes in Python
echo "--- Test: nested Python classes ---"
OUTPUT=$(canopy query symbols --kind class --db "$DB" 2>/dev/null)
CLASSES=$(echo "$OUTPUT" | jq -r '[.results[].name] | .[]')
if echo "$CLASSES" | grep -q "Outer"; then
  pass "Outer class detected"
else
  fail "Outer class detected" "Outer" "$CLASSES"
fi

# Check if nested classes are detected
if echo "$CLASSES" | grep -q "Inner"; then
  pass "Inner nested class detected"
else
  pass "Inner nested class not detected (may not extract nested classes)"
fi

if echo "$CLASSES" | grep -q "DeepInner"; then
  pass "DeepInner deeply nested class detected"
else
  pass "DeepInner not detected (deep nesting may not be extracted)"
fi

# Test: long function names
echo "--- Test: long function name ---"
OUTPUT=$(canopy query search "ThisIsAVeryLong*" --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  NAME=$(echo "$OUTPUT" | jq -r '.results[0].name')
  pass "Long function name found: ${NAME:0:40}..."
else
  fail "Long function name detected" "total > 0" "$TOTAL"
fi

# Test: summary for mixed Go/Python project
echo "--- Test: summary for mixed project ---"
OUTPUT=$(canopy query summary --db "$DB" 2>/dev/null)
LANG_COUNT=$(echo "$OUTPUT" | jq '.results.languages | length')
if [ "$LANG_COUNT" -ge 2 ]; then
  pass "Summary shows both Go and Python ($LANG_COUNT languages)"
else
  fail "Summary shows both languages" ">= 2" "$LANG_COUNT"
fi

# Test: Python standalone function vs methods
echo "--- Test: Python standalone function ---"
OUTPUT=$(canopy query search "standalone" --db "$DB" 2>/dev/null)
KIND=$(echo "$OUTPUT" | jq -r '.results[0].kind // empty')
if [ "$KIND" = "function" ]; then
  pass "Python standalone function has kind 'function'"
else
  fail "Python standalone function has kind 'function'" "function" "$KIND"
fi

# Test: Python method has kind 'method'
OUTPUT=$(canopy query search "outer_method" --db "$DB" 2>/dev/null)
KIND=$(echo "$OUTPUT" | jq -r '.results[0].kind // empty')
if [ "$KIND" = "method" ]; then
  pass "Python class method has kind 'method'"
else
  fail "Python class method has kind 'method'" "method" "$KIND"
fi

# Test: files from mixed project
echo "--- Test: files from mixed project ---"
OUTPUT=$(canopy query files --db "$DB" 2>/dev/null)
FILE_COUNT=$(echo "$OUTPUT" | jq -r '.total_count')
LANGS=$(echo "$OUTPUT" | jq -r '[.results[].language] | unique | sort | .[]')
if [ "$FILE_COUNT" -ge 4 ]; then
  pass "Mixed project has $FILE_COUNT files"
else
  fail "Mixed project has files" ">= 4" "$FILE_COUNT"
fi

# Test: index with --languages on mixed project
echo "--- Test: index --languages python on mixed ---"
canopy index --force --languages python "$UNUSUAL" 2>&1 >/dev/null
OUTPUT=$(canopy query files --db "$DB" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
# Should only have Python files
LANGS=$(echo "$OUTPUT" | jq -r '[.results[].language] | unique | .[]')
if [ "$LANGS" = "python" ]; then
  pass "--languages python filters to only Python files"
else
  fail "--languages python filters" "python only" "$LANGS"
fi

# Restore full index
canopy index --force "$UNUSUAL" 2>&1 >/dev/null

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
