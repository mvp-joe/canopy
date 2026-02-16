#!/bin/bash
# Test: Advanced index options and language filtering
set -euo pipefail

SCRATCH="/tmp/claude-exercise-canopy-20260216-232119/scratch"
PASS=0
FAIL=0
ERRORS=""

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); ERRORS="${ERRORS}\n  FAIL: $1\n    Expected: $2\n    Actual:   $3"; echo "  FAIL: $1"; }

echo "=== Test Suite: Advanced Index Options ==="

# Create a mixed-language project
MIXED="$SCRATCH/mixed-project"
mkdir -p "$MIXED"
cat > "$MIXED/main.go" << 'EOF'
package main

func hello() string { return "hello" }
func main() { println(hello()) }
EOF
cat > "$MIXED/app.py" << 'EOF'
def greet():
    return "hello"

class App:
    def run(self):
        print(greet())
EOF
cat > "$MIXED/util.js" << 'EOF'
function helper() { return 42; }
module.exports = { helper };
EOF
cd "$MIXED" && git init -q && git add -A && git commit -q -m "init" && cd /tmp/claude-exercise-canopy-20260216-232119

# Test: Index all languages
echo "--- Test: index mixed project ---"
canopy index --force "$MIXED" 2>&1
OUTPUT=$(canopy query files --db "$MIXED/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 3 ]; then
  pass "Mixed project indexes all 3 files"
else
  fail "Mixed project indexes all 3 files" "3" "$TOTAL"
fi

# Test: --languages filter to only Go
echo "--- Test: index --languages go ---"
canopy index --force --languages go "$MIXED" 2>&1
OUTPUT=$(canopy query files --db "$MIXED/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
LANG=$(echo "$OUTPUT" | jq -r '.results[0].language // empty')
if [ "$TOTAL" -eq 1 ] && [ "$LANG" = "go" ]; then
  pass "--languages go indexes only the Go file"
else
  fail "--languages go indexes only the Go file" "1 go file" "total=$TOTAL lang=$LANG"
fi

# Test: --languages filter to Go and Python
echo "--- Test: index --languages go,python ---"
canopy index --force --languages go,python "$MIXED" 2>&1
OUTPUT=$(canopy query files --db "$MIXED/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
LANGS=$(echo "$OUTPUT" | jq -r '[.results[].language] | sort | .[]')
if [ "$TOTAL" -eq 2 ]; then
  pass "--languages go,python indexes 2 files"
else
  fail "--languages go,python indexes 2 files" "2" "total=$TOTAL langs=$LANGS"
fi

# Test: --languages with unknown language
echo "--- Test: index --languages nonexistent ---"
canopy index --force --languages nonexistent "$MIXED" 2>&1
OUTPUT=$(canopy query files --db "$MIXED/.canopy/index.db" 2>/dev/null)
TOTAL=$(echo "$OUTPUT" | jq -r '.total_count')
if [ "$TOTAL" -eq 0 ]; then
  pass "--languages nonexistent indexes 0 files"
else
  fail "--languages nonexistent indexes 0 files" "0" "$TOTAL"
fi

# Test: --force deletes and recreates DB
echo "--- Test: --force recreates DB ---"
canopy index --force --languages go "$MIXED" 2>&1
BEFORE=$(canopy query symbols --db "$MIXED/.canopy/index.db" 2>/dev/null | jq -r '.total_count')
canopy index --force "$MIXED" 2>&1
AFTER=$(canopy query symbols --db "$MIXED/.canopy/index.db" 2>/dev/null | jq -r '.total_count')
if [ "$AFTER" -gt "$BEFORE" ]; then
  pass "--force rebuilds with more files when language filter removed"
else
  fail "--force rebuilds" "after > before" "before=$BEFORE after=$AFTER"
fi

# Test: index --parallel produces same results as sequential
echo "--- Test: --parallel vs sequential ---"
canopy index --force "$MIXED" 2>&1
SEQ_TOTAL=$(canopy query symbols --db "$MIXED/.canopy/index.db" 2>/dev/null | jq -r '.total_count')
canopy index --force --parallel "$MIXED" 2>&1
PAR_TOTAL=$(canopy query symbols --db "$MIXED/.canopy/index.db" 2>/dev/null | jq -r '.total_count')
if [ "$SEQ_TOTAL" -eq "$PAR_TOTAL" ]; then
  pass "--parallel produces same symbol count as sequential ($SEQ_TOTAL)"
else
  fail "--parallel produces same symbol count as sequential" "seq=$SEQ_TOTAL" "par=$PAR_TOTAL"
fi

# Test: Re-index without --force should detect no changes
echo "--- Test: re-index detects no changes ---"
OUTPUT=$(canopy index "$MIXED" 2>&1)
# Should still succeed
if echo "$OUTPUT" | grep -q "Indexed"; then
  pass "re-index without --force succeeds"
else
  fail "re-index without --force succeeds" "Indexed" "$OUTPUT"
fi

# Test: summary reflects language filter
echo "--- Test: summary after --languages filter ---"
canopy index --force --languages python "$MIXED" 2>&1
OUTPUT=$(canopy query summary --db "$MIXED/.canopy/index.db" 2>/dev/null)
LANG_COUNT=$(echo "$OUTPUT" | jq '.results.languages | length')
LANG_NAME=$(echo "$OUTPUT" | jq -r '.results.languages[0].language // empty')
if [ "$LANG_COUNT" -eq 1 ] && [ "$LANG_NAME" = "python" ]; then
  pass "summary reflects --languages python filter"
else
  fail "summary reflects --languages python filter" "1 language: python" "count=$LANG_COUNT lang=$LANG_NAME"
fi

# Restore full index for other tests
canopy index --force "$MIXED" 2>&1

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ $FAIL -gt 0 ]; then
  echo -e "Failures:$ERRORS"
  exit 1
fi
