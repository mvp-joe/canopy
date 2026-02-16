#!/bin/bash
# Test: Special characters in paths, relative vs absolute
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Special Paths ==="

# Test: Create a project with spaces in directory name
echo "--- Test: directory with spaces ---"
SPACE_DIR="$SCRATCH/project with spaces"
mkdir -p "$SPACE_DIR"
cat > "$SPACE_DIR/hello.py" << 'EOF'
def greet(name):
    return f"Hello, {name}!"

class Greeter:
    def __init__(self, prefix="Hello"):
        self.prefix = prefix

    def greet(self, name):
        return f"{self.prefix}, {name}!"
EOF
cd "$SPACE_DIR" && git init -q && git add -A && git commit -q -m "init" && cd /tmp/claude-exercise-canopy-20260216-232119
canopy index "$SPACE_DIR" 2>&1
OUTPUT=$(canopy query symbols --db "$SPACE_DIR/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "Indexing and querying project with spaces in path works"
else
  fail "Indexing and querying project with spaces in path works" "total > 0" "total=$TOTAL"
fi

# Test: symbol-at with space in path
echo "--- Test: symbol-at with spaces in path ---"
OUTPUT=$(canopy query symbol-at "$SPACE_DIR/hello.py" 0 4 --db "$SPACE_DIR/.canopy/index.db" 2>/dev/null)
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
if [ "$NAME" = "greet" ]; then
  pass "symbol-at works with spaces in file path"
else
  fail "symbol-at works with spaces in file path" "greet" "$NAME"
fi

# Test: deps with space in path
echo "--- Test: deps with spaces in path ---"
OUTPUT=$(canopy query deps "$SPACE_DIR/hello.py" --db "$SPACE_DIR/.canopy/index.db" 2>/dev/null)
CMD=$(echo "$OUTPUT" | jq -r '.command')
if [ "$CMD" = "deps" ]; then
  pass "deps works with spaces in file path"
else
  fail "deps works with spaces in file path" "command=deps" "$CMD"
fi

# Test: query with relative file path (should work)
echo "--- Test: relative path in query ---"
cd "$SCRATCH/go-project"
OUTPUT=$(canopy query symbol-at "main.go" 8 5 2>/dev/null)
NAME=$(echo "$OUTPUT" | jq -r '.results.name // empty')
cd /tmp/claude-exercise-canopy-20260216-232119
if [ "$NAME" = "User" ]; then
  pass "symbol-at works with relative file path"
else
  fail "symbol-at works with relative file path" "User" "$NAME"
fi

# Test: query files with prefix filter using spaces
echo "--- Test: files --prefix with spaces ---"
OUTPUT=$(canopy query files --prefix "$SPACE_DIR" --db "$SPACE_DIR/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "files --prefix works with spaces in path"
else
  fail "files --prefix works with spaces in path" "total > 0" "total=$TOTAL"
fi

# Test: --db with absolute path
echo "--- Test: --db absolute path ---"
OUTPUT=$(canopy query symbols --db "$SCRATCH/go-project/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -gt 0 ]; then
  pass "--db with absolute path works"
else
  fail "--db with absolute path works" "total > 0" "total=$TOTAL"
fi

# Test: --db with relative path (from repo root)
echo "--- Test: --db relative path ---"
cd "$SCRATCH/go-project"
OUTPUT=$(canopy query symbols --db ".canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
cd /tmp/claude-exercise-canopy-20260216-232119
if [ "$TOTAL" -gt 0 ]; then
  pass "--db with relative path works"
else
  fail "--db with relative path works" "total > 0" "total=$TOTAL"
fi

# Test: index with current directory (no path argument)
echo "--- Test: index with no path (current dir) ---"
cd "$SCRATCH/go-project"
OUTPUT=$(canopy index --force 2>&1)
cd /tmp/claude-exercise-canopy-20260216-232119
if echo "$OUTPUT" | grep -q "Indexed"; then
  pass "index with no path argument works from repo directory"
else
  fail "index with no path argument works" "contains Indexed" "$OUTPUT"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
